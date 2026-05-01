package grpc

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.uber.org/zap"
)

const (
	etcdLeaseTTLSeconds   = 3
	etcdKeepAliveFailMax  = 3
	etcdReRegisterMinWait = time.Second
	etcdReRegisterMaxWait = 30 * time.Second
)

// etcdRegistration 持有一次 etcd 服务注册的运行态(lease + endpoint + keepalive 通道)。
type etcdRegistration struct {
	lease        clientv3.Lease
	leaseID      clientv3.LeaseID
	leaseChannel <-chan *clientv3.LeaseKeepAliveResponse
	em           endpoints.Manager
	endpointKey  string
}

// release 释放本次注册占用的 etcd 资源(删除 endpoint + 关闭 lease)。
func (r *etcdRegistration) release() {
	delCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := r.em.DeleteEndpoint(delCtx, r.endpointKey); err != nil {
		logs.Warn("etcd 删除 endpoint 失败", zap.String("endpoint", r.endpointKey), zap.Error(err))
	}
	if err := r.lease.Close(); err != nil {
		logs.Warn("etcd 关闭 lease 失败", zap.Int64("lease_id", int64(r.leaseID)), zap.Error(err))
	}
}

// etcdRegisterService 完成首次服务注册并启动单 goroutine 保活。
// 后续断连重连不会衍生新的 goroutine,而是在同一个 goroutine 内替换运行态。
func etcdRegisterService(ctx context.Context, servicePrefix, serviceName, addr string, client *clientv3.Client) error {
	reg, err := newEtcdRegistration(ctx, servicePrefix, serviceName, addr, client)
	if err != nil {
		return err
	}
	go runEtcdKeepAlive(ctx, reg, servicePrefix, serviceName, addr, client)
	return nil
}

// newEtcdRegistration 申请 lease 并把 endpoint 注册到 etcd,返回运行态。
func newEtcdRegistration(ctx context.Context, servicePrefix, serviceName, addr string, client *clientv3.Client) (*etcdRegistration, error) {
	lease := clientv3.NewLease(client)
	grantCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	leaseResp, err := lease.Grant(grantCtx, etcdLeaseTTLSeconds)
	if err != nil {
		_ = lease.Close()
		return nil, fmt.Errorf("申请 etcd lease 出错: %w", err)
	}

	leaseChannel, err := lease.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		_ = lease.Close()
		return nil, fmt.Errorf("启动 etcd lease KeepAlive 出错: %w", err)
	}

	em, err := endpoints.NewManager(client, servicePrefix)
	if err != nil {
		_ = lease.Close()
		return nil, fmt.Errorf("创建 etcd endpoint manager 出错: %w", err)
	}

	addCtx, addCancel := context.WithTimeout(ctx, time.Second*3)
	defer addCancel()
	endpointKey := fmt.Sprintf("%s/%s/%s", servicePrefix, serviceName, common.GenID())
	if err := em.AddEndpoint(addCtx, endpointKey, endpoints.Endpoint{Addr: addr}, clientv3.WithLease(leaseResp.ID)); err != nil {
		_ = lease.Close()
		return nil, fmt.Errorf("注册 etcd endpoint %s 出错: %w", endpointKey, err)
	}

	return &etcdRegistration{
		lease:        lease,
		leaseID:      leaseResp.ID,
		leaseChannel: leaseChannel,
		em:           em,
		endpointKey:  endpointKey,
	}, nil
}

// runEtcdKeepAlive 是单 goroutine 状态机:监听 lease 通道,断连阈值后原地重新注册并替换运行态。
// 整个保活生命周期内只会有一个 goroutine,不会因重连衍生新协程。
func runEtcdKeepAlive(parentCtx context.Context, reg *etcdRegistration, servicePrefix, serviceName, addr string, client *clientv3.Client) {
	failed := 0
	for {
		select {
		case <-parentCtx.Done():
			reg.release()
			return
		case resp := <-reg.leaseChannel:
			if resp != nil {
				failed = 0
				continue
			}
			failed++
			logs.Warn("etcd keep alive 失败", zap.String("service", serviceName), zap.Int("failed_count", failed))
			if failed < etcdKeepAliveFailMax {
				// 使用 jitter 避免多副本同步重连时同时打到 etcd
				if !sleepWithCtx(parentCtx, jitter(etcdReRegisterMinWait)) {
					reg.release()
					return
				}
				continue
			}
			reg.release()
			newReg, err := reRegisterWithBackoff(parentCtx, servicePrefix, serviceName, addr, client)
			if err != nil {
				return
			}
			reg = newReg
			failed = 0
		}
	}
}

// reRegisterWithBackoff 在 parentCtx 取消前持续重试注册,使用指数退避 + 抖动避免雪崩。
func reRegisterWithBackoff(parentCtx context.Context, servicePrefix, serviceName, addr string, client *clientv3.Client) (*etcdRegistration, error) {
	backoff := etcdReRegisterMinWait
	for {
		if err := parentCtx.Err(); err != nil {
			return nil, err
		}
		reg, err := newEtcdRegistration(parentCtx, servicePrefix, serviceName, addr, client)
		if err == nil {
			logs.Info("etcd 重新注册成功", zap.String("service", serviceName), zap.String("endpoint", reg.endpointKey))
			return reg, nil
		}
		logs.Warn("etcd 重新注册失败", zap.String("service", serviceName), zap.Error(err))
		if !sleepWithCtx(parentCtx, jitter(backoff)) {
			return nil, parentCtx.Err()
		}
		backoff *= 2
		if backoff > etcdReRegisterMaxWait {
			backoff = etcdReRegisterMaxWait
		}
	}
}

// jitter 在 [d, d*1.5] 之间随机化时长,避免多副本同步抖动打爆 etcd。
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return d + time.Duration(rand.Int63n(int64(d)/2+1))
}

// sleepWithCtx 在 ctx 取消时立即返回 false,避免 time.Sleep 无视取消信号。
func sleepWithCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
