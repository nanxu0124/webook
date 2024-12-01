package repository

import (
	"context"
	"webook/internal/domain" // 引入domain包，定义了User等业务模型
	"webook/internal/repository/cache"
	"webook/internal/repository/dao" // 引入dao包，进行与数据库交互的操作
)

// ErrUserDuplicateEmail 定义一个常量ErrUserDuplicateEmail，指代数据库层返回的重复邮件错误
var ErrUserDuplicateEmail = dao.ErrUserDuplicateEmail

var ErrUserNotFound = dao.ErrDataNotFound

// UserRepository 定义UserRepository结构体，表示用户数据访问对象
type UserRepository struct {
	dao   *dao.UserDAO     // 引用dao层的UserDAO对象，UserDAO负责与数据库进行操作
	cache *cache.UserCache // 引用dao层的UserCache对象，UserCache负责与缓存进行操作
}

// NewUserRepository 函数，创建并返回一个新的UserRepository实例
// 该函数接收一个*dao.UserDAO类型的参数d，用于初始化UserRepository的dao字段
func NewUserRepository(d *dao.UserDAO, c *cache.UserCache) *UserRepository {
	return &UserRepository{
		dao:   d,
		cache: c,
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

func (ur *UserRepository) FindByEmail(ctx context.Context,
	email string) (domain.User, error) {
	u, err := ur.dao.FindByEmail(ctx, email)

	return domain.User{
		Id:       u.Id,
		Email:    u.Email,
		Password: u.Password,
	}, err
}

// FindById 根据用户 ID 从缓存或数据库中查找用户信息
// 如果缓存中有用户数据，直接返回缓存数据；
// 如果缓存中没有数据，则从数据库中查找并将数据缓存起来。
func (ur *UserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
	// 首先尝试从缓存中获取用户数据
	u, err := ur.cache.Get(ctx, id)
	switch err {
	// 如果缓存中有数据，直接返回该数据
	case nil:
		return u, err

	// 如果缓存中没有数据，则从数据库中查找
	case cache.ErrKeyNotExist:
		// 从数据库中查找用户
		ue, err := ur.dao.FindById(ctx, id)
		if err != nil {
			// 如果数据库中没有该用户，返回错误
			return domain.User{}, err
		}
		// 构造用户数据
		u = domain.User{
			Id:       ue.Id,
			Email:    ue.Email,
			Password: ue.Password,
		}
		// 将用户数据存入缓存
		_ = ur.cache.Set(ctx, u)

		// 返回从数据库中查找到的用户数据
		return u, nil

	// 如果其他错误发生，返回该错误
	default:
		return domain.User{}, err
	}
}
