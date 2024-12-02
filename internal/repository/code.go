package repository

import (
	"context"
	"webook/internal/repository/cache"
)

var (
	ErrCodeVerifyTooManyTimes = cache.ErrCodeVerifyTooManyTimes
	ErrCodeSendTooMany        = cache.ErrCodeSendTooMany
)

type CodeRepository struct {
	cache *cache.CodeCache
}

func NewCodeRepository(c *cache.CodeCache) *CodeRepository {
	return &CodeRepository{
		cache: c,
	}
}

// Store 存储验证码
// 该方法通过调用缓存层的 Set 方法，将生成的验证码存储到缓存中。
// - biz: 业务标识符，例如注册、找回密码等场景。
// - phone: 用户的手机号码。
// - code: 要存储的验证码。
func (repo *CodeRepository) Store(ctx context.Context, biz string, phone string, code string) error {
	// 将验证码存储到缓存中
	err := repo.cache.Set(ctx, biz, phone, code)
	return err // 返回存储过程中可能发生的错误
}

// Verify 验证验证码
// 该方法使用缓存层的 Verify 方法比较用户输入的验证码与缓存中存储的验证码。
// 如果验证码正确，则返回 true，并删除缓存中的验证码。
// - biz: 业务标识符，确保验证码是针对特定业务场景。
// - phone: 用户的手机号码。
// - inputCode: 用户输入的验证码。
// 返回值：
// - bool: 是否验证码验证成功（验证码正确）。
// - error: 如果验证过程中发生错误，返回错误信息。
func (repo *CodeRepository) Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error) {
	// 调用缓存中的 Verify 方法来验证验证码
	return repo.cache.Verify(ctx, biz, phone, inputCode)
}
