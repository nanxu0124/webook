package ioc

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	intrv1 "webook/api/proto/gen/intr/v1"
	"webook/interactive/service"
	"webook/internal/web/client"
	"webook/pkg/logger"
)

func InitIntrGRPCClient(svc service.InteractiveService, l logger.Logger) intrv1.InteractiveServiceClient {
	type Config struct {
		Addr      string
		Secure    bool
		Threshold int32
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.intr", &cfg)
	if err != nil {
		panic(err)
	}
	var opts []grpc.DialOption
	if !cfg.Secure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	cc, err := grpc.Dial(cfg.Addr, opts...)
	if err != nil {
		panic(err)
	}

	remote := intrv1.NewInteractiveServiceClient(cc)
	local := client.NewInteractiveServiceAdapter(svc)
	res := client.NewInteractiveClient(remote, local, cfg.Threshold)

	viper.OnConfigChange(func(in fsnotify.Event) {
		// 重置整个 Config
		cfg = Config{}
		err1 := viper.UnmarshalKey("grpc.intr", cfg)
		if err1 != nil {
			l.Error("重新加载grpc.intr的配置失败", logger.Error(err1))
			return
		}
		// 这边更新 Threshold
		res.UpdateThreshold(cfg.Threshold)
	})
	return res
}
