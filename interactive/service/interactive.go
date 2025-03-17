package service

import (
	"context"
	"golang.org/x/sync/errgroup"
	"webook/interactive/domain"
	"webook/interactive/repository"
	"webook/pkg/logger"
)

//go:generate mockgen -source=interactive.go -package=svcmocks -destination=mocks/interactive.mock.go InteractiveService
type InteractiveService interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	// Like 点赞
	Like(ctx context.Context, biz string, bizId int64, uid int64) error
	// CancelLike 取消点赞
	CancelLike(ctx context.Context, biz string, bizId int64, uid int64) error
	// Collect 收藏
	Collect(ctx context.Context, biz string, bizId, cid, uid int64) error
	Get(ctx context.Context, biz string, bizId, uid int64) (domain.Interactive, error)
	GetByIds(ctx context.Context, biz string, bizIds []int64) (map[int64]domain.Interactive, error)
}

type interactiveService struct {
	repo repository.InteractiveRepository
	l    logger.Logger
}

func NewInteractiveService(repo repository.InteractiveRepository, l logger.Logger) InteractiveService {
	return &interactiveService{
		repo: repo,
		l:    l,
	}
}

func (i *interactiveService) GetByIds(ctx context.Context, biz string, bizIds []int64) (map[int64]domain.Interactive, error) {
	intrs, err := i.repo.GetByIds(ctx, biz, bizIds)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]domain.Interactive, len(intrs))
	for _, intr := range intrs {
		res[intr.BizId] = intr
	}
	return res, nil
}

func (i *interactiveService) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	return i.repo.IncrReadCnt(ctx, biz, bizId)
}

func (i *interactiveService) Like(ctx context.Context, biz string, bizId int64, uid int64) error {
	return i.repo.IncrLike(ctx, biz, bizId, uid)
}

func (i *interactiveService) CancelLike(ctx context.Context, biz string, bizId int64, uid int64) error {
	return i.repo.DecrLike(ctx, biz, bizId, uid)
}

func (i *interactiveService) Collect(ctx context.Context, biz string, bizId, cid, uid int64) error {
	return i.repo.AddCollectionItem(ctx, biz, bizId, cid, uid)
}

// Get 获取当前业务对象（例如文章、视频等）的互动信息，包括阅读数、点赞数、收藏数。
// 如果用户已登录，还会返回用户是否点赞和收藏该对象的信息。
// 参数：
//
//	ctx: 上下文对象，用于控制请求的生命周期（如超时、取消等）
//	biz: 业务类型（例如：文章、视频等）
//	bizId: 业务对象 ID（例如：文章 ID 或视频 ID）
//	uid: 用户 ID（如果用户未登录，uid 为 0）
//
// 返回：
//
//	domain.Interactive: 该业务对象的互动信息，包含阅读数、点赞数、收藏数等
//	error: 错误信息，如果发生错误，返回相应的错误
func (i *interactiveService) Get(ctx context.Context, biz string, bizId, uid int64) (domain.Interactive, error) {
	// 也可以考虑将分发的逻辑也下沉到 repository 里面
	// 这里从 repository 获取当前文章（或其他业务对象）的互动数据（如阅读数、点赞数、收藏数等）
	intr, err := i.repo.Get(ctx, biz, bizId)
	if err != nil {
		return domain.Interactive{}, err // 如果获取互动信息失败，返回空结构体和错误
	}

	// 如果 uid > 0，说明用户已登录，进一步查询该用户是否已点赞或收藏该业务对象
	if uid > 0 {
		var eg errgroup.Group // 使用 errgroup.Group 来并发执行多个任务（查询用户点赞和收藏信息）

		// 启动并发查询：是否点赞当前文章
		eg.Go(func() error {
			intr.Liked, err = i.repo.Liked(ctx, biz, bizId, uid)
			return err
		})

		// 启动并发查询：是否收藏当前文章
		eg.Go(func() error {
			intr.Collected, err = i.repo.Collected(ctx, biz, bizId, uid)
			return err
		})

		// 等待并发查询结果
		err = eg.Wait()
		if err != nil {
			// 如果查询失败，只记录日志，不需要中断执行
			i.l.Error("查询用户是否点赞的信息失败",
				logger.String("biz", biz),    // 业务类型
				logger.Int64("bizId", bizId), // 业务对象 ID
				logger.Int64("uid", uid),     // 用户 ID
				logger.Error(err))            // 错误信息
		}
	}

	return intr, err // 返回互动信息（包含阅读数、点赞数、收藏数）和可能的错误
}
