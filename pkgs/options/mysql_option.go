package options

import (
	"fmt"

	"github.com/hyetpang/go-frame/internal/components/mysql"
	"github.com/hyetpang/go-frame/pkgs/common"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// 使用mysql存储,mysqlNames参数,mysqlNames是配置在mysql段的name字段,默认(default)的可以不用传
func WithMysql(mysqlNames ...string) Option {
	var isExists bool
	mysqlNameMap := make(map[string]struct{})
	for _, name := range mysqlNames {
		_, ok := mysqlNameMap[name]
		if ok {
			// 配置名重复，通过 fx.Error 将错误传入 fx 生命周期，由 app.Err() 捕获
			return func(o *Options) {
				o.FxOptions = append(o.FxOptions, fx.Error(fmt.Errorf("配置的mysql名字重复: %v", mysqlNames)))
			}
		}
		mysqlNameMap[name] = struct{}{}
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
	}
	// 多个数据库
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(mysql.New), fx.Invoke(func(dbs map[string]*gorm.DB) error {
			// 验证配置的名字是否都已经存在，返回 error 由 fx 捕获
			for _, name := range mysqlNames {
				if err := validateDB(dbs, name); err != nil {
					return err
				}
			}
			return nil
		}))
	}
}

// validateDB 验证配置名对应的数据库连接是否存在，不存在返回 error
func validateDB(dbs map[string]*gorm.DB, name string) error {
	_, ok := dbs[name]
	if !ok {
		return fmt.Errorf("配置的数据库不存在: %s", name)
	}
	return nil
}
