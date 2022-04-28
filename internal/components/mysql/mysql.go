package mysql

import (
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

func New() *gorm.DB {
	gormLog := zapgorm2.New(zap.L())
	gormLog.SetAsDefault() // optional: configure gorm to use this zapgorm.Logger for callbacks
	connectString := viper.GetString("mysql.connect_string")
	db, err := gorm.Open(mysql.Open(connectString), &gorm.Config{
		Logger: gormLog.LogMode(logger.Info),
		// DryRun: true,
	})
	if err != nil {
		logs.Fatal("数据库连接出错", zap.Error(err), zap.String("connectString", connectString))
	}
	return db
}
