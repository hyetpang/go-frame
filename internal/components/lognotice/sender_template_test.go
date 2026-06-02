package lognotice

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateKeepsValidUTF8OnRuneBoundary(t *testing.T) {
	// 全中文,每个字符 3 字节;在不是 rune 边界的字节处截断应回退到边界,结果仍是合法 UTF-8
	s := strings.Repeat("中", 10) // 30 字节
	for _, max := range []int{1, 2, 3, 4, 5, 7, 10, 16, 29} {
		got := truncate(s, max)
		if !utf8.ValidString(got) {
			t.Fatalf("truncate(%q, %d) = %q 不是合法 UTF-8", s, max, got)
		}
		if len(got) > max {
			t.Fatalf("truncate(%q, %d) 字节长度 = %d, 超过上限", s, max, len(got))
		}
	}
}

func TestTruncateEmojiNotSplit(t *testing.T) {
	s := "🚀🚀🚀" // 每个 emoji 4 字节,共 12 字节
	got := truncate(s, 6)
	if !utf8.ValidString(got) {
		t.Fatalf("truncate emoji = %q 不是合法 UTF-8", got)
	}
}

func TestTruncateShortStringUnchanged(t *testing.T) {
	if got := truncate("abc", 10); got != "abc" {
		t.Fatalf("truncate short = %q, want abc", got)
	}
}

func TestEscapePlainReplacesNewlines(t *testing.T) {
	got := escapePlain("line1\nline2\r\nline3")
	if strings.ContainsAny(got, "\n\r") {
		t.Fatalf("escapePlain 仍含裸换行: %q", got)
	}
	if !strings.Contains(got, `\n`) || !strings.Contains(got, `\r`) {
		t.Fatalf("escapePlain 未替换为可见占位: %q", got)
	}
}

func TestEscapePlainReplacesControlChars(t *testing.T) {
	got := escapePlain("a\x00b\x07c")
	if strings.ContainsRune(got, 0x00) || strings.ContainsRune(got, 0x07) {
		t.Fatalf("escapePlain 未过滤控制字符: %q", got)
	}
}
