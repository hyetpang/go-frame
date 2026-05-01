package mysql

// 本文件实现一个 MySQL 专用的 GORM OpenTelemetry tracing 插件。
//
// 之所以不直接使用 gorm.io/plugin/opentelemetry/tracing：
//   - 上游 sub-plugin 仍然 hard-import gorm.io/driver/clickhouse 与
//     gorm.io/driver/postgres（用于 dialector 类型断言提取 server.address），
//     间接拉入 clickhouse-go、ch-go、pgx/v5、paulmach/orb 等 ~10MB indirect 依赖
//     与对应 CVE 攻击面，对于只用 MySQL 的本框架完全冗余。
//   - 上游 tracing 包还会 import metrics 子包，本框架已用 prometheus 指标，
//     不需要二次上报。
//
// 因此重写一份等价、仅依赖 otel + gorm + gorm/driver/mysql 的轻量插件。
// span 上报的语义属性参考 OpenTelemetry semconv v1.26.0：
//   - db.system        : 固定 "mysql"
//   - server.address   : 取自 mysql DSN 的 Addr 字段
//   - db.query.text    : 完整 SQL（保留 vars 注入后的字符串）
//   - db.operation.name: SQL 首词（select/insert/update/delete 等，已小写）
//   - db.collection.name: tx.Statement.Table（若可用）

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	otelTracerName = "github.com/hyetpang/go-frame/internal/components/mysql"
	otelPluginName = "otelgorm-mysql"
)

var (
	firstWordRegex   = regexp.MustCompile(`^\w+`)
	cCommentRegex    = regexp.MustCompile(`(?is)/\*.*?\*/`)
	lineCommentRegex = regexp.MustCompile(`(?im)(?:--|#).*?$`)
	sqlPrefixRegex   = regexp.MustCompile(`^[\s;]*`)

	dbRowsAffectedKey = attribute.Key("db.rows_affected")
)

// otelMySQLPlugin 是 GORM 的 tracing 插件实现。
type otelMySQLPlugin struct {
	provider trace.TracerProvider
	tracer   trace.Tracer
}

// newOtelMySQLPlugin 创建一个仅支持 MySQL 的 GORM tracing 插件。
// 当全局 TracerProvider 是 noop（未启用 OTel）时，开销接近零。
func newOtelMySQLPlugin() gorm.Plugin {
	p := &otelMySQLPlugin{provider: otel.GetTracerProvider()}
	p.tracer = p.provider.Tracer(otelTracerName)
	return p
}

func (p *otelMySQLPlugin) Name() string { return otelPluginName }

type gormHookFunc func(tx *gorm.DB)

type gormRegister interface {
	Register(name string, fn func(*gorm.DB)) error
}

func (p *otelMySQLPlugin) Initialize(db *gorm.DB) error {
	cb := db.Callback()
	hooks := []struct {
		callback gormRegister
		hook     gormHookFunc
		name     string
	}{
		{cb.Create().Before("gorm:create"), p.before("gorm.Create"), "before:create"},
		{cb.Create().After("gorm:create"), p.after(), "after:create"},

		{cb.Query().Before("gorm:query"), p.before("gorm.Query"), "before:select"},
		{cb.Query().After("gorm:query"), p.after(), "after:select"},

		{cb.Delete().Before("gorm:delete"), p.before("gorm.Delete"), "before:delete"},
		{cb.Delete().After("gorm:delete"), p.after(), "after:delete"},

		{cb.Update().Before("gorm:update"), p.before("gorm.Update"), "before:update"},
		{cb.Update().After("gorm:update"), p.after(), "after:update"},

		{cb.Row().Before("gorm:row"), p.before("gorm.Row"), "before:row"},
		{cb.Row().After("gorm:row"), p.after(), "after:row"},

		{cb.Raw().Before("gorm:raw"), p.before("gorm.Raw"), "before:raw"},
		{cb.Raw().After("gorm:raw"), p.after(), "after:raw"},
	}

	var firstErr error
	for _, h := range hooks {
		if err := h.callback.Register("otel:"+h.name, h.hook); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// contextWrapper 用于在 after 钩子里恢复原始 context。
type contextWrapper struct {
	context.Context
	parent context.Context
}

func (p *otelMySQLPlugin) before(spanName string) gormHookFunc {
	return func(tx *gorm.DB) {
		parentCtx := tx.Statement.Context
		ctx, span := p.tracer.Start(parentCtx, spanName, trace.WithSpanKind(trace.SpanKindClient))
		tx.Statement.Context = contextWrapper{Context: ctx, parent: parentCtx}

		// 仅识别 mysql dialector，提取 server.address。
		if dialector, ok := tx.Config.Dialector.(*mysql.Dialector); ok {
			if dialector.Config != nil && dialector.Config.DSNConfig != nil &&
				dialector.Config.DSNConfig.Addr != "" {
				span.SetAttributes(semconv.ServerAddress(dialector.Config.DSNConfig.Addr))
			}
		}
	}
}

func (p *otelMySQLPlugin) after() gormHookFunc {
	return func(tx *gorm.DB) {
		defer func() {
			if c, ok := tx.Statement.Context.(contextWrapper); ok {
				tx.Statement.Context = c.parent
			}
		}()

		span := trace.SpanFromContext(tx.Statement.Context)
		if !span.IsRecording() {
			return
		}
		defer span.End()

		attrs := make([]attribute.KeyValue, 0, 5)
		attrs = append(attrs, semconv.DBSystemMySQL)

		query := tx.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
		attrs = append(attrs, semconv.DBQueryText(query))
		operation := dbOperation(query)
		attrs = append(attrs, semconv.DBOperationName(operation))

		if tx.Statement.Table != "" {
			attrs = append(attrs, semconv.DBCollectionName(tx.Statement.Table))
			// db.query.summary 不在 v1.26.0，按 operation+table 形式作为 span 名。
			span.SetName(operation + " " + tx.Statement.Table)
		}
		if tx.Statement.RowsAffected != -1 {
			attrs = append(attrs, dbRowsAffectedKey.Int64(tx.Statement.RowsAffected))
		}
		span.SetAttributes(attrs...)

		switch tx.Error {
		case nil,
			gorm.ErrRecordNotFound,
			driver.ErrSkip,
			io.EOF,
			sql.ErrNoRows:
			// 忽略
		default:
			span.RecordError(tx.Error)
			span.SetStatus(codes.Error, tx.Error.Error())
		}
	}
}

func dbOperation(query string) string {
	s := cCommentRegex.ReplaceAllString(query, "")
	s = lineCommentRegex.ReplaceAllString(s, "")
	s = sqlPrefixRegex.ReplaceAllString(s, "")
	return strings.ToLower(firstWordRegex.FindString(s))
}
