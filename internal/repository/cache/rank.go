package cache

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"time"
	"webook/internal/domain"
)

type RankingCache interface {
	Set(ctx context.Context, art []domain.Article) error
	Get(ctx context.Context) ([]domain.Article, error)
}

type RedisRankingCache struct {
	client     redis.Cmdable
	key        string
	expiration time.Duration
}

func (r *RedisRankingCache) Set(ctx context.Context, art []domain.Article) error {
	for _, v := range art {
		v.Content = ""
	}

	val, err := json.Marshal(art)
	if err != nil {
		return err
	}
	// 过期时间要设置得比定时计算的间隔长
	return r.client.Set(ctx, r.key, val, r.expiration).Err()
}

func (r *RedisRankingCache) Get(ctx context.Context) ([]domain.Article, error) {
	val, err := r.client.Get(ctx, r.key).Bytes()
	if err != nil {
		return nil, err
	}
	var res []domain.Article
	err = json.Unmarshal(val, &res)
	return nil, err
}

func NewRedisRankingCache(client redis.Cmdable, expiration time.Duration) RankingCache {
	return &RedisRankingCache{
		key:        "ranking:article",
		client:     client,
		expiration: expiration,
	}
}
