/*
 * @Date: 2022-04-30 10:35:09
 * @LastEditTime: 2022-04-30 17:57:51
 * @FilePath: \go-frame\internal\components\mysql\mysql.go
 */
package mysql

import (
	"strings"
	"time"

	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/zapgorm2"
)

func New(zapLog *zap.Logger) *gorm.DB {
	gormLog := zapgorm2.New(zapLog)
	gormLog.SetAsDefault() // optional: configure gorm to use this zapgorm.Logger for callbacks
	connectString := viper.GetString("mysql.connect_string")
	nameStrategy := schema.NamingStrategy{}
	nameStrategy.TablePrefix = viper.GetString("mysql.table_prefix")
	if len(nameStrategy.TablePrefix) > 0 && !strings.HasSuffix(nameStrategy.TablePrefix, "_") {
		nameStrategy.TablePrefix = nameStrategy.TablePrefix + "_"
	}
	db, err := gorm.Open(mysql.Open(connectString), &gorm.Config{
		NamingStrategy: nameStrategy,
		Logger:         gormLog.LogMode(logger.Info),
	})
	if err != nil {
		logs.Fatal("数据库连接出错", zap.Error(err), zap.String("connectString", connectString))
	}
	sqlDB, err := db.DB()
	if err != nil {
		logs.Fatal("获取数据库底层连接出错", zap.Error(err), zap.String("connectString", connectString))
	}
	maxIdleTime := viper.GetInt("mysql.max_idle_time")
	if maxIdleTime == 0 {
		maxIdleTime = 30
	}
	sqlDB.SetConnMaxIdleTime(time.Duration(maxIdleTime) * time.Minute)

	maxLifeTime := viper.GetInt("mysql.max_life_time")
	if maxLifeTime == 0 {
		maxLifeTime = 60
	}
	sqlDB.SetConnMaxLifetime(time.Minute * time.Duration(maxLifeTime))

	maxIdleConns := viper.GetInt("mysql.max_idle_conns")
	if maxIdleConns == 0 {
		maxIdleConns = 10
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)
	maxOpenConns := viper.GetInt("mysql.max_open_conns")
	if maxOpenConns == 0 {
		maxOpenConns = 100
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)

	return db
}
