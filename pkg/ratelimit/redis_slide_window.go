package ratelimit

import (
	_ "embed"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
	"time"
)

//go:embed lua/slide_window.lua
var luaScript string

type RedisSlidingWindowLimiter struct {
	cmd      redis.Cmdable
	interval time.Duration
	// 阈值
	rate int
}

func NewRedisSlidingWindowLimiter(cmd redis.Cmdable, interval time.Duration, rate int) *RedisSlidingWindowLimiter {
	return &RedisSlidingWindowLimiter{
		cmd:      cmd,
		interval: interval,
		rate:     rate,
	}
}

func (r *RedisSlidingWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
	return r.cmd.Eval(ctx, luaScript, []string{key},
		r.interval.Milliseconds(),
		r.rate, time.Now().UnixMilli()).Bool()
}
