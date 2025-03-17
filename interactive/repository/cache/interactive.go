package cache

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
	"webook/interactive/domain"
)

var (
	//go:embed lua/interative_incr_cnt.lua
	luaIncrCnt string
)

var ErrKeyNotExist = redis.Nil

const (
	fieldReadCnt    = "read_cnt"
	fieldCollectCnt = "collect_cnt"
	fieldLikeCnt    = "like_cnt"
)

type InteractiveCache interface {

	// IncrReadCntIfPresent 如果在缓存中有对应的数据，就 +1
	IncrReadCntIfPresent(ctx context.Context, biz string, bizId int64) error
	IncrLikeCntIfPresent(ctx context.Context, biz string, bizId int64) error
	DecrLikeCntIfPresent(ctx context.Context, biz string, bizId int64) error
	IncrCollectCntIfPresent(ctx context.Context, biz string, bizId int64) error
	// Get 查询缓存中数据
	Get(ctx context.Context, biz string, bizId int64) (domain.Interactive, error)
	Set(ctx context.Context, biz string, bizId int64, intr domain.Interactive) error
}

type RedisInteractiveCache struct {
	client     redis.Cmdable
	expiration time.Duration
}

func NewRedisInteractiveCache(client redis.Cmdable) InteractiveCache {
	return &RedisInteractiveCache{
		client: client,
	}
}

// IncrReadCntIfPresent 增加指定业务（biz）和业务 ID（bizId）对应的 read 计数（字段名为 fieldReadCnt）
// 如果该键存在，执行自增操作。
// 返回一个错误（如果有的话），否则返回 nil
func (r *RedisInteractiveCache) IncrReadCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	// 调用 Redis 执行 Lua 脚本，增加缓存中的阅读计数
	return r.client.Eval(ctx, luaIncrCnt, // 使用 Lua 脚本来进行自增操作
		[]string{r.key(biz, bizId)}, // 键名，传递给脚本的第一个参数（KEYS[1]）
		fieldReadCnt, 1).Err()       // 字段名，传递给脚本的第二个参数（ARGV[1]）
}

// IncrLikeCntIfPresent 增加指定业务（biz）和业务 ID（bizId）对应的 like 计数（字段名为 fieldLikeCnt）
// 如果该键存在，执行自增操作。
// 返回一个错误（如果有的话），否则返回 nil
func (r *RedisInteractiveCache) IncrLikeCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	// 使用 Eval 方法执行一个 Lua 脚本
	return r.client.Eval(
		ctx,                         // 上下文对象，用于控制请求的超时和取消
		luaIncrCnt,                  // Lua 脚本的内容
		[]string{r.key(biz, bizId)}, // 键名，传递给脚本的第一个参数（KEYS[1]）
		fieldLikeCnt,                // 字段名，传递给脚本的第二个参数（ARGV[1]）
		1,                           // 自增值（delta），传递给脚本的第三个参数（ARGV[2]）
	).Err() // 调用 Eval 返回的结果，提取并返回错误
}

// DecrLikeCntIfPresent 减少指定业务（biz）和业务 ID（bizId）对应的 like 计数（字段名为 fieldLikeCnt）
func (r *RedisInteractiveCache) DecrLikeCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	// 使用 Redis 脚本减少缓存中的点赞计数
	return r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldLikeCnt, -1).Err()
}

// IncrCollectCntIfPresent 如果缓存中存在该业务对象的收藏计数，则自增
func (r *RedisInteractiveCache) IncrCollectCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	// 调用 Redis 执行 Lua 脚本，增加缓存中的收藏计数
	return r.client.Eval(ctx, luaIncrCnt, // 使用 Lua 脚本来进行自增操作
		[]string{r.key(biz, bizId)}, // 键名，通过 biz 和 bizId 生成的 Redis 键
		fieldCollectCnt, 1).Err()    // 字段名，传递给脚本的第二个参数（ARGV[1]）
}

// Get 从 Redis 缓存中获取指定业务对象的互动数据（点赞数、收藏数、阅读数等）。
func (r *RedisInteractiveCache) Get(ctx context.Context, biz string, bizId int64) (domain.Interactive, error) {
	// 使用 HMGet 从 Redis 获取该业务对象的互动数据，获取指定字段（点赞数、收藏数、阅读数）
	data, err := r.client.HGetAll(ctx, r.key(biz, bizId)).Result()
	if err != nil {
		return domain.Interactive{}, err // 如果获取 Redis 数据失败，返回错误
	}

	// 如果数据为空，说明缓存中没有该业务对象的互动数据，返回一个错误
	if len(data) == 0 {
		return domain.Interactive{}, ErrKeyNotExist // 如果缓存不存在该键，返回指定的错误
	}

	// 从 Redis 获取的结果是字符串类型，需要将其转换为整型
	// 使用 strconv.ParseInt 将字段值转换为 int64 类型
	collectCnt, _ := strconv.ParseInt(data[fieldCollectCnt], 10, 64) // 收藏数
	likeCnt, _ := strconv.ParseInt(data[fieldLikeCnt], 10, 64)       // 点赞数
	readCnt, _ := strconv.ParseInt(data[fieldReadCnt], 10, 64)       // 阅读数

	// 返回包含互动数据的结构体（domain.Interactive），不需要返回错误
	return domain.Interactive{
		BizId:      bizId,
		CollectCnt: collectCnt, // 收藏数
		LikeCnt:    likeCnt,    // 点赞数
		ReadCnt:    readCnt,    // 阅读数
	}, err
}

// Set 设置指定业务对象的互动数据到 Redis 缓存中。
func (r *RedisInteractiveCache) Set(ctx context.Context, biz string, bizId int64, intr domain.Interactive) error {
	// 生成 Redis 键名，结合业务类型（biz）和业务对象 ID（bizId）
	key := r.key(biz, bizId)

	// 将互动数据（点赞数、收藏数、阅读数）设置到 Redis 哈希表中
	err := r.client.HMSet(ctx, key, // Redis 中设置哈希表的操作
		fieldLikeCnt, intr.LikeCnt, // 设置点赞数
		fieldCollectCnt, intr.CollectCnt, // 设置收藏数
		fieldReadCnt, intr.ReadCnt, // 设置阅读数
	).Err()

	if err != nil {
		return err // 如果设置失败，返回错误
	}

	// 设置 Redis 缓存的过期时间为 15 分钟
	return r.client.Expire(ctx, key, time.Minute*15).Err()
}

func (r *RedisInteractiveCache) key(biz string, bizId int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, bizId)
}
