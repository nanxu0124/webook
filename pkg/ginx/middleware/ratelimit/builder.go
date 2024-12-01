package ratelimit

import (
	_ "embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"net/http"
	"time"
)

// Builder 用于构建限流中间件
// 它包括配置前缀、Redis 客户端、限流间隔时间和阈值
type Builder struct {
	prefix   string        // 限流 key 的前缀，用于构建唯一的 Redis key
	cmd      redis.Cmdable // Redis 命令接口，提供执行 Redis 命令的方法
	interval time.Duration // 限流的时间间隔
	rate     int           // 限流的阈值，表示在指定的时间间隔内允许的最大请求次数
}

// luaScript 是通过 embed 嵌入的 Lua 脚本，
// 用于在 Redis 上执行限流逻辑（滑动窗口算法）
//
//go:embed slide_window.lua
var luaScript string

// NewBuilder 创建一个新的限流中间件构建器
// cmd 是 Redis 客户端，interval 是限流时间间隔，rate 是每个时间间隔允许的最大请求次数
func NewBuilder(cmd redis.Cmdable, interval time.Duration, rate int) *Builder {
	return &Builder{
		cmd:      cmd,
		prefix:   "ip-limiter", // 默认的 Redis 键前缀
		interval: interval,
		rate:     rate,
	}
}

// Prefix 设置限流键的前缀，允许根据需要自定义前缀
func (b *Builder) Prefix(prefix string) *Builder {
	b.prefix = prefix
	return b
}

// Build 创建并返回一个 Gin 中间件处理函数
// 这个函数会在每个请求中进行限流操作
func (b *Builder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 调用 limit 函数来检查请求是否被限流
		limited, err := b.limit(ctx)
		if err != nil {
			// 如果执行限流检查时发生错误，返回 500 系统错误
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if limited {
			// 如果被限流，返回 429 Too Many Requests 错误
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		// 如果没有被限流，继续执行请求的后续处理
		ctx.Next()
	}
}

// limit 是核心限流逻辑的实现
// 它会根据客户端的 IP 地址，在 Redis 中执行 Lua 脚本，判断该 IP 是否超过了限流阈值
func (b *Builder) limit(ctx *gin.Context) (bool, error) {
	// 使用客户端 IP 和指定的前缀来构建 Redis key
	key := fmt.Sprintf("%s:%s", b.prefix, ctx.ClientIP())
	// 执行 Lua 脚本，传递限流相关的参数
	return b.cmd.Eval(ctx, luaScript, []string{key},
		b.interval.Milliseconds(), b.rate, time.Now().UnixMilli()).Bool()
}
