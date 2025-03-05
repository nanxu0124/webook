package cache

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ecodeclub/ekit/syncx/atomicx"
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

func NewRedisRankingCache(client redis.Cmdable) RankingCache {
	return &RedisRankingCache{
		key:        "ranking:article",
		client:     client,
		expiration: time.Minute * 3,
	}
}

// RankingLocalCache 本地缓存
type RankingLocalCache struct {
	topN       *atomicx.Value[[]domain.Article]
	ddl        *atomicx.Value[time.Time]
	expiration time.Duration
}

func NewRankingLocalCache() *RankingLocalCache {
	return &RankingLocalCache{
		topN:       atomicx.NewValue[[]domain.Article](),
		ddl:        atomicx.NewValueOf[time.Time](time.Now()),
		expiration: time.Minute * 3,
	}
}

func (r *RankingLocalCache) Set(_ context.Context, arts []domain.Article) error {
	r.ddl.Store(time.Now().Add(time.Minute * 3))
	r.topN.Store(arts)
	return nil
}

func (r *RankingLocalCache) Get(_ context.Context) ([]domain.Article, error) {
	arts := r.topN.Load()
	if len(arts) == 0 || r.ddl.Load().Before(time.Now()) {
		return nil, errors.New("本地缓存失效了")
	}
	return arts, nil
}

func (r *RankingLocalCache) ForceGet(_ context.Context) ([]domain.Article, error) {
	return r.topN.Load(), nil
}
