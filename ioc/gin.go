package ioc

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
	"webook/internal/web"
	"webook/internal/web/middleware"
	"webook/pkg/ginx/middleware/ratelimit"
)

func InitWebServer(funcs []gin.HandlerFunc, userHdl *web.UserHandler) *gin.Engine {
	server := gin.Default()
	gin.ForceConsoleColor()
	server.Use(funcs...)
	// 注册路由
	userHdl.RegisterRoutes(server)
	return server
}

func GinMiddlewares(cmd redis.Cmdable) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		ratelimit.NewBuilder(cmd, time.Minute, 100).Build(),
		corsHandler(),

		// 使用 JWT
		middleware.NewJWTLoginMiddlewareBuilder().Build(),
	}
}

func corsHandler() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowCredentials: true,                                      // 允许客户端发送认证信息
		AllowHeaders:     []string{"Content-Type", "Authorization"}, // 允许的请求头
		ExposeHeaders:    []string{"X-Jwt-Token", "X-Refresh-Token"},
		AllowOriginFunc: func(origin string) bool {
			// 允许来自 localhost 和指定公司域名的请求
			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			return strings.Contains(origin, "baidu.com")
		},
		MaxAge: 12 * time.Hour, // 预检请求的缓存时间，12小时内不会再进行预检
	})
}
