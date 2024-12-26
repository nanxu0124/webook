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
		dao.NewGORMInteractiveDAO,

		cache.NewRedisUserCache, cache.NewRedisCodeCache, cache.NewRedisArticleCache,
		cache.NewRedisInteractiveCache,

		repository.NewCachedUserRepository,
		repository.NewCachedCodeRepository,
		repository.NewArticleRepository,
		repository.NewCachedInteractiveRepository,

		service.NewUserService,
		service.NewSMSCodeService,
		service.NewArticleService,
		ioc.InitSmsService,
		service.NewInteractiveService,

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
		dao.NewGORMInteractiveDAO,
		cache.NewRedisUserCache,
		cache.NewRedisArticleCache,
		cache.NewRedisInteractiveCache,
		repository.NewCachedUserRepository,
		repository.NewArticleRepository,
		repository.NewCachedInteractiveRepository,
		service.NewArticleService,
		service.NewInteractiveService,
		web.NewArticleHandler,
	)
	return new(web.ArticleHandler)
}

func InitInteractiveService() service.InteractiveService {
	wire.Build(
		InitTestDB, InitTestRedis, InitTestLogger,
		service.NewInteractiveService,
		repository.NewCachedInteractiveRepository,
		dao.NewGORMInteractiveDAO,
		cache.NewRedisInteractiveCache,
	)
	return service.NewInteractiveService(nil, nil)
}
