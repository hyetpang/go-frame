/*
 * @Date: 2022-04-30 10:35:09
 * @LastEditTime: 2022-05-07 22:22:12
 * @FilePath: \go-frame\internal\components\mysql\mysql.go
 */
package mysql

import (
	"time"

	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/HyetPang/go-frame/pkgs/validate"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/zapgorm2"
)

func New(zapLog *zap.Logger) map[string]*gorm.DB {
	configs := make([]*config, 0, 3)
	err := viper.UnmarshalKey("mysql", &configs)
	if err != nil {
		logs.Fatal("mysql配置Unmarshal到对象出错", zap.Error(err), zap.Any("conf", configs))
	}
	if len(configs) < 1 {
		logs.Fatal("必须配置一个数据库", zap.Error(err), zap.Any("conf", configs))
	}
	for _, conf := range configs {
		validate.MustValidate(conf)
	}
	return newMysqls(configs, zapLog)
}

func NewOne(zapLog *zap.Logger) *gorm.DB {
	conf := new(config)
	err := viper.UnmarshalKey("mysql", &conf)
	if err != nil {
		logs.Fatal("mysql配置Unmarshal到对象出错", zap.Error(err))
	}
	validate.MustValidate(conf)
	return newMysql(conf, zapLog)
}

func newMysqls(configs []*config, zapLog *zap.Logger) map[string]*gorm.DB {
	dbs := make(map[string]*gorm.DB)
	for _, conf := range configs {
		_, ok := dbs[conf.Name]
		if ok {
			logs.Fatal("数据库连接名字重复", zap.String("name", conf.Name))
		}
		dbs[conf.Name] = newMysql(conf, zapLog)
	}
	return dbs
}

// TODO 增加指标监控 https://github.com/go-gorm/prometheus
func newMysql(conf *config, zapLog *zap.Logger) *gorm.DB {
	gormLog := zapgorm2.New(zapLog)
	gormLog.SetAsDefault() // optional: configure gorm to use this zapgorm.Logger for callbacks
	nameStrategy := schema.NamingStrategy{}
	nameStrategy.TablePrefix = conf.TablePrefix
	if len(nameStrategy.TablePrefix) > 0 {
		nameStrategy.TablePrefix = nameStrategy.TablePrefix + "_"
	}
	// logLevel := logger.Warn
	// if dev.IsDebug {
	// 	// 开发环境打印执行的sql
	// 	logLevel = logger.Info
	// } else {
	// 	//
	// 	gormLog.IgnoreRecordNotFoundError = true
	// }
	db, err := gorm.Open(mysql.Open(conf.ConnectString), &gorm.Config{
		NamingStrategy: nameStrategy,
		Logger:         gormLog.LogMode(logger.Info),
	})
	if err != nil {
		logs.Fatal("数据库连接出错", zap.Error(err), zap.String("connectString", conf.ConnectString))
	}
	sqlDB, err := db.DB()
	if err != nil {
		logs.Fatal("获取数据库底层连接出错", zap.Error(err), zap.String("connectString", conf.ConnectString))
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
	return db
}
