package service

import (
	"context"
	"github.com/ecodeclub/ekit/queue"
	"github.com/ecodeclub/ekit/slice"
	"math"
	"time"
	service2 "webook/interactive/service"
	"webook/internal/domain"
	"webook/internal/repository"
)

type RankingService interface {
	// RankTopN 计算 TopN
	RankTopN(ctx context.Context) error
	GetTopN(ctx context.Context) ([]domain.Article, error)
}

type BatchRankingService struct {
	intrSvc   service2.InteractiveService
	artSvc    ArticleService
	repo      repository.RankingRepository
	BatchSize int
	N         int
	scoreFunc func(likeCnt int64, utime time.Time) float64
}

func NewBatchRankingService(intrSvc service2.InteractiveService, artSvc ArticleService, repo repository.RankingRepository) RankingService {
	res := &BatchRankingService{
		intrSvc:   intrSvc,
		artSvc:    artSvc,
		repo:      repo,
		BatchSize: 100,
		N:         100,
	}
	res.scoreFunc = res.score

	return res
}

func (b *BatchRankingService) GetTopN(ctx context.Context) ([]domain.Article, error) {
	return b.repo.GetTopN(ctx)
}

func (b *BatchRankingService) RankTopN(ctx context.Context) error {
	arts, err := b.rankTopN(ctx)
	if err != nil {
		return err
	}
	// 准备放到缓存里面
	return b.repo.ReplaceTopN(ctx, arts)
}

func (b *BatchRankingService) rankTopN(ctx context.Context) ([]domain.Article, error) {
	now := time.Now()
	// 只计算七天内的，因为超过七天的可以认为绝对不可能成为热榜了
	// 如果一个批次里面 utime 最小已经是七天之前的，就中断当前计算
	ddl := now.Add(-time.Hour * 24 * 7)
	offset := 0

	type Score struct {
		art   domain.Article
		score float64
	}
	// 这是一个优先级队列，维持住了 topN 的 id。
	topN := queue.NewPriorityQueue[Score](b.N,
		func(src Score, dst Score) int {
			if src.score > dst.score {
				return 1
			} else if src.score == dst.score {
				return 0
			} else {
				return -1
			}
		})

	for {
		arts, err := b.artSvc.ListPub(ctx, now, offset, b.BatchSize)
		if err != nil {
			return nil, err
		}
		artIds := slice.Map[domain.Article, int64](arts, func(idx int, src domain.Article) int64 {
			return src.Id
		})
		intrMap, err := b.intrSvc.GetByIds(ctx, "article", artIds)
		if err != nil {
			return nil, err
		}

		for _, art := range arts {
			intr, ok := intrMap[art.Id]
			if !ok {
				continue
			}
			score := b.scoreFunc(intr.LikeCnt, art.Utime)

			err = topN.Enqueue(Score{
				art:   art,
				score: score,
			})
			if err == queue.ErrOutOfCapacity {
				val, _ := topN.Dequeue()
				if val.score < score {
					_ = topN.Enqueue(Score{
						art:   art,
						score: score,
					})
				} else {
					_ = topN.Enqueue(val)
				}
			}
		}
		if len(arts) < b.BatchSize ||
			arts[len(arts)-1].Utime.Before(ddl) {
			break
		}
		offset = offset + len(arts)
	}

	res := make([]domain.Article, b.N)
	for i := b.N - 1; i >= 0; i-- {
		val, err := topN.Dequeue()
		if err != nil {
			// 说明取完了，不够 n
			break
		}
		res[i] = val.art
	}
	return res, nil
}

func (b *BatchRankingService) score(likeCnt int64, utime time.Time) float64 {
	// 这个 factor 也可以做成一个参数
	const factor = 1.5
	return float64(likeCnt-1) /
		math.Pow(time.Since(utime).Hours()+2, factor)
}
