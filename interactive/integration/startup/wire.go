//go:build wireinject

package startup

import (
	"github.com/google/wire"
	repository2 "webook/interactive/repository"
	cache2 "webook/interactive/repository/cache"
	dao2 "webook/interactive/repository/dao"
	service2 "webook/interactive/service"
)

// 第三方依赖
var thirdProvider = wire.NewSet(
	InitTestDB, InitTestRedis, InitTestLogger,
	InitKafka, NewSyncProducer,
)

var interactiveSvcProvider = wire.NewSet(
	service2.NewInteractiveService,
	repository2.NewCachedInteractiveRepository,
	dao2.NewGORMInteractiveDAO,
	cache2.NewRedisInteractiveCache,
)

func InitInteractiveService() service2.InteractiveService {
	wire.Build(
		thirdProvider,
		interactiveSvcProvider)
	return service2.NewInteractiveService(nil, nil)
}
