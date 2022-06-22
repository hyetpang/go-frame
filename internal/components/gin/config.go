package gin

type config struct {
	Addr    string `mapstructure:"addr" validate:"required"` // 监听地址
	IsDoc   bool   `mapstructure:"is_doc"`                   // 是否开启文档
	IsPprof bool   `mapstructure:"is_pprof"`                 // 是否开启pprof
	IsProd  bool   `mapstructure:"is_prod"`                  // 是否线上试环境
}
