package ioc

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
	"webook/internal/web"
	ijwt "webook/internal/web/jwt"
	"webook/internal/web/middleware"
	iplimit "webook/pkg/ginx/middleware/ratelimit"
	"webook/pkg/ratelimit"
)

func InitWebServer(midls []gin.HandlerFunc, userHdl *web.UserHandler) *gin.Engine {
	server := gin.Default()
	server.Use(midls...)
	userHdl.RegisterRoutes(server)
	return server
}

func InitMiddlewares(cmd redis.Cmdable, jwtHdl ijwt.Handler) []gin.HandlerFunc {
	return []gin.HandlerFunc{

		// 跨域
		cors.New(cors.Config{
			// : []string{"http://localhost:3000"},
			//AllowMethods: []string{"POST", "GET"},
			AllowHeaders:     []string{"Content-Type", "authorization"},
			ExposeHeaders:    []string{"x-jwt-token", "x-refresh-token"},
			AllowCredentials: true,
			AllowOriginFunc: func(origin string) bool {
				if strings.HasPrefix(origin, "http://localhost") {
					return true
				}
				return strings.Contains(origin, "company.com")
			},
			MaxAge: time.Hour,
		}),

		// IP限流
		initIPLimiter(cmd),

		// ignore path
		middleware.NewLoginJWTMiddlewareBuilder(jwtHdl).
			IgnorePaths("/users/login").
			IgnorePaths("/users/signup").
			IgnorePaths("/users/login_sms/code/send").
			IgnorePaths("/users/login_sms").
			IgnorePaths("/users/refresh_token").
			Build(),
	}
}

func initIPLimiter(cmd redis.Cmdable) gin.HandlerFunc {
	return iplimit.NewBuilder(ratelimit.NewRedisSlidingWindowLimiter(cmd, time.Minute, 100)).Build()
}
