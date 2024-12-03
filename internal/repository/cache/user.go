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

// UserCache 是一个用户缓存的接口
// 该接口定义了两个方法：Get 和 Set，用于从缓存中获取用户信息和将用户信息存储到缓存中
// UserCache 的实现通常会选择 Redis 或内存缓存作为底层存储
// 以提高频繁访问的用户数据的查询速度，并减少数据库的压力
type UserCache interface {

	// Get 从缓存中获取用户信息
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - id: 用户的唯一标识 ID
	// 返回:
	//   - domain.User: 缓存中存储的用户信息
	//   - error: 如果缓存中没有该用户信息或发生错误，返回错误信息
	Get(ctx context.Context, id int64) (domain.User, error)

	// Set 将用户信息存储到缓存中
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - u: 用户信息，包含用户的基本信息（如 ID、邮箱等）
	// 返回:
	//   - error: 如果存储过程中发生错误，返回错误信息；如果存储成功，返回 nil
	Set(ctx context.Context, u domain.User) error
}

// RedisUserCache 实现 UserCache 接口，封装了对 Redis 的操作
type RedisUserCache struct {
	cmd        redis.Cmdable // Redis 命令接口，定义了与 Redis 进行交互的基本命令
	expiration time.Duration // 缓存过期时间
}

func NewRedisUserCache(cmd redis.Cmdable) UserCache {
	return &RedisUserCache{
		cmd:        cmd,
		expiration: time.Minute * 15, // 设置缓存过期时间为 15 分钟
	}
}

func (cache *RedisUserCache) Get(ctx context.Context, id int64) (domain.User, error) {
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

func (cache *RedisUserCache) Set(ctx context.Context, u domain.User) error {
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
func (cache *RedisUserCache) key(id int64) string {
	return fmt.Sprintf("user:info:%d", id)
}
