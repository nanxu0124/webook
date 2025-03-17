// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/google/wire"
	"webook/interactive/events"
	repository2 "webook/interactive/repository"
	cache2 "webook/interactive/repository/cache"
	dao2 "webook/interactive/repository/dao"
	service2 "webook/interactive/service"
	article2 "webook/internal/events/article"
	"webook/internal/repository"
	"webook/internal/repository/cache"
	"webook/internal/repository/dao"
	"webook/internal/repository/dao/article"
	"webook/internal/service"
	"webook/internal/web"
	"webook/internal/web/jwt"
	"webook/ioc"
)

import (
	_ "github.com/spf13/viper/remote"
)

// Injectors from wire.go:

func InitApp() *App {
	cmdable := ioc.InitRedis()
	handler := jwt.NewRedisHandler(cmdable)
	logger := ioc.InitLogger()
	v := ioc.GinMiddlewares(cmdable, handler, logger)
	db := ioc.InitDB(logger)
	userDAO := dao.NewGormUserDAO(db)
	userCache := cache.NewRedisUserCache(cmdable)
	userRepository := repository.NewCachedUserRepository(userDAO, userCache)
	userService := service.NewUserService(userRepository, logger)
	smsService := ioc.InitSmsService(cmdable)
	codeCache := cache.NewRedisCodeCache(cmdable)
	codeRepository := repository.NewCachedCodeRepository(codeCache)
	codeService := service.NewSMSCodeService(smsService, codeRepository, logger)
	userHandler := web.NewUserHandler(userService, codeService, handler)
	articleDAO := article.NewGORMArticleDAO(db)
	articleCache := cache.NewRedisArticleCache(cmdable)
	articleRepository := repository.NewArticleRepository(articleDAO, userRepository, articleCache, logger)
	client := ioc.InitKafka()
	syncProducer := ioc.NewSyncProducer(client)
	producer := article2.NewKafkaProducer(syncProducer)
	articleService := service.NewArticleService(articleRepository, logger, producer)
	interactiveDAO := dao2.NewGORMInteractiveDAO(db)
	interactiveCache := cache2.NewRedisInteractiveCache(cmdable)
	interactiveRepository := repository2.NewCachedInteractiveRepository(interactiveDAO, interactiveCache, logger)
	interactiveService := service2.NewInteractiveService(interactiveRepository, logger)
	articleHandler := web.NewArticleHandler(articleService, interactiveService, logger)
	engine := ioc.InitWebServer(v, userHandler, articleHandler)
	interactiveReadEventBatchConsumer := events.NewInteractiveReadEventBatchConsumer(client, logger, interactiveRepository)
	v2 := ioc.NewConsumers(interactiveReadEventBatchConsumer)
	redisRankingCache := cache.NewRedisRankingCache(cmdable)
	rankingLocalCache := cache.NewRankingLocalCache()
	rankingRepository := repository.NewCachedRankingRepository(redisRankingCache, rankingLocalCache)
	rankingService := service.NewBatchRankingService(interactiveService, articleService, rankingRepository)
	rankingJob := ioc.InitRankingJob(rankingService, logger)
	cron := ioc.InitJobs(logger, rankingJob)
	app := &App{
		web:       engine,
		consumers: v2,
		cron:      cron,
	}
	return app
}

// wire.go:

// 第三方依赖
var thirdProvider = wire.NewSet(ioc.InitDB, ioc.InitRedis, ioc.InitLogger, ioc.InitKafka, ioc.NewSyncProducer)

var rankServiceProvider = wire.NewSet(service.NewBatchRankingService, repository.NewCachedRankingRepository, cache.NewRedisRankingCache, cache.NewRankingLocalCache)
