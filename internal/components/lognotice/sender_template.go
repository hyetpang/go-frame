package lognotice

import (
	"html"
	"strings"
	"unicode/utf8"
)

const (
	// maxNoticeFieldLen 限制写入 webhook 的单字段长度,
	// 防止超长用户输入触发第三方平台限流或导致告警通道被刷屏。
	maxNoticeFieldLen = 1024
)

// truncate 将字符串按字节上限 max 安全截断,超出部分以 "..." 提示。
// 截断按 UTF-8 rune 边界回退,避免把中文/emoji 切成半个码点产生乱码或非法 UTF-8。
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return safeCutBytes(s, max)
	}
	return safeCutBytes(s, max-3) + "..."
}

// safeCutBytes 返回 s 中不超过 limit 字节、且不切断 UTF-8 rune 的最长前缀。
func safeCutBytes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if limit >= len(s) {
		return s
	}
	// 从 limit 处向前回退,直到落在某个 rune 的起始边界上
	cut := limit
	for cut > 0 {
		if utf8.RuneStart(s[cut]) {
			break
		}
		cut--
	}
	return s[:cut]
}

// escapeHTML 对字符串做 HTML 转义,用于走 markdown / HTML parse_mode 的 webhook,
// 防止用户输入注入 <a>/<font>/@all 等富文本元素或被解析为协议链接。
func escapeHTML(s string) string {
	return html.EscapeString(truncate(s, maxNoticeFieldLen))
}

// escapePlain 对字符串做基础清理:截断 + 控制字符替换,
// 用于飞书等纯文本通道。为避免换行注入伪造多行结构,
// 用户字段里的 \n / \r 会被替换为可见占位 "\n" / "\r"(字面反斜杠 + 字母),
// 其余不可打印控制字符替换为 '?'。
func escapePlain(s string) string {
	s = truncate(s, maxNoticeFieldLen)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			b.WriteRune('?')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
