package ioc

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"webook/internal/repository/dao"
	"webook/pkg/logger"
)

func InitDB(l logger.Logger) *gorm.DB {
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	c := Config{
		DSN: "root:root@tcp(localhost:3306)/mysql", // 默认的数据库连接字符串
	}

	// 使用 viper 从配置文件中读取 db 配置
	err := viper.UnmarshalKey("db", &c)
	if err != nil {
		panic(fmt.Errorf("初始化配置失败 %v, 原因 %w", c, err))
	}

	// 使用 GORM 打开数据库连接
	db, err := gorm.Open(mysql.Open(c.DSN), &gorm.Config{
		Logger: glogger.New(gormLoggerFunc(l.Debug), // 自定义日志记录
			glogger.Config{
				SlowThreshold: 0,            // 不记录慢查询
				LogLevel:      glogger.Info, // 设置日志等级为 Info
			}),
	})
	if err != nil {
		panic(err) // 打开数据库失败时，抛出 panic
	}

	// 初始化数据库表结构
	err = dao.InitTables(db)
	if err != nil {
		panic(err) // 初始化表失败时，抛出 panic
	}

	return db // 返回数据库连接对象
}

type gormLoggerFunc func(msg string, fields ...logger.Field)

func (g gormLoggerFunc) Printf(msg string, args ...interface{}) {
	g(msg, logger.Field{Key: "args", Value: args})
}
