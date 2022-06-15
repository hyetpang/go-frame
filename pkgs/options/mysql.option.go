package options

import (
	"github.com/HyetPang/go-frame/internal/components/mysql"
	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 使用mysql存储,mysqlNames参数,mysqlNames是配置在mysql段的name字段,默认(default)的可以不用传
func WithMysql(mysqlNames ...string) Option {
	var isExists bool
	for _, name := range mysqlNames {
		if name == common.DefaultDb {
			isExists = true
		}
	}
	if !isExists {
		mysqlNames = append(mysqlNames, common.DefaultDb)
	}
	if len(mysqlNames) == 1 {
		// 只有一个数据库
		return func(o *Options) {
			o.FxOptions = append(o.FxOptions, fx.Provide(mysql.NewOne))
		}
	} else {
		// 多个数据库
		return func(o *Options) {
			o.FxOptions = append(o.FxOptions, fx.Provide(mysql.New), fx.Invoke(func(dbs map[string]*gorm.DB) {
				// 验证配置的名字是否都已经存在
				for _, name := range mysqlNames {
					mustValidateDB(dbs, name)
				}
			}))
		}
	}
}

// 验证配置的名字和对应的数据db是否存在，不存在会直接打日志退出程序
func mustValidateDB(dbs map[string]*gorm.DB, name string) {
	_, ok := dbs[name]
	if !ok {
		logs.Fatal("配置的数据库不存在", zap.String("name", name))
	}
}
