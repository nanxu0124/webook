package repository

import (
	"context"
	"webook/internal/repository/cache"
)

var (
	ErrCodeVerifyTooManyTimes = cache.ErrCodeVerifyTooManyTimes
	ErrCodeSendTooMany        = cache.ErrCodeSendTooMany
)

// CodeRepository 是用于操作验证码数据的接口，主要包括验证码的存储和验证
// 该接口用于与验证码缓存系统进行交互，通常与 Redis 等缓存系统集成
// 主要有两个功能：
//   - 存储验证码：将验证码存储在缓存中，并设置过期时间
//   - 验证验证码：检查用户输入的验证码是否正确，并在验证成功后将验证码删除
type CodeRepository interface {

	// Store 存储验证码到缓存中
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期，便于实现超时或取消操作
	//   - biz: 业务场景标识，用于区分不同的验证码场景（例如注册、登录等）
	//   - phone: 用户的手机号码，验证码是与手机号码绑定的
	//   - code: 要存储的验证码
	// 返回:
	//   - error: 如果存储成功，返回 nil；如果存储失败，返回相应的错误信息
	Store(ctx context.Context, biz string, phone string, code string) error

	// Verify 验证用户输入的验证码
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - biz: 业务场景标识，用于区分不同的验证码场景
	//   - phone: 用户的手机号码
	//   - inputCode: 用户输入的验证码
	// 返回:
	//   - bool: 如果验证码验证成功，返回 true；否则返回 false
	//   - error: 验证过程中发生的错误。如果发生错误，返回相应的错误信息
	Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error)
}

// CachedCodeRepository 实现 CodeRepository 接口
type CachedCodeRepository struct {
	cache cache.CodeCache
}

func NewCachedCodeRepository(c cache.CodeCache) CodeRepository {
	return &CachedCodeRepository{
		cache: c,
	}
}

func (repo *CachedCodeRepository) Store(ctx context.Context, biz string, phone string, code string) error {
	// 将验证码存储到缓存中
	err := repo.cache.Set(ctx, biz, phone, code)
	return err // 返回存储过程中可能发生的错误
}

func (repo *CachedCodeRepository) Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error) {
	// 调用缓存中的 Verify 方法来验证验证码
	return repo.cache.Verify(ctx, biz, phone, inputCode)
}
