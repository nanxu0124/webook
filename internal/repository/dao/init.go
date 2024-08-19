package dao

import "gorm.io/gorm"

func INitTable(db *gorm.DB) error {
	return db.AutoMigrate(&User{})
}
