package service

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"webook/internal/domain"
	"webook/internal/repository"
)

// ErrUserDuplicateEmail 定义一个常量ErrUserDuplicateEmail，指代用户重复邮件错误，来自于repository层
var ErrUserDuplicateEmail = repository.ErrUserDuplicate
var ErrInvalidUserOrPassword = errors.New("邮箱或者密码不正确")

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

// FindOrCreate 如果手机号不存在，那么会初始化一个用户
func (svc *UserService) FindOrCreate(ctx context.Context, phone string) (domain.User, error) {
	// 这是一种优化写法
	// 大部分人会命中这个分支
	u, err := svc.repo.FindByPhone(ctx, phone)       // 从数据库中查找用户
	if !errors.Is(err, repository.ErrUserNotFound) { // 如果用户已经存在，则直接返回
		return u, err
	}
	// 如果找不到用户，则执行用户注册操作
	err = svc.repo.Create(ctx, domain.User{
		Phone: phone, // 创建新用户时只需要手机号
	})
	// 注册过程中，如果发生了非手机号码冲突的错误，说明是系统错误
	if err != nil && !errors.Is(err, repository.ErrUserDuplicate) {
		return domain.User{}, err // 返回错误，表示用户创建失败
	}
	// 如果注册成功或者是重复注册（用户已经存在），从数据库重新查询该手机号的用户
	return svc.repo.FindByPhone(ctx, phone) // 返回用户
}

func (svc *UserService) Login(ctx context.Context,
	email, password string) (domain.User, error) {
	u, err := svc.repo.FindByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return domain.User{}, ErrInvalidUserOrPassword
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return domain.User{}, ErrInvalidUserOrPassword
	}
	return u, err
}

func (svc *UserService) Profile(ctx context.Context,
	id int64) (domain.User, error) {
	return svc.repo.FindById(ctx, id)
}
