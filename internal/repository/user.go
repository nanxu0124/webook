package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"webook/internal/domain" // 引入domain包，定义了User等业务模型
	"webook/internal/repository/cache"
	"webook/internal/repository/dao" // 引入dao包，进行与数据库交互的操作
)

var (
	ErrUserDuplicate = dao.ErrUserDuplicate
	ErrUserNotFound  = dao.ErrDataNotFound
)

// UserRepository 是一个用于操作用户数据的接口，主要包括用户的创建、查找等操作
// 该接口提供了一些常见的用户管理功能，与数据库交互
type UserRepository interface {
	Create(ctx context.Context, u domain.User) error
	FindByPhone(ctx context.Context, phone string) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindById(ctx context.Context, id int64) (domain.User, error)
	// Update 更新数据，只有非 0 值才会更新
	Update(ctx context.Context, u domain.User) error
}

// CachedUserRepository 实现 UserRepository 接口
type CachedUserRepository struct {
	dao   dao.UserDAO     // 引用dao层的UserDAO对象，UserDAO负责与数据库进行操作
	cache cache.UserCache // 引用dao层的UserCache对象，UserCache负责与缓存进行操作
}

func NewCachedUserRepository(d dao.UserDAO, c cache.UserCache) UserRepository {
	return &CachedUserRepository{
		dao:   d,
		cache: c,
	}
}

func (ur *CachedUserRepository) Create(ctx context.Context, u domain.User) error {
	return ur.dao.Insert(ctx, dao.User{
		Email: sql.NullString{
			String: u.Email,
			Valid:  u.Email != "",
		},
		Phone: sql.NullString{
			String: u.Phone,
			Valid:  u.Phone != "",
		},
		Password: u.Password,
	})
}

func (ur *CachedUserRepository) FindByPhone(ctx context.Context,
	phone string) (domain.User, error) {
	u, err := ur.dao.FindByPhone(ctx, phone)
	return ur.entityToDomain(u), err
}

func (ur *CachedUserRepository) FindByEmail(ctx context.Context,
	email string) (domain.User, error) {
	u, err := ur.dao.FindByEmail(ctx, email)
	return ur.entityToDomain(u), err
}

func (ur *CachedUserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
	// 首先尝试从缓存中获取用户数据
	u, err := ur.cache.Get(ctx, id)
	switch {
	// 如果缓存中有数据，直接返回该数据
	case err == nil:
		return u, err
		// 如果缓存中没有数据，则从数据库中查找
	case errors.Is(err, cache.ErrKeyNotExist):
		// 从数据库中查找用户
		ue, err := ur.dao.FindById(ctx, id)
		if err != nil {
			// 如果数据库中没有该用户，返回错误
			return domain.User{}, err
		}
		// 构造用户数据
		u = ur.entityToDomain(ue)
		// 将用户数据存入缓存
		_ = ur.cache.Set(ctx, u)

		// 返回从数据库中查找到的用户数据
		return u, nil
		// 如果其他错误发生，返回该错误
	default:
		return domain.User{}, err
	}
}

func (ur *CachedUserRepository) Update(ctx context.Context, u domain.User) error {
	err := ur.dao.UpdateNonZeroFields(ctx, ur.domainToEntity(u))
	if err != nil {
		return err
	}
	return ur.cache.Delete(ctx, u.Id)
}

// domainToEntity 将领域模型（domain.User）转换为数据库实体（dao.User）
func (ur *CachedUserRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{
		Id: u.Id,
		Email: sql.NullString{
			String: u.Email,
			Valid:  u.Email != "",
		},
		Phone: sql.NullString{
			String: u.Phone,
			Valid:  u.Phone != "",
		},
		Birthday: sql.NullInt64{
			Int64: u.Birthday.UnixMilli(),
			Valid: !u.Birthday.IsZero(),
		},
		Nickname: sql.NullString{
			String: u.Nickname,
			Valid:  u.Nickname != "",
		},
		AboutMe: sql.NullString{
			String: u.AboutMe,
			Valid:  u.AboutMe != "",
		},
		Password: u.Password,
	}
}

// entityToDomain 将数据库实体（dao.User）转换为领域模型（domain.User）
// 该方法的目的是将数据访问层（DAO）中的实体对象转换为领域模型对象
// 领域模型对象用于业务逻辑层处理，通常领域模型中包含的字段与数据库实体可能有所不同
func (ur *CachedUserRepository) entityToDomain(ue dao.User) domain.User {
	// 从数据库实体（dao.User）构建领域模型（domain.User）
	var birthday time.Time
	if ue.Birthday.Valid {
		birthday = time.UnixMilli(ue.Birthday.Int64)
	}
	return domain.User{
		Id:       ue.Id,           // 用户 ID
		Email:    ue.Email.String, // 用户邮箱（确保处理数据库 NULL 值）
		Password: ue.Password,     // 用户密码
		Phone:    ue.Phone.String, // 用户手机号（确保处理数据库 NULL 值）
		Nickname: ue.Nickname.String,
		AboutMe:  ue.AboutMe.String,
		Birthday: birthday,
		Ctime:    time.UnixMilli(ue.Ctime),
	}
}
