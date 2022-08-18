package options

func WithGraceRestart() Option {
	return func(o *Options) {
		o.UseGraceRestart = true
	}
}
