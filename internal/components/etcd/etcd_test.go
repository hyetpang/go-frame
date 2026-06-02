package etcd

import (
	"reflect"
	"testing"
)

func TestParseEndpoints(t *testing.T) {
	cases := []struct {
		name      string
		addresses string
		want      []string
	}{
		{"普通逗号分隔", "a,b", []string{"a", "b"}},
		{"带空格", "a, b", []string{"a", "b"}},
		{"前后空格", "  a  ,  b  ", []string{"a", "b"}},
		{"过滤空串", "a,,b", []string{"a", "b"}},
		{"全空串", " , , ", []string{}},
		{"单地址", "127.0.0.1:2379", []string{"127.0.0.1:2379"}},
		{"空字符串", "", []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseEndpoints(c.addresses)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("parseEndpoints(%q) = %#v, want %#v", c.addresses, got, c.want)
			}
		})
	}
}
