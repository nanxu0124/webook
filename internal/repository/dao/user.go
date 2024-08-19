package dao

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

var (
	ErrUserDuplicate = errors.New("用户冲突")
	ErrUserNotFound  = gorm.ErrRecordNotFound
)

type UserDAO interface {
	Insert(ctx context.Context, u User) error
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByPhone(ctx context.Context, phone string) (User, error)
	UpdateById(ctx context.Context, u User) error
	FindById(ctx context.Context, userId int64) (User, error)
	FindByWechat(ctx context.Context, openId string) (User, error)
}

type GormUserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) UserDAO {
	return &GormUserDAO{
		db: db,
	}
}

// User 直接对应数据库表结构
type User struct {
	Id       int64          `gorm:"primaryKey,autoIncrement"`
	Email    sql.NullString `gorm:"unique"`
	Password string
	Phone    sql.NullString `gorm:"unique"`

	WechatOpenId  sql.NullString `gorm:"unique"`
	WechatUnionId sql.NullString

	Ctime int64
	Utime int64

	Nickname        string
	BirthDay        string
	PersonalProfile string
}

func (dao *GormUserDAO) Insert(ctx context.Context, u User) error {
	now := time.Now().UnixMilli()
	u.Utime = now
	u.Ctime = now

	err := dao.db.WithContext(ctx).Create(&u).Error
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		const uniqueConflicts uint16 = 1062
		if mysqlErr.Number == uniqueConflicts { // 先查询会有并发问题
			return ErrUserDuplicate
		}
	}
	return nil
}

func (dao *GormUserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	return u, err
}

func (dao *GormUserDAO) FindByPhone(ctx context.Context, phone string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).Where("phone = ?", phone).First(&u).Error
	return u, err
}

func (dao *GormUserDAO) UpdateById(ctx context.Context, u User) error {
	now := time.Now().UnixMilli()
	u.Utime = now

	var user User
	err := dao.db.Model(&user).WithContext(ctx).Where("id = ?", u.Id).Updates(map[string]interface{}{
		"utime":            u.Utime,
		"nickname":         u.Nickname,
		"birth_day":        u.BirthDay,
		"personal_profile": u.PersonalProfile,
	}).Error

	return err
}

func (dao *GormUserDAO) FindById(ctx context.Context, userId int64) (User, error) {
	var user User
	err := dao.db.WithContext(ctx).Where("id = ?", userId).First(&user).Error
	return user, err
}

func (dao *GormUserDAO) FindByWechat(ctx context.Context, openId string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).Where("wechat_open_id = ?", openId).First(&u).Error
	return u, err
}
