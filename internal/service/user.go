package service

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"webook/internal/domain"
	"webook/internal/repository"
)

// ErrUserDuplicateEmail 定义一个常量ErrUserDuplicateEmail，指代用户重复邮件错误，来自于repository层
var ErrUserDuplicateEmail = repository.ErrUserDuplicateEmail

// UserService 结构体，表示用户相关的业务逻辑服务
type UserService struct {
	repo *repository.UserRepository // 引用repository层的UserRepository对象，用于数据访问
}

// NewUserService 函数，创建并返回一个新的UserService实例
// 该函数接收一个*repository.UserRepository类型的参数repo，用于初始化UserService的repo字段
func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo, // 将传入的repository.UserRepository对象赋值给UserService的repo字段
	}
}

// Signup 方法用于用户注册
// 参数ctx为上下文，用于控制操作的生命周期；u为要注册的用户信息，包含Email和Password字段
// 该方法首先对用户密码进行加密，然后将加密后的密码保存到数据库
func (svc *UserService) Signup(ctx context.Context, u domain.User) error {
	// 使用bcrypt生成加密后的密码
	// bcrypt 会自动为每个密码生成一个随机的盐值，并将其与密码一起存储在最终的哈希值中，不需要手动生成或存储盐值
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		// 如果密码加密失败，返回错误
		return err
	}
	// 将加密后的密码转为字符串并赋值给u.Password
	u.Password = string(hash)
	// 调用repository层的Create方法将加密后的用户信息保存到数据库
	return svc.repo.Create(ctx, u)
}
