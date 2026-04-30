package logs

import (
	"testing"

	"go.uber.org/zap"
)

func TestApplyDefaultsUsesConfiguredRotationAndStacktrace(t *testing.T) {
	conf := &config{
		LogMaxSize:      64,
		LogMaxBackups:   8,
		LogMaxAge:       3,
		StacktraceLevel: 2,
	}

	applyDefaults(conf)

	if conf.LogMaxSize != 64 || conf.LogMaxBackups != 8 || conf.LogMaxAge != 3 {
		t.Fatalf("rotation defaults overwrote configured values: %+v", conf)
	}
	if stacktraceLevel(conf) != zap.ErrorLevel {
		t.Fatalf("stacktrace level = %v, want error", stacktraceLevel(conf))
	}
}

func TestApplyDefaultsFillsMissingRotationAndStacktrace(t *testing.T) {
	conf := &config{}

	applyDefaults(conf)

	if conf.LogMaxSize != logMaxSize || conf.LogMaxBackups != logMaxBackups || conf.LogMaxAge != logMaxAge {
		t.Fatalf("rotation defaults = %+v", conf)
	}
	if stacktraceLevel(conf) != zap.WarnLevel {
		t.Fatalf("stacktrace level = %v, want warn", stacktraceLevel(conf))
	}
}
