package ioc

import (
	"github.com/redis/go-redis/v9"
	"webook/config"
)

func InitRedis() redis.Cmdable {
	rCfg := config.Config.Redis
	cmd := redis.NewClient(&redis.Options{
		Addr:     rCfg.Addr,
		Password: rCfg.Password,
		DB:       rCfg.DB,
	})
	return cmd
}
