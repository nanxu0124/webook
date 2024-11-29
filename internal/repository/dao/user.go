package dao

import (
	"context"
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

// ErrUserDuplicateEmail 表示用户邮箱冲突错误
var ErrUserDuplicateEmail = errors.New("邮件冲突")

// ErrDataNotFound 通用的数据没找到错误（即Gorm的记录未找到）
var ErrDataNotFound = gorm.ErrRecordNotFound

// UserDAO 是与用户相关的数据访问对象，它封装了与用户数据表交互的所有操作
type UserDAO struct {
	db *gorm.DB // Gorm DB 实例，用于与数据库交互
}

// NewUserDAO 创建并返回一个新的 UserDAO 实例
// 参数 db 是已经初始化好的 Gorm DB 实例
func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		db: db,
	}
}

// Insert 将用户数据插入到数据库中
// 如果出现唯一约束冲突（例如邮箱重复），则返回自定义的 ErrUserDuplicateEmail 错误
func (ud *UserDAO) Insert(ctx context.Context, u User) error {
	// 获取当前时间戳，用于设置用户的创建时间和更新时间
	now := time.Now().UnixMilli()
	u.Ctime = now
	u.Utime = now

	// 使用Gorm的Create方法将用户数据插入到数据库
	// 如果插入时出现错误，检查是否是由于邮箱唯一索引冲突导致的
	err := ud.db.WithContext(ctx).Create(&u).Error
	if me, ok := err.(*mysql.MySQLError); ok {
		const uniqueIndexErrNo uint16 = 1062 // 唯一索引冲突错误码
		if me.Number == uniqueIndexErrNo {
			// 如果是唯一索引冲突，返回自定义的 ErrUserDuplicateEmail 错误
			return ErrUserDuplicateEmail
		}
	}
	return err // 如果是其他错误，直接返回
}

// FindByEmail 根据用户邮箱查找用户
// 如果用户存在，返回用户数据；如果没有找到用户，返回 ErrDataNotFound 错误
func (ud *UserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).First(&u, "email = ?", email).Error
	return u, err
}

// FindById 根据用户ID查找用户
// 如果用户存在，返回用户数据；如果没有找到用户，返回 ErrDataNotFound 错误
func (ud *UserDAO) FindById(ctx context.Context, id int64) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).First(&u, "id = ?", id).Error
	return u, err
}

// User 表示用户的数据模型，映射到数据库中的用户表
// 通过Gorm的标签来定义字段属性，比如主键、唯一索引等
type User struct {
	Id int64 `gorm:"primaryKey,autoIncrement"` // 主键，自动递增
	// 设置邮箱字段为唯一索引
	Email    string `gorm:"unique"`
	Password string // 用户密码

	// 创建时间戳字段
	Ctime int64 // 创建时间
	// 更新时间戳字段
	Utime int64 // 更新时间
}
