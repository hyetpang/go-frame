package app

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/hyetpang/go-frame/pkgs/options"
	"go.uber.org/zap"
)

// TestRunCallsExitFuncOnConfigLoadFailure 验证配置文件不存在时 Run 走 fatalExit,
// 以非 0 退出码触发,k8s/systemd 才能感知失败重启。
func TestRunCallsExitFuncOnConfigLoadFailure(t *testing.T) {
	var exitCode atomic.Int32
	exitCode.Store(-1)

	origExit := exitFunc
	origLog := fatalLog
	t.Cleanup(func() {
		exitFunc = origExit
		fatalLog = origLog
	})
	exitFunc = func(code int) { exitCode.Store(int32(code)) }
	fatalLog = func(msg string, fields ...zap.Field) {} // 吞掉日志噪声

	Run(options.WithConfigFile("/definitely/does/not/exist/app.toml"))

	if got := exitCode.Load(); got != 1 {
		t.Fatalf("exitFunc 应被以 1 调用,实际 = %d", got)
	}
}

// TestFatalExitFlushesAndExits 单独验证 fatalExit 内部行为:
// 调日志 + 调 exitFunc(1)。zap.Sync 失败不应阻断 exit 流程。
func TestFatalExitFlushesAndExits(t *testing.T) {
	var (
		exitCode atomic.Int32
		logged   atomic.Bool
	)
	exitCode.Store(-1)

	origExit := exitFunc
	origLog := fatalLog
	t.Cleanup(func() {
		exitFunc = origExit
		fatalLog = origLog
	})
	exitFunc = func(code int) { exitCode.Store(int32(code)) }
	fatalLog = func(_ string, _ ...zap.Field) { logged.Store(true) }

	fatalExit(os.ErrInvalid)

	if !logged.Load() {
		t.Fatal("fatalExit 应记录日志")
	}
	if got := exitCode.Load(); got != 1 {
		t.Fatalf("fatalExit 应触发 exitFunc(1),实际 = %d", got)
	}
}

// TestRunWithUnreadableConfigPathTriggersExit 验证目录路径(非文件)也能走失败分支。
func TestRunWithUnreadableConfigPathTriggersExit(t *testing.T) {
	dir := t.TempDir()
	// 传入目录而非文件 — viper 读取会失败
	path := filepath.Join(dir, "subdir-not-file")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}

	var exitCode atomic.Int32
	exitCode.Store(-1)
	origExit := exitFunc
	origLog := fatalLog
	t.Cleanup(func() {
		exitFunc = origExit
		fatalLog = origLog
	})
	exitFunc = func(code int) { exitCode.Store(int32(code)) }
	fatalLog = func(_ string, _ ...zap.Field) {}

	Run(options.WithConfigFile(path))
	if got := exitCode.Load(); got != 1 {
		t.Fatalf("目录路径应触发 exitFunc(1),实际 = %d", got)
	}
}
