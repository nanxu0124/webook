package main

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"net/http"
)

func main() {
	initViperV2Watch()
	initPrometheus()

	app := InitApp()
	for _, c := range app.consumers {
		err := c.Start()
		if err != nil {
			panic(err)
		}
	}
	app.cron.Start()
	// 停掉所有的 jobs
	defer func() {
		ctx := app.cron.Stop()
		<-ctx.Done()
	}()

	server := app.web
	//注册路由
	server.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello, world")
	})
	server.Run(":8080")
}

func initViperV2Watch() {
	cfile := pflag.String("config", "config/dev.yaml", "配置文件路径")
	pflag.Parse()
	// 直接指定文件路径
	viper.SetConfigFile(*cfile)
	viper.WatchConfig()
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}
