package options

import (
	"github.com/HyetPang/go-frame/internal/components/grpc"
	"go.uber.org/fx"
)

// GRPC server,使用etcd作为服务发现时，需要先调用options.WithEtcd(),初始化一个etcd客户端
func WithGRPCServer(options ...grpcOption) Option {
	gOptions := new(grpcOptions)
	for _, o := range options {
		o(gOptions)
	}
	fxOption := fx.Provide(grpc.NewServer)
	if gOptions.etcd {
		fxOption = fx.Provide(grpc.NewServerEtcd)
	}
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fxOption)
	}
}

// GRPC client,使用etcd作为服务发现时，需要先调用options.WithEtcd(),初始化一个etcd客户端
func WithGRPCClient(options ...grpcOption) Option {
	gOptions := new(grpcOptions)
	for _, o := range options {
		o(gOptions)
	}
	fxOption := fx.Provide(grpc.NewClient)
	if gOptions.etcd {
		fxOption = fx.Provide(grpc.NewClientEtcd)
	}
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fxOption)
	}
}

// Grpc 选项参数
type (
	grpcOptions struct {
		etcd bool // 是否使用etcd作为服务发现
	}
	grpcOption func(*grpcOptions)
)

// 使用etcd作为服务发现
func GrpcOptionEtcd() grpcOption {
	return func(gOption *grpcOptions) {
		gOption.etcd = true
	}
}
