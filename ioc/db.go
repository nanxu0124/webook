package ioc

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"webook/internal/repository/dao"
)

func InitDB() *gorm.DB {
	type config struct {
		Dsn string `yaml:"dsn"`
	}
	var c config
	err := viper.UnmarshalKey("mysql", &c)
	if err != nil {
		panic(fmt.Errorf("初始化配置失败 %v", err))
	}

	db, err := gorm.Open(mysql.Open(c.Dsn))
	if err != nil {
		panic(err)
	}
	err = dao.INitTable(db)
	if err != nil {
		panic(err)
	}
	return db
}
