package kafka

import (
	"reflect"
	"testing"
)

func TestParseAddrs(t *testing.T) {
	cases := []struct {
		name string
		addr string
		want []string
	}{
		{"普通逗号分隔", "a,b", []string{"a", "b"}},
		{"带空格", "a, b", []string{"a", "b"}},
		{"前后空格", "  a  ,  b  ", []string{"a", "b"}},
		{"过滤空串", "a,,b", []string{"a", "b"}},
		{"全空串", " , , ", []string{}},
		{"单地址", "127.0.0.1:9092", []string{"127.0.0.1:9092"}},
		{"空字符串", "", []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseAddrs(c.addr)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("parseAddrs(%q) = %#v, want %#v", c.addr, got, c.want)
			}
		})
	}
}
