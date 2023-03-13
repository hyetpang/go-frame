package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HyetPang/go-frame/pkgs/common"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
)

// grpc 使用etcd心跳保活
func etcdKeepLive(ctx context.Context, leaseChannel <-chan *clientv3.LeaseKeepAliveResponse, cleanFunc func(), servicePrefix, serviceName, addr string, client *clientv3.Client) {
	go func() {
		failedCount := 0
		for {
			select {
			case resp := <-leaseChannel:
				if resp != nil {
					// log.Println("keep alive success.")
				} else {
					log.Println("keep alive failed.")
					failedCount++
					for failedCount > 3 {
						cleanFunc()
						if err := etcdRegisterService(ctx, servicePrefix, serviceName, addr, client); err != nil {
							time.Sleep(time.Second)
							continue
						}
						return
					}
					continue
				}
			case <-ctx.Done():
				cleanFunc()
				return
			}
		}
	}()
}

// 服务注册
func etcdRegisterService(ctx context.Context, servicePrefix, serviceName, addr string, client *clientv3.Client) error {
	// 创建一个租约
	lease := clientv3.NewLease(client)
	cancelCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	leaseResp, err := lease.Grant(cancelCtx, 3)
	if err != nil {
		return err
	}

	leaseChannel, err := lease.KeepAlive(ctx, leaseResp.ID) // 长链接, 不用设置超时时间
	if err != nil {
		return err
	}

	em, err := endpoints.NewManager(client, servicePrefix)
	if err != nil {
		return err
	}

	cancelCtx, cancel = context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	if err := em.AddEndpoint(cancelCtx, fmt.Sprintf("%s/%s/%s", servicePrefix, serviceName, common.GenNanoIdString()), endpoints.Endpoint{
		Addr: addr,
	}, clientv3.WithLease(leaseResp.ID)); err != nil {
		return err
	}
	del := func() {
		cancelCtx, cancel = context.WithTimeout(ctx, time.Second*3)
		defer cancel()
		em.DeleteEndpoint(cancelCtx, serviceName)
		lease.Close()
	}
	// 保持注册状态(连接断开重连)
	etcdKeepLive(ctx, leaseChannel, del, servicePrefix, serviceName, addr, client)
	return nil
}
