package lognotice

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// privateCIDRs 收集需要拒绝的内网/回环/链路本地/IPv6 ULA 地址段,用于 SSRF 防护。
var privateCIDRs = func() []*net.IPNet {
	blocks := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918 私网
		"172.16.0.0/12",  // RFC1918 私网
		"192.168.0.0/16", // RFC1918 私网
		"169.254.0.0/16", // 链路本地
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 ULA
		"fe80::/10",      // IPv6 链路本地
	}
	out := make([]*net.IPNet, 0, len(blocks))
	for _, cidr := range blocks {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		out = append(out, ipNet)
	}
	return out
}()

// validateWebhookURL 校验通知 webhook URL,拒绝私网/回环/链路本地等危险目标,
// 强制 https(白名单 host 例外),并支持白名单后缀匹配限制可达域名。
func validateWebhookURL(rawURL string, allowedHosts []string) error {
	if strings.TrimSpace(rawURL) == "" {
		return errors.New("webhook URL 为空")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("webhook URL 解析失败: %w", err)
	}
	if parsed.Host == "" {
		return errors.New("webhook URL 缺少 host")
	}
	host := parsed.Hostname()
	if host == "" {
		return errors.New("webhook URL host 为空")
	}

	allowed := normalizeAllowedHosts(allowedHosts)
	hostMatched := matchHostInAllowList(host, allowed)
	if len(allowed) > 0 && !hostMatched {
		return fmt.Errorf("webhook host %q 不在白名单内", host)
	}

	// 仅在白名单未显式包含该 host 时强制 https,允许白名单内的 host 走 http(开发场景)。
	if parsed.Scheme != "https" && !hostMatched {
		return fmt.Errorf("webhook 必须使用 https,实际 scheme=%q", parsed.Scheme)
	}

	if ip := net.ParseIP(host); ip != nil {
		if isPrivateOrLoopback(ip) && !hostMatched {
			return fmt.Errorf("webhook host %q 命中私网/回环/链路本地地址段", host)
		}
		return nil
	}

	// 主机名:解析后逐个检查,任一 IP 命中私网即拒绝。
	if hostMatched {
		// 已显式列入白名单,跳过 IP 黑名单(避免开发环境内网域名被误杀)。
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("webhook host %q DNS 解析失败: %w", host, err)
	}
	for _, ip := range ips {
		if isPrivateOrLoopback(ip) {
			return fmt.Errorf("webhook host %q 解析到私网/回环 IP %s", host, ip.String())
		}
	}
	return nil
}

func isPrivateOrLoopback(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	for _, block := range privateCIDRs {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func normalizeAllowedHosts(allowed []string) []string {
	out := make([]string, 0, len(allowed))
	for _, host := range allowed {
		host = strings.TrimSpace(strings.ToLower(host))
		if host == "" {
			continue
		}
		out = append(out, host)
	}
	return out
}

// matchHostInAllowList 精确匹配或后缀匹配(.domain),白名单为空视为不命中。
func matchHostInAllowList(host string, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	host = strings.ToLower(host)
	for _, pattern := range allowed {
		if host == pattern {
			return true
		}
		if strings.HasSuffix(host, "."+pattern) {
			return true
		}
	}
	return false
}
