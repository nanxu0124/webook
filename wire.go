//go:build wireinject

package main

import (
	"github.com/google/wire"
	events2 "webook/interactive/events"
	repository2 "webook/interactive/repository"
	cache2 "webook/interactive/repository/cache"
	dao2 "webook/interactive/repository/dao"
	service2 "webook/interactive/service"
	eventsArticle "webook/internal/events/article"
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
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitLogger,
	ioc.InitKafka,
	ioc.NewSyncProducer,
)

var rankServiceProvider = wire.NewSet(
	service.NewBatchRankingService,
	repository.NewCachedRankingRepository,
	cache.NewRedisRankingCache,
	cache.NewRankingLocalCache,
)

// 这一部分是用作本地 interactive 服务
// 防止切换服务的时候出问题
var interactiveServiceProducer = wire.NewSet(
	dao2.NewGORMInteractiveDAO,
	cache2.NewRedisInteractiveCache,
	repository2.NewCachedInteractiveRepository,
	service2.NewInteractiveService,
	events2.NewInteractiveReadEventBatchConsumer,
)

func InitApp() *App {
	wire.Build(
		// 最基础的第三方依赖
		thirdProvider,

		// cron 部分
		rankServiceProvider,
		ioc.InitJobs,
		ioc.InitRankingJob,

		// 微服务部分
		interactiveServiceProducer,
		ioc.InitIntrGRPCClient,

		// DAO 部分
		dao.NewGormUserDAO,
		article.NewGORMArticleDAO,

		// Cache 部分
		cache.NewRedisUserCache,
		cache.NewRedisCodeCache,
		cache.NewRedisArticleCache,

		// repository 部分
		repository.NewCachedUserRepository,
		repository.NewCachedCodeRepository,
		repository.NewArticleRepository,

		// events 部分
		eventsArticle.NewKafkaProducer,
		ioc.NewConsumers,

		// service 部分
		service.NewUserService,
		service.NewSMSCodeService,
		service.NewArticleService,
		ioc.InitSmsService,

		// handler 部分
		ijwt.NewRedisHandler,
		web.NewUserHandler,
		web.NewArticleHandler,

		// gin 的中间件
		ioc.GinMiddlewares,

		// Web 服务器
		ioc.InitWebServer,

		wire.Struct(new(App), "*"),
	)

	return new(App)
}
