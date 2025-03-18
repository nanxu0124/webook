//go:build wireinject

package main

import (
	"github.com/google/wire"
	"webook/interactive/events"
	"webook/interactive/grpc"
	"webook/interactive/ioc"
	"webook/interactive/repository"
	"webook/interactive/repository/cache"
	"webook/interactive/repository/dao"
	"webook/interactive/service"
)

// 第三方依赖
var thirdProvider = wire.NewSet(
	ioc.InitDB, ioc.InitRedis, ioc.InitLogger,
	ioc.InitKafka,
)

var interactiveSvcProvider = wire.NewSet(
	service.NewInteractiveService,
	repository.NewCachedInteractiveRepository,
	dao.NewGORMInteractiveDAO,
	cache.NewRedisInteractiveCache,
)

func Init() *App {
	wire.Build(
		thirdProvider,
		interactiveSvcProvider,
		grpc.NewInteractiveServiceServer,
		events.NewInteractiveReadEventBatchConsumer,
		ioc.InitGRPCxServer,
		ioc.NewConsumers,
		wire.Struct(new(App), "*"),
	)
	return new(App)
}
