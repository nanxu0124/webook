package service

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"webook/internal/domain"
	"webook/internal/repository"
)

var (
	ErrUserDuplicateEmail    = repository.ErrUserDuplicate
	ErrInvalidUserOrPassword = errors.New("账号或密码不正确")
)

type UserService interface {
	SignUp(ctx context.Context, u domain.User) error
	Login(ctx context.Context, email string, password string) (domain.User, error)
	Edit(ctx context.Context, u domain.User) error
	GetProfile(ctx context.Context, userId int64) (domain.User, error)
	FindOrCreate(ctx context.Context, phone string) (domain.User, error)
	FindOrCreateByWechat(ctx context.Context, info domain.WechatInfo) (domain.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{
		repo: repo,
	}
}

func (svc *userService) SignUp(ctx context.Context, u domain.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return svc.repo.Create(ctx, u)
}

func (svc *userService) Login(ctx context.Context, email string, password string) (domain.User, error) {
	u, err := svc.repo.FindByEmail(ctx, email)
	if err == repository.ErrUserNotFound {
		return domain.User{}, ErrInvalidUserOrPassword
	}
	if err != nil {
		return domain.User{}, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return domain.User{}, ErrInvalidUserOrPassword
	}

	return u, nil
}

func (svc *userService) Edit(ctx context.Context, u domain.User) error {
	return svc.repo.EditById(ctx, u)
}

func (svc *userService) GetProfile(ctx context.Context, userId int64) (domain.User, error) {
	return svc.repo.FindById(ctx, userId)
}

func (svc *userService) FindOrCreate(ctx context.Context, phone string) (domain.User, error) {
	// 这里需要一个user的id
	// 先查找用户是否存在，存在就返回id
	// 不存在就先创建用户，在返回id
	u, err := svc.repo.FindByPhone(ctx, phone)
	if err != repository.ErrUserNotFound {
		// 这是快路径
		// 绝大多数请求会进来这里，因为绝大多数人已经注册过了
		return u, err
	}

	// 下边是慢路径
	// 当找不到用户的时候就需要创建
	u = domain.User{
		Phone: phone,
	}
	err = svc.repo.Create(ctx, u)
	if err != nil && err != repository.ErrUserDuplicate {
		return u, err
	}

	// 这里会遇到主从不一致的问题
	return svc.repo.FindByPhone(ctx, phone)
}

func (svc *userService) FindOrCreateByWechat(ctx context.Context, info domain.WechatInfo) (domain.User, error) {
	// 这里需要一个user的id
	// 先查找用户是否存在，存在就返回id
	// 不存在就先创建用户，在返回id
	u, err := svc.repo.FindByWechat(ctx, info.OpenID)
	if err != repository.ErrUserNotFound {
		// 这是快路径
		// 绝大多数请求会进来这里，因为绝大多数人已经注册过了
		return u, err
	}

	// 下边是慢路径
	// 当找不到用户的时候就需要创建
	u = domain.User{
		WechatInfo: info,
	}
	err = svc.repo.Create(ctx, u)
	if err != nil && err != repository.ErrUserDuplicate {
		return u, err
	}

	// 这里会遇到主从不一致的问题
	return svc.repo.FindByWechat(ctx, info.OpenID)
}
