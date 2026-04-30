package mysql

import (
	"fmt"
	"time"

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
	lc.Append(fx.StopHook(func() {
		for name, db := range dbs {
			if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
				if e := sqlDB.Close(); e != nil {
					logs.Error("关闭mysql连接出错", zap.Error(e), zap.String("name", name))
				}
			}
		}
	}))
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
	lc.Append(fx.StopHook(func() {
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			if e := sqlDB.Close(); e != nil {
				logs.Error("关闭mysql连接出错", zap.Error(e))
			}
		}
	}))
	return db, nil
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
		nameStrategy.TablePrefix = nameStrategy.TablePrefix + "_"
	}
	gormLog := zapgorm2.New(zapLog)
	gormLog.IgnoreRecordNotFoundError = conf.GormLogIgnoreRecordNotFoundError
	gormLog.LogLevel = logger.LogLevel(conf.GormLogLevel)
	gormLog.SetAsDefault() // optional: configure gorm to use this zapgorm.Logger for callbacks
	db, err := gorm.Open(mysql.Open(conf.ConnectString), &gorm.Config{
		NamingStrategy: nameStrategy,
		Logger:         gormLog,
	})
	if err != nil {
		return nil, fmt.Errorf("数据库连接出错 name=%s: %w", conf.Name, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库底层连接出错 name=%s: %w", conf.Name, err)
	}
	maxIdleTimeConfig := conf.MaxIdleTime
	if maxIdleTimeConfig == 0 {
		maxIdleTimeConfig = maxIdleTime
	}
	sqlDB.SetConnMaxIdleTime(time.Duration(maxIdleTimeConfig) * time.Minute)

	maxLifeTimeConfig := conf.MaxLifeTime
	if maxLifeTimeConfig == 0 {
		maxLifeTimeConfig = maxLifeTime
	}
	sqlDB.SetConnMaxLifetime(time.Minute * time.Duration(maxLifeTimeConfig))

	maxIdleConnsConfig := conf.MaxIdleConns
	if maxIdleConnsConfig == 0 {
		maxIdleConnsConfig = maxIdleConns
	}
	sqlDB.SetMaxIdleConns(maxIdleConnsConfig)
	maxOpenConnsConfig := conf.MaxOpenConns
	if maxOpenConnsConfig == 0 {
		maxOpenConnsConfig = maxOpenConns
	}
	sqlDB.SetMaxOpenConns(maxOpenConnsConfig)
	return db, nil
}

func closeMysqls(dbs map[string]*gorm.DB) {
	for _, db := range dbs {
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}
