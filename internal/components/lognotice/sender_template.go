package lognotice

import (
	"html"
	"strings"
)

const (
	// maxNoticeFieldLen 限制写入 webhook 的单字段长度,
	// 防止超长用户输入触发第三方平台限流或导致告警通道被刷屏。
	maxNoticeFieldLen = 1024
)

// truncate 将字符串截断到 max 长度,超出部分以 "..." 提示。
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// escapeHTML 对字符串做 HTML 转义,用于走 markdown / HTML parse_mode 的 webhook,
// 防止用户输入注入 <a>/<font>/@all 等富文本元素或被解析为协议链接。
func escapeHTML(s string) string {
	return html.EscapeString(truncate(s, maxNoticeFieldLen))
}

// escapePlain 对字符串做基础清理:截断 + 控制字符替换,
// 用于飞书等纯文本通道(避免换行注入伪造多行结构)。
func escapePlain(s string) string {
	s = truncate(s, maxNoticeFieldLen)
	// 替换 \r 与裸 \n 之外的控制字符,防止终端转义注入
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			b.WriteRune('?')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
