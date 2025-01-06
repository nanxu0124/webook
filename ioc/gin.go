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
	"webook/pkg/ginx/middleware/metrics"
	"webook/pkg/logger"
)

func InitWebServer(funcs []gin.HandlerFunc, userHdl *web.UserHandler, artHdl *web.ArticleHandler) *gin.Engine {
	server := gin.Default() // 初始化一个默认的 Gin 引擎实例
	gin.ForceConsoleColor() // 强制开启控制台的彩色输出

	// 使用传入的中间件
	server.Use(funcs...)

	// 注册用户相关的路由
	userHdl.RegisterRoutes(server)
	artHdl.RegisterRoutes(server)

	return server // 返回配置好的 Gin 引擎实例
}

func GinMiddlewares(cmd redis.Cmdable, hdl ijwt.Handler, l logger.Logger) []gin.HandlerFunc {

	pb := &metrics.PrometheusBuilder{
		Namespace:  "webook_server",
		Subsystem:  "webook",
		Name:       "gin_http",
		InstanceID: "my-instance-1",
		Help:       "GIN 中 HTTP 请求",
	}

	return []gin.HandlerFunc{
		// 限流
		//ratelimit.NewBuilder(cmd, time.Minute, 100).Build(), // 限制每分钟最多 100 次请求

		// 跨域
		corsHandler(), // 配置 CORS 中间件

		// prometheus 中间件
		pb.BuildResponseTime(),
		pb.BuildActiveRequest(),

		// 使用 JWT 中间件
		middleware.NewJWTLoginMiddlewareBuilder(hdl).Build(),

		// 访问日志中间件
		//accesslog.NewMiddlewareBuilder(func(ctx context.Context, al accesslog.AccessLog) {
		//	// 设置为 DEBUG 级别
		//	l.Debug("GIN 收到请求", logger.Field{
		//		Key:   "req",
		//		Value: al,
		//	})
		//}).AllowReqBody().AllowRespBody().Build(),
	}
}

func corsHandler() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowCredentials: true,                                       // 允许客户端发送认证信息
		AllowHeaders:     []string{"Content-Type", "Authorization"},  // 允许的请求头
		ExposeHeaders:    []string{"X-Jwt-Token", "X-Refresh-Token"}, // 暴露的响应头
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
