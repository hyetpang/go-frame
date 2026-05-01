package lognotice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	// senderHTTPTimeout 限定单次 webhook 调用全程超时(连接 + 请求 + 响应),
	// 避免上游卡死拖慢 lognotice 的 watch goroutine。
	senderHTTPTimeout = 5 * time.Second
	// senderResponseLimit 限制响应体读取上限,防止上游异常巨型响应耗尽内存。
	senderResponseLimit = 64 * 1024
)

// senderBase 封装 webhook 调用所需的 *http.Client。
//
// 职责:
//  1. 用进程内独立的 http.Client 替代 gout 的全局 SetTimeout,避免对业务侧
//     全局 gout 状态的副作用(三个 sender 此前每次发送都会改写包级 timeout)。
//  2. 在拨号阶段(safeDialer)复检解析到的 IP,堵上"启动期 validateWebhookURL
//     检查通过、运行期 DNS rebinding 仍可打到内网"的 TOCTOU 攻击面。
//
// 三个 sender 共享同一个 senderBase,只在 payload 构造与响应解析上不同。
type senderBase struct {
	httpClient *http.Client
}

func newSenderBase(allowedHosts []string) *senderBase {
	dialer := &net.Dialer{
		Timeout:   senderHTTPTimeout,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		DialContext:           safeDialContext(dialer, allowedHosts),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   senderHTTPTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &senderBase{
		httpClient: &http.Client{
			Timeout:   senderHTTPTimeout,
			Transport: transport,
		},
	}
}

// postJSON 发送 JSON POST,把响应反序列化到 resp(若非 nil)。
// 失败原因可能是网络错误、TOCTOU 复检拒拨、HTTP 非 2xx 或响应体解析失败。
func (b *senderBase) postJSON(ctx context.Context, url string, payload, resp any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("payload 序列化失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	limited := io.LimitReader(httpResp.Body, senderResponseLimit)
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		// 把首段 body 读入错误,便于排查 webhook 接口返回的具体错误描述
		raw, _ := io.ReadAll(limited)
		return fmt.Errorf("webhook 返回非 2xx: status=%d body=%s", httpResp.StatusCode, string(raw))
	}
	if resp == nil {
		_, _ = io.Copy(io.Discard, limited)
		return nil
	}
	if err := json.NewDecoder(limited).Decode(resp); err != nil {
		return fmt.Errorf("响应反序列化失败: %w", err)
	}
	return nil
}

// safeDialContext 包装 net.Dialer.DialContext,在每次拨号前复检解析到的 IP。
//
// 为什么需要在拨号期再查一遍:
//   - 启动期 validateWebhookURL 已经做了 LookupIP + 私网黑名单,
//     但这是 TOCTOU(Time-Of-Check vs Time-Of-Use):DNS 结果在运行期仍可能变更
//     (短 TTL、被劫持、或 DNS rebinding 攻击)。
//   - 实际 HTTP 请求是另一次解析,如果只在启动期校验,理论上仍可能被引到
//     169.254.169.254 等元数据服务。
//
// 白名单 host(matchHostInAllowList)允许指向私网,与 validateWebhookURL 行为一致,
// 避免开发环境内网域名被误拦。
func safeDialContext(dialer *net.Dialer, allowedHosts []string) func(context.Context, string, string) (net.Conn, error) {
	allowed := normalizeAllowedHosts(allowedHosts)
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("拨号地址解析失败 %s: %w", addr, err)
		}
		// 白名单 host 跳过 IP 黑名单复检(允许指向开发内网),与 validateWebhookURL 对齐。
		if matchHostInAllowList(host, allowed) {
			return dialer.DialContext(ctx, network, addr)
		}
		// 直接给 IP 字面量:就地校验
		if ip := net.ParseIP(host); ip != nil {
			if isPrivateOrLoopback(ip) {
				return nil, fmt.Errorf("拒绝拨号到私网/回环 IP %s", host)
			}
			return dialer.DialContext(ctx, network, addr)
		}
		// 主机名:解析,任一 IP 命中私网即拒拨
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("拨号期 DNS 解析失败 %s: %w", host, err)
		}
		for _, ip := range ips {
			if isPrivateOrLoopback(ip.IP) {
				return nil, fmt.Errorf("拨号期检测到 host %s 解析到私网/回环 IP %s", host, ip.IP.String())
			}
		}
		// 用解析出的第一个 IP 直接拨号,避免 dialer 内部再次解析时落到不同 IP。
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}
