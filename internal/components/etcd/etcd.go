package etcd

import (
	"strings"
	"time"

	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/HyetPang/go-frame/pkgs/validate"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

func New(zaoLog *zap.Logger) *clientv3.Client {
	conf := newConfig()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            strings.Split(conf.Addresses, ","),
		AutoSyncInterval:     0,
		DialTimeout:          time.Second * 5,
		DialKeepAliveTime:    time.Second * 3,
		DialKeepAliveTimeout: time.Second * 5,
		MaxCallSendMsgSize:   0,
		MaxCallRecvMsgSize:   0,
		RejectOldCluster:     false,
		Logger:               zaoLog,
		PermitWithoutStream:  false,
	})
	if err != nil {
		logs.Fatal("创建etcd客户端出错", zap.Error(err), zap.String("addresses", conf.Addresses))
	}
	return cli
}

func newConfig() *config {
	conf := new(config)
	err := viper.UnmarshalKey("etcd", &conf)
	if err != nil {
		logs.Fatal("kafka配置Unmarshal到对象出错", zap.Error(err), zap.Any("conf", conf))
	}
	validate.Must(conf)
	return conf
}
