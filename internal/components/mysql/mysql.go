/*
 * @Date: 2022-04-30 10:35:09
 * @LastEditTime: 2022-05-07 22:22:12
 * @FilePath: \go-frame\internal\components\mysql\mysql.go
 */
package mysql

import (
	"time"

	"github.com/HyetPang/go-frame/pkgs/dev"
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

func New(zapLog *zap.Logger) *gorm.DB {
	conf := new(config)
	err := viper.UnmarshalKey("mysql", conf)
	if err != nil {
		logs.Fatal("mysql配置Unmarshal到对象出错", zap.Error(err), zap.Any("conf", conf))
	}
	validate.MustValidate(conf)
	return newMysql(conf, zapLog)
}

func newMysql(conf *config, zapLog *zap.Logger) *gorm.DB {
	gormLog := zapgorm2.New(zapLog)
	gormLog.SetAsDefault() // optional: configure gorm to use this zapgorm.Logger for callbacks
	nameStrategy := schema.NamingStrategy{}
	nameStrategy.TablePrefix = conf.TablePrefix
	if len(nameStrategy.TablePrefix) > 0 {
		nameStrategy.TablePrefix = nameStrategy.TablePrefix + "_"
	}
	logLevel := logger.Warn
	if dev.IsDebug {
		logLevel = logger.Info
	} else {
		gormLog.IgnoreRecordNotFoundError = true
	}
	db, err := gorm.Open(mysql.Open(conf.ConnectString), &gorm.Config{
		NamingStrategy: nameStrategy,
		Logger:         gormLog.LogMode(logLevel),
	})
	if err != nil {
		logs.Fatal("数据库连接出错", zap.Error(err), zap.String("connectString", conf.ConnectString))
	}
	sqlDB, err := db.DB()
	if err != nil {
		logs.Fatal("获取数据库底层连接出错", zap.Error(err), zap.String("connectString", conf.ConnectString))
	}
	maxIdleTime := conf.MaxIdleTime
	if maxIdleTime == 0 {
		maxIdleTime = 30
	}
	sqlDB.SetConnMaxIdleTime(time.Duration(maxIdleTime) * time.Minute)

	maxLifeTime := conf.MaxLifeTime
	if maxLifeTime == 0 {
		maxLifeTime = 60
	}
	sqlDB.SetConnMaxLifetime(time.Minute * time.Duration(maxLifeTime))

	maxIdleConns := conf.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 10
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)
	maxOpenConns := conf.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 100
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)

	return db
}
