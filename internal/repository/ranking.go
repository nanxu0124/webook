package repository

import (
	"context"
	"webook/internal/domain"
	"webook/internal/repository/cache"
)

type RankingRepository interface {
	ReplaceTopN(ctx context.Context, arts []domain.Article) error
	GetTopN(ctx context.Context) ([]domain.Article, error)
}

type CachedRankingRepository struct {
	redisCache *cache.RedisRankingCache
	localCache *cache.RankingLocalCache
}

func NewCachedRankingRepository(cache *cache.RedisRankingCache, localCache *cache.RankingLocalCache) RankingRepository {
	return &CachedRankingRepository{
		redisCache: cache,
		localCache: localCache,
	}
}

func (c *CachedRankingRepository) ReplaceTopN(ctx context.Context, arts []domain.Article) error {
	_ = c.localCache.Set(ctx, arts)
	return c.redisCache.Set(ctx, arts)
}

func (c *CachedRankingRepository) GetTopN(ctx context.Context) ([]domain.Article, error) {
	arts, err := c.localCache.Get(ctx)
	if err == nil {
		return arts, nil
	}

	arts, err = c.redisCache.Get(ctx)
	if err == nil {
		// 回写本地缓存
		_ = c.localCache.Set(ctx, arts)
	} else {
		return c.localCache.ForceGet(ctx)
	}
	return arts, err
}
