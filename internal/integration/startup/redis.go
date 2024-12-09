package startup

import (
	"context"
	"github.com/redis/go-redis/v9"
)

var redisClient redis.Cmdable

func InitTestRedis() redis.Cmdable {
	if redisClient == nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})

		for err := redisClient.Ping(context.Background()).Err(); err != nil; {
			panic(err)
		}
	}
	return redisClient
}
