//go:build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"webook/internal/repository"
	"webook/internal/repository/cache"
	"webook/internal/repository/dao"
	"webook/internal/service"
	"webook/internal/web"
	ijwt "webook/internal/web/jwt"
	"webook/ioc"
)

func InitWebServer() *gin.Engine {
	wire.Build(
		// 最基础的第三方依赖
		ioc.InitDB, ioc.InitRedis,

		dao.NewGormUserDAO,

		cache.NewRedisUserCache, cache.NewRedisCodeCache,

		repository.NewCachedUserRepository,
		repository.NewCachedCodeRepository,

		service.NewUserService,
		service.NewSMSCodeService,
		ioc.InitSmsService,

		ijwt.NewRedisHandler,
		web.NewUserHandler,
		ioc.GinMiddlewares,
		ioc.InitWebServer,
		ioc.InitLogger,
	)

	return new(gin.Engine)
}
