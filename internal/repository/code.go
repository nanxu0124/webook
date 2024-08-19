package repository

import (
	"context"
	"webook/internal/repository/cache"
)

var (
	ErrCodeSendTooMany        = cache.ErrCodeSendTooMany
	ErrCodeVerifyTooManyTimes = cache.ErrCodeVerifyTooManyTimes
)

type CodeRepository interface {
	Store(ctx context.Context, biz string, phone string, code string) error
	Verify(ctx context.Context, biz string, phone string, code string) (bool, error)
}

type CacheCodeRepository struct {
	cc cache.CodeCache
}

func NewCodeRepository(cc cache.CodeCache) CodeRepository {
	return &CacheCodeRepository{
		cc: cc,
	}
}

func (repo *CacheCodeRepository) Store(ctx context.Context, biz string, phone string, code string) error {
	return repo.cc.Set(ctx, biz, phone, code)
}

func (repo *CacheCodeRepository) Verify(ctx context.Context, biz string, phone string, code string) (bool, error) {
	return repo.cc.Verify(ctx, biz, phone, code)
}
