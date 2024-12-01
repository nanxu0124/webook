package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
	"webook/internal/domain"
)

// ErrKeyNotExist  Redis 特有的错误，用于表示键不存在
var ErrKeyNotExist = redis.Nil

// UserCache 用于缓存用户信息
// 它封装了对 Redis 的操作，包括从缓存中获取用户数据和将用户数据存入缓存
type UserCache struct {
	cmd        redis.Cmdable // Redis 命令接口，定义了与 Redis 进行交互的基本命令
	expiration time.Duration // 缓存过期时间
}

// NewUserCache 创建并返回一个新的 UserCache 实例
// 参数 cmd 是一个 Redis 命令接口实例，通常是 *redis.Client
func NewUserCache(cmd redis.Cmdable) *UserCache {
	return &UserCache{
		cmd:        cmd,              // 设置 Redis 命令接口
		expiration: time.Minute * 15, // 设置缓存过期时间为 15 分钟
	}
}

// Get 根据用户 ID 从缓存中获取用户信息
// 如果缓存中有数据，返回该数据；如果没有数据或发生错误，返回错误
func (cache *UserCache) Get(ctx context.Context, id int64) (domain.User, error) {
	key := cache.key(id) // 生成缓存的键
	// 从 Redis 获取数据
	data, err := cache.cmd.Get(ctx, key).Result()
	if err != nil {
		// 如果获取数据失败，直接返回错误
		return domain.User{}, err
	}
	// 反序列化从 Redis 获取的 JSON 数据
	var u domain.User
	err = json.Unmarshal([]byte(data), &u)
	return u, err
}

// Set 将用户数据存入缓存
// 数据会被序列化为 JSON 格式，并设置过期时间
func (cache *UserCache) Set(ctx context.Context, u domain.User) error {
	// 将用户数据序列化为 JSON 格式
	data, err := json.Marshal(u)
	if err != nil {
		// 如果序列化失败，返回错误
		return err
	}
	// 生成缓存的键
	key := cache.key(u.Id)
	// 将数据存入 Redis，设置过期时间
	return cache.cmd.Set(ctx, key, data, cache.expiration).Err()
}

// key 生成缓存中用户数据的键
// 键的格式为 "user:info:{id}"，以便区分不同的用户
func (cache *UserCache) key(id int64) string {
	return fmt.Sprintf("user:info:%d", id)
}
