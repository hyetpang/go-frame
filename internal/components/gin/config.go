package gin

type config struct {
	Addr        string `mapstructure:"addr"`                                            // 监听地址
	DocPath     string `mapstructure:"doc_path" validate:"required_with=IsDoc"`         // 文档路径
	PprofPrefix string `mapstructure:"pprof_prefix"`                                    // pprof前缀
	MetricsPath string `mapstructure:"metrics_path" validate:"required_with=IsMetrics"` // 指标导出器路径
	IsDoc       bool   `mapstructure:"is_doc"`                                          // 是否开启文档
	IsPprof     bool   `mapstructure:"is_pprof"`                                        // 是否开启pprof
	IsMetrics   bool   `mapstructure:"is_metrics"`                                      // 是否使用指标导出器,这里用的是Prometheus的sdk
	IsProd      bool   `mapstructure:"is_prod"`                                         // 是否线上
}
