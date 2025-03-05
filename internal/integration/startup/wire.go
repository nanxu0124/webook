//go:build wireinject

package startup

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"time"
	article2 "webook/internal/events/article"
	"webook/internal/repository"
	"webook/internal/repository/cache"
	"webook/internal/repository/dao"
	"webook/internal/repository/dao/article"
	"webook/internal/service"
	"webook/internal/web"
	ijwt "webook/internal/web/jwt"
	"webook/ioc"
)

// 第三方依赖
var thirdProvider = wire.NewSet(
	InitTestDB, InitTestRedis, InitTestLogger,
	InitKafka, NewSyncProducer,
)

var userSvcProvider = wire.NewSet(
	dao.NewGormUserDAO,
	cache.NewRedisUserCache,
	repository.NewCachedUserRepository,
	service.NewUserService)

var articlSvcProvider = wire.NewSet(
	article.NewGORMArticleDAO,
	article2.NewKafkaProducer,
	cache.NewRedisArticleCache,
	repository.NewArticleRepository,
	service.NewArticleService)

var interactiveSvcProvider = wire.NewSet(
	service.NewInteractiveService,
	repository.NewCachedInteractiveRepository,
	dao.NewGORMInteractiveDAO,
	cache.NewRedisInteractiveCache,
)

var rankServiceProvider = wire.NewSet(
	service.NewBatchRankingService,
	repository.NewCachedRankingRepository,
	cache.NewRedisRankingCache,
)

//go:generate wire
func InitWebServer() *gin.Engine {
	wire.Build(
		thirdProvider,
		userSvcProvider,
		articlSvcProvider,
		interactiveSvcProvider,
		cache.NewRedisCodeCache,
		repository.NewCachedCodeRepository,
		// service 部分
		// 集成测试我们显式指定使用内存实现
		ioc.InitSmsService,

		service.NewSMSCodeService,
		// handler 部分
		web.NewUserHandler,
		web.NewArticleHandler,

		ijwt.NewRedisHandler,

		// gin 的中间件
		ioc.GinMiddlewares,

		// Web 服务器
		ioc.InitWebServer,
	)
	// 随便返回一个
	return gin.Default()
}

func InitArticleHandler(dao article.ArticleDAO) *web.ArticleHandler {
	wire.Build(
		thirdProvider,
		userSvcProvider,
		interactiveSvcProvider,
		article2.NewKafkaProducer,
		cache.NewRedisArticleCache,
		repository.NewArticleRepository,
		service.NewArticleService,
		web.NewArticleHandler)
	return new(web.ArticleHandler)
}

func InitUserSvc() service.UserService {
	wire.Build(
		thirdProvider,
		userSvcProvider)
	return service.NewUserService(nil, nil)
}

func InitRankingService(expiration time.Duration) service.RankingService {
	wire.Build(
		thirdProvider,
		interactiveSvcProvider,
		articlSvcProvider,
		// 用不上这个 user repo，所以随便搞一个
		wire.InterfaceValue(
			new(repository.UserRepository),
			&repository.CachedUserRepository{}),

		rankServiceProvider)
	return &service.BatchRankingService{}
}

func InitInteractiveService() service.InteractiveService {
	wire.Build(
		thirdProvider,
		interactiveSvcProvider)
	return service.NewInteractiveService(nil, nil)
}

func InitJwtHdl() ijwt.Handler {
	wire.Build(
		thirdProvider,
		ijwt.NewRedisHandler)
	return ijwt.NewRedisHandler(nil)
}
