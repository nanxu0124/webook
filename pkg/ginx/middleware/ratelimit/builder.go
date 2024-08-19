package ratelimit

import (
	_ "embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"webook/pkg/ratelimit"
)

type Builder struct {
	prefix  string
	limiter ratelimit.Limiter
}

func NewBuilder(limiter ratelimit.Limiter) *Builder {
	return &Builder{
		prefix:  "ip-limiter",
		limiter: limiter,
	}
}

func (b *Builder) Prefix(prefix string) *Builder {
	b.prefix = prefix
	return b
}

func (b *Builder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limited, err := b.limit(ctx)
		if err != nil {
			// 这一步很有意思，就是如果这边出错了
			// 要怎么办？
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if limited {
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		ctx.Next()
	}
}

func (b *Builder) limit(ctx *gin.Context) (bool, error) {
	key := fmt.Sprintf("%s:%s", b.prefix, ctx.ClientIP())
	return b.limiter.Limit(ctx, key)
}
