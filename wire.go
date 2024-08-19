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

func initWebServer() *gin.Engine {
	wire.Build(
		// 最基础的第三方依赖
		ioc.InitDB, ioc.InitCache,

		dao.NewUserDAO,

		cache.NewRedisUserCache, cache.NewRedisCodeCache,

		repository.NewUserRepository,
		repository.NewCodeRepository,

		service.NewUserService,
		service.NewCodeService,
		ioc.InitSMSService,

		web.NewUserHandler,
		ijwt.NewRedisJWT,

		ioc.InitMiddlewares,
		ioc.InitWebServer,
	)

	return new(gin.Engine)
}
