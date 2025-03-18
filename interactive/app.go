package main

import (
	"webook/pkg/grpcx"
	"webook/pkg/saramax"
)

type App struct {
	// 所有需要 main 函数控制启动、关闭的都要在这里
	server    *grpcx.Server
	consumers []saramax.Consumer
}
