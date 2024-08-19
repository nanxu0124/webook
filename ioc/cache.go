package ioc

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitCache() redis.Cmdable {
	type config struct {
		Addr string `yaml:"addr"`
	}
	var c config
	err := viper.UnmarshalKey("Redis", &c)
	if err != nil {
		panic(fmt.Errorf("初始化配置失败 %v", err))
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: c.Addr,
	})
	return redisClient
}
