package service

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"webook/internal/domain"
	"webook/internal/repository"
	"webook/pkg/logger"
)

var (
	ErrUserDuplicateEmail    = repository.ErrUserDuplicate
	ErrInvalidUserOrPassword = errors.New("邮箱或者密码不正确")
)

type UserService interface {
	Signup(ctx context.Context, u domain.User) error
	FindOrCreate(ctx context.Context, phone string) (domain.User, error)
	Login(ctx context.Context, email, password string) (domain.User, error)
	Profile(ctx context.Context, id int64) (domain.User, error)
	// UpdateNonSensitiveInfo 更新非敏感数据
	UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error
}

// UserService 结构体，表示用户相关的业务逻辑服务
type userService struct {
	repo   repository.UserRepository // 引用repository层的UserRepository对象，用于数据访问
	logger logger.Logger
}

// NewUserService 实现 UserService 接口
func NewUserService(repo repository.UserRepository, l logger.Logger) UserService {
	return &userService{
		repo:   repo,
		logger: l,
	}
}

// Signup 方法用于用户注册
// 参数ctx为上下文，用于控制操作的生命周期；u为要注册的用户信息，包含Email和Password字段
// 该方法首先对用户密码进行加密，然后将加密后的密码保存到数据库
func (svc *userService) Signup(ctx context.Context, u domain.User) error {
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
func (svc *userService) FindOrCreate(ctx context.Context, phone string) (domain.User, error) {
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

func (svc *userService) Login(ctx context.Context, email, password string) (domain.User, error) {
	// 查找数据库中是否存在该邮箱的用户
	u, err := svc.repo.FindByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		// 如果用户没有找到，返回一个“用户或密码错误”的错误
		return domain.User{}, ErrInvalidUserOrPassword
	}

	// 使用 bcrypt 的 CompareHashAndPassword 方法来验证用户输入的密码
	// 将数据库中的密码哈希和用户输入的密码进行比较
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		// 如果密码不匹配，返回一个“用户或密码错误”的错误
		return domain.User{}, ErrInvalidUserOrPassword
	}

	// 密码验证通过，返回用户信息
	return u, err
}

func (svc *userService) UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error {
	// 依赖于 repository 中更新会忽略 0 值
	// 这个转换的意义在于，在 service 层面上维护住了什么是敏感字段这个语义
	user.Email = ""
	user.Phone = ""
	user.Password = ""
	return svc.repo.Update(ctx, user)
}

func (svc *userService) Profile(ctx context.Context,
	id int64) (domain.User, error) {
	return svc.repo.FindById(ctx, id)
}
