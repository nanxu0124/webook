package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
	"webook/internal/domain"
)

var ErrKeyNotExist = redis.Nil

type UserCache interface {
	Set(ctx context.Context, u domain.User) error
	Get(ctx context.Context, id int64) (domain.User, error)
}

type RedisUserCache struct {
	// 传单机 Redis 可以
	// 穿 cluster 的 Redis 也可以
	client     redis.Cmdable
	expiration time.Duration
}

// A 用到了 B， B 一定是接口
// A 用到了 B， B 一定是A的字段
// A 用到了 B， A 绝对不初始化 B， 而是外面传入

func NewRedisUserCache(client redis.Cmdable) UserCache {
	return &RedisUserCache{
		client:     client,
		expiration: time.Minute * 15,
	}
}

func (cache *RedisUserCache) key(id int64) string {
	return fmt.Sprintf("user:info:%d", id)
}

func (cache *RedisUserCache) Set(ctx context.Context, u domain.User) error {
	val, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return cache.client.Set(ctx, cache.key(u.Id), val, cache.expiration).Err()
}

// Get 只要error为nil 就认为缓存里有数据
// 如果没有数据，返回一个特定的error
func (cache *RedisUserCache) Get(ctx context.Context, id int64) (domain.User, error) {
	val, err := cache.client.Get(ctx, cache.key(id)).Bytes()
	if err != nil {
		return domain.User{}, err
	}
	var u domain.User
	err = json.Unmarshal(val, &u)
	return u, err
}
