package ioc

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/plugin/prometheus"
	"time"
	"webook/interactive/repository/dao"
	prometheus2 "webook/pkg/gormx/callbacks/prometheus"
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

	// 获取底层的 *sql.DB 对象
	sqlDB, err := db.DB()
	if err != nil {
		// 错误处理
		panic("failed to get DB instance")
	}
	sqlDB.SetMaxOpenConns(100)                 // 设置最大打开连接数
	sqlDB.SetMaxIdleConns(30)                  // 设置最大空闲连接数
	sqlDB.SetConnMaxLifetime(time.Minute * 30) // 设置连接的最大生命周期

	// 接入 prometheus
	err = db.Use(prometheus.New(prometheus.Config{
		DBName: "webook",
		// 每 15 秒采集一些数据
		RefreshInterval: 15,
		MetricsCollector: []prometheus.MetricsCollector{
			&prometheus.MySQL{
				VariableNames: []string{"Threads_running"},
			},
		}, // user defined metrics
	}))
	if err != nil {
		panic(err)
	}

	// 接入回调
	prom := prometheus2.Callbacks{
		Namespace:  "webook_interactive_server",
		Subsystem:  "webook_interactive",
		Name:       "gorm",
		InstanceID: "my-instance-1",
		Help:       "gorm DB 查询",
	}
	err = prom.Register(db)
	if err != nil {
		panic(err)
	}

	// 初始化数据库表结构
	err = dao.InitTables(db)
	if err != nil {
		panic(err) // 初始化表失败时，抛出 panic
	}

	return db // 返回数据库连接对象
}

func InitTiDB(l logger.Logger) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&tls=%s",
		"root", "", "127.0.0.1", "4001", "webook", "false")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		//Logger: glogger.New(gormLoggerFunc(l.Debug), // 自定义日志记录
		//	glogger.Config{
		//		SlowThreshold: 0,            // 不记录慢查询
		//		LogLevel:      glogger.Info, // 设置日志等级为 Info
		//	}),
	})
	if err != nil {
		panic(err)
	}
	// 初始化数据库表结构
	err = dao.InitTables(db)
	if err != nil {
		panic(err) // 初始化表失败时，抛出 panic
	}
	return db
}

type gormLoggerFunc func(msg string, fields ...logger.Field)

func (g gormLoggerFunc) Printf(msg string, args ...interface{}) {
	g(msg, logger.Field{Key: "args", Value: args})
}
