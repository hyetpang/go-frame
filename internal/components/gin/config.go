package gin

type config struct {
	Addr        string `mapstructure:"addr" validate:"required"` // 监听地址
	IsDoc       bool   `mapstructure:"is_doc"`                   // 是否开启文档
	DocPrefix   string `mapstructure:"doc_prefix"`               // 文档路由前缀
	IsPprof     bool   `mapstructure:"is_pprof"`                 // 是否开启pprof
	IsProd      bool   `mapstructure:"is_prod"`                  // 是否线上试环境
	MetricsPath string `mapstructure:"metrics_path"`             // 指标导出器路径
	IsMetrics   bool   `mapstructure:"is_metrics"`               // 是否使用指标导出器,这里用的是Prometheus的sdk
}
