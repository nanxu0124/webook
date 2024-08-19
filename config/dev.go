package config

var Config = config{
	DB: DBConfig{
		// 本地连接
		DSN: "root:root@tcp(localhost:13306)/webook",
	},
	Redis: RedisConfig{
		Addr: "localhost:6379",
	},
}
