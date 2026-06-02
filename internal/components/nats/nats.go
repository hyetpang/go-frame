package nats

import "strings"

// parseAddrs 解析逗号分隔的 NATS 地址串,对每个地址做 TrimSpace 并过滤空串,
// 避免 "a, b" 这类带空格的配置产生非法地址。
func parseAddrs(addr string) []string {
	parts := strings.Split(addr, ",")
	addrs := make([]string, 0, len(parts))
	for _, p := range parts {
		if a := strings.TrimSpace(p); a != "" {
			addrs = append(addrs, a)
		}
	}
	return addrs
}
