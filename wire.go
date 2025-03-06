//go:build wireinject

package main

import (
	"github.com/google/wire"
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

func InitApp() *App {
	wire.Build(
		// 最基础的第三方依赖
		thirdProvider,

		// cron 部分
		rankServiceProvider,
		ioc.InitJobs,
		ioc.InitRankingJob,

		// DAO 部分
		dao.NewGormUserDAO,
		article.NewGORMArticleDAO,
		dao.NewGORMInteractiveDAO,

		// Cache 部分
		cache.NewRedisUserCache,
		cache.NewRedisCodeCache,
		cache.NewRedisArticleCache,
		cache.NewRedisInteractiveCache,

		// repository 部分
		repository.NewCachedUserRepository,
		repository.NewCachedCodeRepository,
		repository.NewArticleRepository,
		repository.NewCachedInteractiveRepository,

		// events 部分
		eventsArticle.NewKafkaProducer,
		//eventsArticle.NewInteractiveReadEventConsumer,
		eventsArticle.NewInteractiveReadEventBatchConsumer,
		ioc.NewConsumers,

		// service 部分
		service.NewUserService,
		service.NewSMSCodeService,
		service.NewArticleService,
		service.NewInteractiveService,
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
