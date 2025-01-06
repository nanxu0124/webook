package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"net/http"
	"time"
)

func main() {
	// 当前目录的配置
	initViperRemote()
	initPrometheus()
	app := InitApp()
	for _, c := range app.consumers {
		err := c.Start()
		if err != nil {
			panic(err)
		}
	}
	server := app.web
	// 注册路由
	server.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello, world")
	})
	server.Run("localhost:8080")
}

func initViper() {
	cfile := pflag.String("config",
		"config/dev.yaml", "配置文件路径")
	pflag.Parse()
	// 直接指定文件路径
	viper.SetConfigFile(*cfile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

func initViperRemote() {
	err := viper.AddRemoteProvider("etcd3",
		"http://127.0.0.1:12379", "/config")
	if err != nil {
		panic(err)
	}
	viper.SetConfigType("yaml")
	err = viper.ReadRemoteConfig()
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			err = viper.WatchRemoteConfig()
			if err != nil {
				fmt.Println(err)
				return
			}
			time.Sleep(time.Second) // 睡个一秒钟
		}
	}()
}
