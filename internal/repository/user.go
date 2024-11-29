package repository

import (
	"context"
	"webook/internal/domain"         // 引入domain包，定义了User等业务模型
	"webook/internal/repository/dao" // 引入dao包，进行与数据库交互的操作
)

// ErrUserDuplicateEmail 定义一个常量ErrUserDuplicateEmail，指代数据库层返回的重复邮件错误
var ErrUserDuplicateEmail = dao.ErrUserDuplicateEmail

// UserRepository 定义UserRepository结构体，表示用户数据访问对象
type UserRepository struct {
	dao *dao.UserDAO // 引用dao层的UserDAO对象，UserDAO负责与数据库进行操作
}

// NewUserRepository 函数，创建并返回一个新的UserRepository实例
// 该函数接收一个*dao.UserDAO类型的参数d，用于初始化UserRepository的dao字段
func NewUserRepository(d *dao.UserDAO) *UserRepository {
	return &UserRepository{
		dao: d, // 将传入的dao.UserDAO对象赋值给UserRepository的dao字段
	}
}

// Create 方法用于创建一个新的用户
// 参数ctx为上下文，用于控制操作的生命周期；u为要创建的用户信息，包含Email和Password
// 该方法将用户数据传递给dao层的Insert方法，完成用户的创建操作
func (ur *UserRepository) Create(ctx context.Context, u domain.User) error {
	// 调用dao层的Insert方法，将User对象插入数据库
	err := ur.dao.Insert(ctx, dao.User{
		Email:    u.Email,    // 从传入的domain.User中获取Email字段
		Password: u.Password, // 从传入的domain.User中获取Password字段
	})
	// 返回dao层Insert方法的错误，若插入成功，err为nil
	return err
}
