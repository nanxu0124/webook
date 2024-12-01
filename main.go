package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"strings"
	"time"
	"webook/internal/repository"
	"webook/internal/repository/dao"
	"webook/internal/service"
	"webook/internal/web"
	"webook/internal/web/middleware"
	"webook/pkg/ginx/middleware/ratelimit"
)

func main() {
	// 初始化数据库连接
	db := initDB()

	// 初始化Web服务器
	server := initWebServer()

	// 初始化用户相关的服务、路由等
	initUser(server, db)

	// 启动Web服务器，监听8080端口
	server.Run(":8080")
}

// initDB 初始化数据库连接和表
func initDB() *gorm.DB {
	// 使用GORM的MySQL驱动，连接到数据库
	db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:13316)/webook"))
	if err != nil {
		// 如果连接失败，panic并输出错误信息
		panic(err)
	}

	// 初始化数据库表
	err = dao.InitTables(db)
	if err != nil {
		// 如果表初始化失败，panic并输出错误信息
		panic(err)
	}

	// 返回数据库连接实例
	return db
}

// initWebServer 初始化Web服务器和中间件
func initWebServer() *gin.Engine {
	// 创建Gin默认的Web服务器
	server := gin.Default()
	gin.ForceConsoleColor()

	cmd := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})
	// 使用滑动窗口限流中间件，设置时间窗口为 1 分钟，最大请求次数为 rate 次
	// 这会限制每个客户端在 1 分钟内最多只能发送 rate 次请求
	// 使用 redis 客户端创建限流中间件实例，并将其应用到 Web 服务器
	server.Use(ratelimit.NewBuilder(cmd, time.Minute, 10).Build())

	// 使用 CORS 中间件来允许跨域请求
	// 配置允许客户端发送认证信息和特定请求头，同时控制允许的源（Origin）
	server.Use(cors.New(cors.Config{
		AllowCredentials: true,                                      // 允许客户端发送认证信息
		AllowHeaders:     []string{"Content-Type", "Authorization"}, // 允许的请求头
		ExposeHeaders:    []string{"X-Jwt-Token"},
		AllowOriginFunc: func(origin string) bool {
			// 允许来自 localhost 和指定公司域名的请求
			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			return strings.Contains(origin, "baidu.com")
		},
		MaxAge: 12 * time.Hour, // 预检请求的缓存时间，12小时内不会再进行预检
	}))

	// 使用 JWT 认证中间件
	// 这个中间件会对每个请求进行验证，确保请求带有有效的 JWT token
	// 如果没有 token 或 token 无效，请求会被中止
	usingJWT(server)

	// 返回配置好的Web服务器实例
	return server
}

// usingJWT 用于设置并启用JWT中间件
// 它会在所有请求中应用JWT中间件，确保每个请求在访问需要认证的接口时
// 都会检查JWT的有效性
// 该函数会把JWT中间件（由JWTLoginMiddlewareBuilder构建）应用到传入的Gin服务器实例上
func usingJWT(server *gin.Engine) {
	// 创建一个JWTLoginMiddlewareBuilder实例，用于构建JWT中间件
	mldBd := &middleware.JWTLoginMiddlewareBuilder{}

	// 将JWT中间件添加到Gin的中间件链中
	// 这意味着所有请求都会经过JWT中间件，除非被显式排除（例如在登录和注册接口中）
	server.Use(mldBd.Build())
}

// initUser 初始化与用户相关的服务和路由
func initUser(server *gin.Engine, db *gorm.DB) {
	// 创建UserDAO实例，传入数据库连接
	ud := dao.NewUserDAO(db)

	// 创建UserRepository实例，传入UserDAO
	ur := repository.NewUserRepository(ud)

	// 创建UserService实例，传入UserRepository
	us := service.NewUserService(ur)

	// 创建UserHandler实例，传入UserService
	c := web.NewUserHandler(us)

	// 注册与用户相关的路由
	c.RegisterRoutes(server)
}
