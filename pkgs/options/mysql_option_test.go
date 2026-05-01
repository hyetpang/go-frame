package options

import (
	"testing"

	"go.uber.org/fx"
)

// TestWithMysqlReturnsFxErrorOnDuplicateName 验证 WithMysql 传入重复名字时
// fx.New 返回非空错误，而不是调用 os.Exit 绕过 Stop 流程。
func TestWithMysqlReturnsFxErrorOnDuplicateName(t *testing.T) {
	o := &Options{}
	// 传入两个相同的名字，触发重名检测
	opt := WithMysql("analytics", "analytics")
	opt(o)

	app := fx.New(o.FxOptions...)
	if app.Err() == nil {
		t.Fatal("期望 WithMysql 重名时 fx.New 返回 error，实际得到 nil")
	}
}
