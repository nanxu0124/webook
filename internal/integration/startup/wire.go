//go:build wireinject

package startup

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"webook/internal/repository"
	"webook/internal/repository/cache"
	"webook/internal/repository/dao"
	"webook/internal/repository/dao/article"
	"webook/internal/service"
	"webook/internal/web"
	ijwt "webook/internal/web/jwt"
	"webook/ioc"
)

func InitWebServer() *gin.Engine {
	wire.Build(
		// 最基础的第三方依赖
		InitTestDB, InitTestRedis, InitTestLogger,

		dao.NewGormUserDAO,
		article.NewGORMArticleDAO,

		cache.NewRedisUserCache, cache.NewRedisCodeCache, cache.NewRedisArticleCache,

		repository.NewCachedUserRepository,
		repository.NewCachedCodeRepository,
		repository.NewArticleRepository,

		service.NewUserService,
		service.NewSMSCodeService,
		service.NewArticleService,
		ioc.InitSmsService,

		ijwt.NewRedisHandler,
		web.NewUserHandler,
		web.NewArticleHandler,
		ioc.GinMiddlewares,
		ioc.InitWebServer,
	)
	// 随便返回一个
	return gin.Default()
}

func InitArticleHandler(articleDao article.ArticleDAO) *web.ArticleHandler {
	wire.Build(
		InitTestDB, InitTestRedis, InitTestLogger,
		dao.NewGormUserDAO,
		cache.NewRedisUserCache,
		cache.NewRedisArticleCache,
		repository.NewCachedUserRepository,
		repository.NewArticleRepository,
		service.NewArticleService,
		web.NewArticleHandler,
	)
	return new(web.ArticleHandler)
}
