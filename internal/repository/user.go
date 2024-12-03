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

	// Create 用于在数据存储中创建一个新的用户
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期，支持超时、取消操作等
	//   - u: 用户对象，包含了要创建的新用户的详细信息（如姓名、手机号、邮箱等）
	// 返回:
	//   - error: 如果创建成功返回 nil；如果创建失败（例如数据库操作失败）则返回错误信息
	Create(ctx context.Context, u domain.User) error

	// FindByPhone 根据用户的手机号码查找用户
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - phone: 用户的手机号，唯一标识用户身份
	// 返回:
	//   - domain.User: 查找到的用户对象。如果未找到，返回空用户对象
	//   - error: 查找过程中的错误。如果没有找到对应的用户，通常会返回 `ErrDataNotFound`
	FindByPhone(ctx context.Context, phone string) (domain.User, error)

	// FindByEmail 根据用户的邮箱地址查找用户
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - email: 用户的邮箱地址，唯一标识用户身份
	// 返回:
	//   - domain.User: 查找到的用户对象。如果未找到，返回空用户对象
	//   - error: 查找过程中的错误。如果没有找到对应的用户，通常会返回 `ErrDataNotFound`
	FindByEmail(ctx context.Context, email string) (domain.User, error)

	// FindById 根据用户的 ID 查找用户
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - id: 用户的唯一标识符（ID）
	// 返回:
	//   - domain.User: 查找到的用户对象。如果未找到，返回空用户对象
	//   - error: 查找过程中的错误。如果没有找到对应的用户，通常会返回 `ErrDataNotFound`
	FindById(ctx context.Context, id int64) (domain.User, error)
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

// entityToDomain 将数据库实体（dao.User）转换为领域模型（domain.User）
// 该方法的目的是将数据访问层（DAO）中的实体对象转换为领域模型对象
// 领域模型对象用于业务逻辑层处理，通常领域模型中包含的字段与数据库实体可能有所不同
func (ur *CachedUserRepository) entityToDomain(ue dao.User) domain.User {
	// 从数据库实体（dao.User）构建领域模型（domain.User）
	return domain.User{
		Id:       ue.Id,           // 用户 ID
		Email:    ue.Email.String, // 用户邮箱（确保处理数据库 NULL 值）
		Password: ue.Password,     // 用户密码
		Phone:    ue.Phone.String, // 用户手机号（确保处理数据库 NULL 值）
		Ctime:    time.UnixMilli(ue.Ctime),
	}
}
