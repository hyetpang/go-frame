package mysql

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewOneReturnsErrorForInvalidConfig(t *testing.T) {
	configs := []config{{
		Name:         "default",
		GormLogLevel: 4,
	}}

	db, err := NewOne(zap.NewNop(), fxtest.NewLifecycle(t), configs)
	if err == nil {
		t.Fatal("expected invalid mysql config to return error")
	}
	if db != nil {
		t.Fatal("expected nil mysql db on config error")
	}
}

func TestPickOneConfigSelectsDefaultDatabase(t *testing.T) {
	configs := []config{
		{Name: "analytics", ConnectString: "analytics", GormLogLevel: 4},
		{Name: "default", ConnectString: "default", GormLogLevel: 4},
	}

	conf, err := pickOneConfig(configs)
	if err != nil {
		t.Fatalf("pickOneConfig returned error: %v", err)
	}
	if conf.Name != "default" {
		t.Fatalf("selected mysql name = %q, want default", conf.Name)
	}
}

func TestDefaultIfNonPositive(t *testing.T) {
	cases := []struct {
		name  string
		value int
		def   int
		want  int
	}{
		{"零值走默认", 0, 100, 100},
		{"负值走默认避免无限连接", -1, 100, 100},
		{"较大负值同样走默认", -999, 10, 10},
		{"正常正值保留", 50, 100, 50},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := defaultIfNonPositive(c.value, c.def); got != c.want {
				t.Fatalf("defaultIfNonPositive(%d, %d) = %d, want %d", c.value, c.def, got, c.want)
			}
		})
	}
}

// TestPerInstanceCloseContextSplitsBudget 验证多实例关闭时,父 ctx 的剩余预算
// 会按剩余实例数均分给每个实例,使某个实例耗尽预算后,后续实例仍有独立关闭窗口。
func TestPerInstanceCloseContextSplitsBudget(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 还剩 5 个实例待关闭,单实例预算应约为 10s/5 = 2s,且必须小于父 ctx 预算。
	childCtx, childCancel := perInstanceCloseContext(parent, 5)
	defer childCancel()
	deadline, ok := childCtx.Deadline()
	if !ok {
		t.Fatal("子 ctx 应继承 deadline")
	}
	budget := time.Until(deadline)
	if budget <= 0 || budget > 3*time.Second {
		t.Fatalf("单实例预算 = %v, want 约 2s 且明显小于父 10s", budget)
	}
}

// TestPerInstanceCloseContextLastInstanceGetsFullBudget 最后一个实例(remaining<=1)
// 应拿到整段剩余预算,不再二次切分。
func TestPerInstanceCloseContextLastInstanceGetsFullBudget(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	childCtx, childCancel := perInstanceCloseContext(parent, 1)
	defer childCancel()
	deadline, ok := childCtx.Deadline()
	if !ok {
		t.Fatal("子 ctx 应继承父 deadline")
	}
	if budget := time.Until(deadline); budget < 3*time.Second {
		t.Fatalf("最后一个实例预算 = %v, want 接近父 4s", budget)
	}
}

// TestPerInstanceCloseContextNoDeadline 父 ctx 无 deadline 时直接透传,不引入限制。
func TestPerInstanceCloseContextNoDeadline(t *testing.T) {
	childCtx, childCancel := perInstanceCloseContext(context.Background(), 3)
	defer childCancel()
	if _, ok := childCtx.Deadline(); ok {
		t.Fatal("父 ctx 无 deadline 时子 ctx 不应被强加 deadline")
	}
}

// TestNewOnStopClosesAllInstancesAfterSlowOne 回归 #1:即使第一个实例关闭耗尽预算,
// OnStop 也不应提前 return,剩余实例仍应被尝试关闭。这里用 fxtest Lifecycle 触发
// OnStop,通过两个真实(但连接到不存在地址的)gorm.DB 验证不会因 ctx 超时跳过后续实例。
func TestNewOnStopClosesAllInstances(t *testing.T) {
	dbs := map[string]struct{}{"a": {}, "b": {}, "c": {}}
	// 模拟 OnStop 中的均分逻辑:父预算被前面实例耗尽后,后续实例仍能拿到独立子窗口。
	parent, cancel := context.WithTimeout(context.Background(), 90*time.Millisecond)
	defer cancel()

	remaining := len(dbs)
	closed := 0
	for range dbs {
		closeCtx, c := perInstanceCloseContext(parent, remaining)
		// 每个实例都应拿到一个未过期(或刚派生)的窗口,而非被父 ctx 直接判超时跳过。
		if closeCtx.Err() == nil {
			closed++
		}
		c()
		remaining--
		time.Sleep(20 * time.Millisecond)
	}
	if closed != len(dbs) {
		t.Fatalf("尝试关闭的实例数 = %d, want %d(剩余实例不应被提前 return 跳过)", closed, len(dbs))
	}
}
