// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package startup

import (
	"github.com/gin-gonic/gin"
	"webook/internal/repository"
	"webook/internal/repository/cache"
	"webook/internal/repository/dao"
	"webook/internal/repository/dao/article"
	"webook/internal/service"
	"webook/internal/web"
	"webook/internal/web/jwt"
	"webook/ioc"
)

// Injectors from wire.go:

func InitWebServer() *gin.Engine {
	cmdable := InitTestRedis()
	handler := jwt.NewRedisHandler(cmdable)
	logger := InitTestLogger()
	v := ioc.GinMiddlewares(cmdable, handler, logger)
	gormDB := InitTestDB()
	userDAO := dao.NewGormUserDAO(gormDB)
	userCache := cache.NewRedisUserCache(cmdable)
	userRepository := repository.NewCachedUserRepository(userDAO, userCache)
	userService := service.NewUserService(userRepository, logger)
	smsService := ioc.InitSmsService(cmdable)
	codeCache := cache.NewRedisCodeCache(cmdable)
	codeRepository := repository.NewCachedCodeRepository(codeCache)
	codeService := service.NewSMSCodeService(smsService, codeRepository, logger)
	userHandler := web.NewUserHandler(userService, codeService, handler)
	articleDAO := article.NewGORMArticleDAO(gormDB)
	articleRepository := repository.NewArticleRepository(articleDAO)
	articleService := service.NewArticleService(articleRepository)
	articleHandler := web.NewArticleHandler(articleService, logger)
	engine := ioc.InitWebServer(v, userHandler, articleHandler)
	return engine
}

func InitArticleHandler(dao2 article.ArticleDAO) *web.ArticleHandler {
	articleRepository := repository.NewArticleRepository(dao2)
	articleService := service.NewArticleService(articleRepository)
	logger := InitTestLogger()
	articleHandler := web.NewArticleHandler(articleService, logger)
	return articleHandler
}