package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hyetpang/go-frame/internal/lifecycle"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/zapgorm2"
)

func New(zapLog *zap.Logger, lc fx.Lifecycle, configs []config) (map[string]*gorm.DB, error) {
	if len(configs) < 1 {
		return nil, fmt.Errorf("必须配置一个数据库")
	}
	configPtrs := make([]*config, 0, len(configs))
	for i := range configs {
		conf := &configs[i]
		if err := common.Validate(conf); err != nil {
			return nil, fmt.Errorf("mysql配置验证不通过 name=%s: %w", conf.Name, err)
		}
		configPtrs = append(configPtrs, conf)
	}
	dbs, err := newMysqls(configPtrs, zapLog)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// 多 DB 串行关闭:任一卡住时剩余 DB 仍能在 fx Stop timeout 内尝试关闭。
			// 把总预算按剩余实例数均分给每个实例独立子超时,避免某个慢 Close
			// 耗尽整体 ctx 后,后续实例因 ctx 已超时而直接跳过、得不到关闭机会。
			// 不提前 return,继续尽力关闭剩余实例并聚合错误。
			remaining := len(dbs)
			var errs []error
			for name, db := range dbs {
				sqlDB, err := db.DB()
				if err != nil || sqlDB == nil {
					remaining--
					continue
				}
				// 为当前实例分配独立子超时:剩余总预算 / 剩余待关闭实例数。
				closeCtx, cancel := perInstanceCloseContext(ctx, remaining)
				e := lifecycle.CloseWithContext(closeCtx, "mysql/"+name, sqlDB.Close)
				cancel()
				remaining--
				if e != nil {
					logs.Error("关闭mysql连接出错", zap.Error(e), zap.String("name", name))
					errs = append(errs, e)
				}
			}
			return errors.Join(errs...)
		},
	})
	return dbs, nil
}

func NewOne(zapLog *zap.Logger, lc fx.Lifecycle, configs []config) (*gorm.DB, error) {
	conf, err := pickOneConfig(configs)
	if err != nil {
		return nil, err
	}
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("mysql配置验证不通过: %w", err)
	}
	db, err := newMysql(conf, zapLog)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil || sqlDB == nil {
				return nil
			}
			if e := lifecycle.CloseWithContext(ctx, "mysql", sqlDB.Close); e != nil {
				logs.Error("关闭mysql连接出错", zap.Error(e))
				return e
			}
			return nil
		},
	})
	return db, nil
}

// perInstanceCloseContext 为多实例关闭中的单个实例派生独立子超时。
// 当父 ctx 设置了 deadline 时,把剩余时间均分给剩余待关闭实例(remaining),
// 这样即便前一个实例耗尽了自己那一份预算,后续实例仍能拿到属于自己的关闭窗口;
// 父 ctx 无 deadline 时直接透传,不引入额外限制。
func perInstanceCloseContext(ctx context.Context, remaining int) (context.Context, context.CancelFunc) {
	deadline, ok := ctx.Deadline()
	if !ok || remaining <= 1 {
		// 无 deadline,或已是最后一个实例:整段剩余预算都给它,无需再切分。
		return context.WithCancel(ctx)
	}
	budget := time.Until(deadline)
	if budget <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, budget/time.Duration(remaining))
}

func pickOneConfig(configs []config) (*config, error) {
	if len(configs) < 1 {
		return nil, fmt.Errorf("必须配置一个数据库")
	}
	for i := range configs {
		if configs[i].Name == common.DefaultDb {
			return &configs[i], nil
		}
	}
	return &configs[0], nil
}

func newMysqls(configs []*config, zapLog *zap.Logger) (map[string]*gorm.DB, error) {
	dbs := make(map[string]*gorm.DB)
	for _, conf := range configs {
		_, ok := dbs[conf.Name]
		if ok {
			closeMysqls(dbs)
			return nil, fmt.Errorf("数据库连接名字重复 name=%s", conf.Name)
		}
		db, err := newMysql(conf, zapLog)
		if err != nil {
			closeMysqls(dbs)
			return nil, err
		}
		dbs[conf.Name] = db
	}
	return dbs, nil
}

// TODO 增加指标监控 https://github.com/go-gorm/prometheus
func newMysql(conf *config, zapLog *zap.Logger) (*gorm.DB, error) {
	nameStrategy := schema.NamingStrategy{}
	nameStrategy.TablePrefix = conf.TablePrefix
	if len(nameStrategy.TablePrefix) > 0 {
		nameStrategy.TablePrefix += "_"
	}
	gormLog := zapgorm2.New(zapLog)
	gormLog.IgnoreRecordNotFoundError = conf.GormLogIgnoreRecordNotFoundError
	gormLog.LogLevel = logger.LogLevel(conf.GormLogLevel)
	db, err := gorm.Open(mysql.Open(conf.ConnectString), &gorm.Config{
		NamingStrategy: nameStrategy,
		Logger:         gormLog,
	})
	if err != nil {
		return nil, fmt.Errorf("数据库连接出错 name=%s: %w", conf.Name, err)
	}
	// 注入项目内置的 MySQL 专用 OpenTelemetry tracing 插件；Tracing 关闭时
	// 全局 TracerProvider 是 noop，本插件成本接近零。该实现替换上游
	// gorm.io/plugin/opentelemetry/tracing，避免间接引入 clickhouse、postgres
	// 等无关 driver（详见 otelplugin.go 的说明）。
	if err := db.Use(newOtelMySQLPlugin()); err != nil {
		return nil, fmt.Errorf("注入 gorm OpenTelemetry 插件出错 name=%s: %w", conf.Name, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库底层连接出错 name=%s: %w", conf.Name, err)
	}
	sqlDB.SetConnMaxIdleTime(time.Duration(defaultIfNonPositive(conf.MaxIdleTime, maxIdleTime)) * time.Minute)
	sqlDB.SetConnMaxLifetime(time.Minute * time.Duration(defaultIfNonPositive(conf.MaxLifeTime, maxLifeTime)))
	sqlDB.SetMaxIdleConns(defaultIfNonPositive(conf.MaxIdleConns, maxIdleConns))
	sqlDB.SetMaxOpenConns(defaultIfNonPositive(conf.MaxOpenConns, maxOpenConns))
	return db, nil
}

// defaultIfNonPositive 在配置值 <= 0 时回退到默认值。
// 统一用 <= 0 兜底:未配置(0)与误配负数(如 -1)都走默认。否则 -1 会被原样透传给
// database/sql,SetMaxOpenConns(-1) 表示无限连接可能耗尽 DB,SetMaxIdleConns(-1)
// 等同关闭空闲连接复用,均非预期行为。
func defaultIfNonPositive(value, def int) int {
	if value <= 0 {
		return def
	}
	return value
}

func closeMysqls(dbs map[string]*gorm.DB) {
	for _, db := range dbs {
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}
