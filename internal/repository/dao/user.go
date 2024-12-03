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
	// ErrUserDuplicate 表示用户邮箱或者手机号冲突错误
	ErrUserDuplicate = errors.New("用户邮箱或者手机号冲突")

	// ErrDataNotFound 通用的数据没找到错误（即Gorm的记录未找到）
	ErrDataNotFound = gorm.ErrRecordNotFound
)

// UserDAO 定义了操作用户数据的接口，通常用于与数据库进行交互
// DAO（Data Access Object）层的主要职责是与数据源（如数据库、文件系统等）进行交互
// 提供持久化相关的操作。这个接口包含了用户数据的常见操作，如插入、查找等
type UserDAO interface {

	// Insert 插入新的用户记录
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期，便于实现超时或取消操作
	//   - u: 要插入的用户数据，包含用户的基本信息（如 id、email、password 等）
	// 返回:
	//   - error: 如果插入成功，返回 nil；如果插入失败，返回相应的错误信息
	Insert(ctx context.Context, u User) error

	// FindByPhone 根据手机号码查找用户
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - phone: 用户的手机号码
	// 返回:
	//   - User: 匹配的用户数据
	//   - error: 如果查询成功，返回用户信息；如果查询失败（如没有该用户），返回错误信息
	FindByPhone(ctx context.Context, phone string) (User, error)

	// FindByEmail 根据邮箱查找用户
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - email: 用户的电子邮件地址
	// 返回:
	//   - User: 匹配的用户数据
	//   - error: 如果查询成功，返回用户信息；如果查询失败（如没有该用户），返回错误信息
	FindByEmail(ctx context.Context, email string) (User, error)

	// FindById 根据用户的 ID 查找用户
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - id: 用户的唯一 ID
	// 返回:
	//   - User: 匹配的用户数据
	//   - error: 如果查询成功，返回用户信息；如果查询失败（如没有该用户），返回错误信息
	FindById(ctx context.Context, id int64) (User, error)
}

// GormUserDAO 是与用户相关的数据访问对象，它封装了与用户数据表交互的所有操作
type GormUserDAO struct {
	db *gorm.DB // Gorm DB 实例，用于与数据库交互
}

// NewGormUserDAO 创建并返回一个新的 UserDAO 实例
// 参数 db 是已经初始化好的 Gorm DB 实例
func NewGormUserDAO(db *gorm.DB) UserDAO {
	return &GormUserDAO{
		db: db,
	}
}

func (ud *GormUserDAO) Insert(ctx context.Context, u User) error {
	// 获取当前时间戳，用于设置用户的创建时间和更新时间
	now := time.Now().UnixMilli()
	u.Ctime = now
	u.Utime = now

	// 使用Gorm的Create方法将用户数据插入到数据库
	// 如果插入时出现错误，检查是否是由于邮箱唯一索引冲突导致的
	err := ud.db.WithContext(ctx).Create(&u).Error
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		const uniqueIndexErrNo uint16 = 1062 // 唯一索引冲突错误码
		if me.Number == uniqueIndexErrNo {
			// 如果是唯一索引冲突，返回自定义的 ErrUserDuplicate 错误
			return ErrUserDuplicate
		}
	}
	return err // 如果是其他错误，直接返回
}

func (ud *GormUserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).First(&u, "email = ?", email).Error
	return u, err
}

func (ud *GormUserDAO) FindByPhone(ctx context.Context, phone string) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).First(&u, "phone = ?", phone).Error
	return u, err
}

func (ud *GormUserDAO) FindById(ctx context.Context, id int64) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).First(&u, "id = ?", id).Error
	return u, err
}

// User 表示用户的数据模型，映射到数据库中的用户表
// 通过Gorm的标签来定义字段属性，比如主键、唯一索引等
type User struct {
	Id int64 `gorm:"primaryKey,autoIncrement"` // 主键，自动递增
	// 设置邮箱字段为唯一索引
	Email    sql.NullString `gorm:"unique"`
	Password string         // 用户密码

	//Phone *string
	Phone sql.NullString `gorm:"unique"`

	// 创建时间戳字段
	Ctime int64 // 创建时间
	// 更新时间戳字段
	Utime int64 // 更新时间
}
