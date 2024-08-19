package repository

import (
	"context"
	"database/sql"
	"webook/internal/domain"
	"webook/internal/repository/cache"
	"webook/internal/repository/dao"
)

var (
	ErrUserDuplicate = dao.ErrUserDuplicate
	ErrUserNotFound  = dao.ErrUserNotFound
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User) error
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindByPhone(ctx context.Context, phone string) (domain.User, error)
	EditById(ctx context.Context, u domain.User) error
	FindById(ctx context.Context, userId int64) (domain.User, error)
	FindByWechat(ctx context.Context, openId string) (domain.User, error)
}

type MySQLUserRepository struct {
	dao   dao.UserDAO
	cache cache.UserCache
}

func NewUserRepository(dao dao.UserDAO, c cache.UserCache) UserRepository {
	return &MySQLUserRepository{
		dao:   dao,
		cache: c,
	}
}

func (r *MySQLUserRepository) Create(ctx context.Context, u domain.User) error {
	return r.dao.Insert(ctx, r.domainToEntity(u))
}

func (r *MySQLUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := r.dao.FindByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	return r.EntityToDomain(u), err
}

func (r *MySQLUserRepository) FindByPhone(ctx context.Context, phone string) (domain.User, error) {
	u, err := r.dao.FindByPhone(ctx, phone)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{
		Id: u.Id,
	}, err
}

func (r *MySQLUserRepository) EditById(ctx context.Context, u domain.User) error {
	return r.dao.UpdateById(ctx, r.domainToEntity(u))
}

func (r *MySQLUserRepository) FindById(ctx context.Context, userId int64) (domain.User, error) {

	// 先从redis找
	u, err := r.cache.Get(ctx, userId)
	if err == nil {
		// err为nil redis里边找到了数据
		return u, nil
	}
	//if err == cache.ErrKeyNotExist {
	//	// redis里边没有这个数据
	//
	//}
	// 如果是其他错误，可能是缓存崩溃了
	// 这种情况如果加载数据，需要对数据库做限流
	// 如果不加载数据，用户体验不好
	// 正常应该加载数据，做好数据库限流

	ue, err := r.dao.FindById(ctx, userId)
	if err != nil {
		return domain.User{}, err
	}
	u = r.EntityToDomain(ue)

	go func() {
		err = r.cache.Set(ctx, u)
		if err != nil {
			// 打日志，做监控
		}
	}()

	return u, nil
}

func (r *MySQLUserRepository) FindByWechat(ctx context.Context, openId string) (domain.User, error) {
	u, err := r.dao.FindByWechat(ctx, openId)
	if err != nil {
		return domain.User{}, err
	}
	return r.EntityToDomain(u), err
}

func (r *MySQLUserRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{
		Id: u.Id,
		Email: sql.NullString{
			String: u.Email,
			// 我确实有手机号
			Valid: u.Email != "",
		},
		Phone: sql.NullString{
			String: u.Phone,
			Valid:  u.Phone != "",
		},
		Password: u.Password,

		WechatOpenId: sql.NullString{
			String: u.WechatInfo.OpenID,
			Valid:  u.WechatInfo.OpenID != "",
		},
		WechatUnionId: sql.NullString{
			String: u.WechatInfo.UnionID,
			Valid:  u.WechatInfo.UnionID != "",
		},

		Nickname:        u.Nickname,
		BirthDay:        u.BirthDay,
		PersonalProfile: u.PersonalProfile,
	}
}

func (r *MySQLUserRepository) EntityToDomain(u dao.User) domain.User {
	return domain.User{
		Id:       u.Id,
		Email:    u.Email.String,
		Password: u.Password,
		Phone:    u.Phone.String,

		WechatInfo: domain.WechatInfo{
			OpenID:  u.WechatOpenId.String,
			UnionID: u.WechatUnionId.String,
		},

		Nickname:        u.Nickname,
		BirthDay:        u.BirthDay,
		PersonalProfile: u.PersonalProfile,
	}
}
