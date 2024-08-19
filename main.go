package main

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {

	initViper()
	initLogger()

	engine := initWebServer()

	engine.Run(":8080")
}

func initLogger() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	// 如果这里不replace，zap.L().Info() 啥都打印不出来
	zap.ReplaceGlobals(logger)
	zap.L().Info("hello, logger 启动了")
}

func initViper() {
	viper.SetConfigFile("config/dev.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}
