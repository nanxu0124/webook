package repository

import (
	"context"
	"errors"
	"github.com/ecodeclub/ekit/slice"
	"gorm.io/gorm"
	"webook/interactive/domain"
	"webook/interactive/repository/cache"
	dao2 "webook/interactive/repository/dao"
	"webook/pkg/logger"
)

var ErrDataNotFound = gorm.ErrRecordNotFound

type InteractiveRepository interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	BatchIncrReadCnt(ctx context.Context, bizs []string, bizIds []int64) error
	IncrLike(ctx context.Context, biz string, bizId, uid int64) error
	DecrLike(ctx context.Context, biz string, bizId, uid int64) error
	AddCollectionItem(ctx context.Context, biz string, bizId, cid int64, uid int64) error
	Get(ctx context.Context, biz string, bizId int64) (domain.Interactive, error)
	Liked(ctx context.Context, biz string, id int64, uid int64) (bool, error)
	Collected(ctx context.Context, biz string, id int64, uid int64) (bool, error)
	GetByIds(ctx context.Context, biz string, ids []int64) ([]domain.Interactive, error)
}

type CachedReadCntRepository struct {
	cache cache.InteractiveCache
	dao   dao2.InteractiveDAO
	l     logger.Logger
}

func NewCachedInteractiveRepository(dao dao2.InteractiveDAO, cache cache.InteractiveCache, l logger.Logger) InteractiveRepository {
	return &CachedReadCntRepository{
		dao:   dao,
		cache: cache,
		l:     l,
	}
}

func (c *CachedReadCntRepository) GetByIds(ctx context.Context, biz string, ids []int64) ([]domain.Interactive, error) {
	vals, err := c.dao.GetByIds(ctx, biz, ids)
	if err != nil {
		return nil, err
	}
	return slice.Map[dao2.Interactive, domain.Interactive](vals,
		func(idx int, src dao2.Interactive) domain.Interactive {
			return c.toDomain(src)
		}), nil
}

// IncrReadCnt 增加阅读计数，先更新数据库，再更新缓存
// 参数:
//
//	ctx: 上下文对象，用于控制请求的生命周期（如超时、取消等）
//	biz: 业务类型（例如：文章、视频等）
//	bizId: 业务对象 ID（例如：文章 ID 或视频 ID）
func (c *CachedReadCntRepository) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	// 首先更新数据库中的阅读计数
	err := c.dao.IncrReadCnt(ctx, biz, bizId)
	if err != nil {
		return err // 如果数据库操作失败，则直接返回错误
	}

	// 然后更新缓存中的阅读计数（如果缓存中存在对应的键）
	// 这部分会存在一定的不一致风险，但由于阅读数不要求完全准确，业务上是可以容忍的
	return c.cache.IncrReadCntIfPresent(ctx, biz, bizId)
}

// BatchIncrReadCnt 批量增加文章的阅读计数
// 该方法会同时更新数据库和缓存中的阅读计数，更新操作会在后台并发执行
func (c *CachedReadCntRepository) BatchIncrReadCnt(ctx context.Context, bizs []string, bizIds []int64) error {
	// 启动一个新的 goroutine 用于异步更新缓存中的阅读计数
	//go func() {
	//	// 遍历每个业务和对应的业务ID
	//	for i := 0; i < len(bizs); i++ {
	//		// 更新缓存中的阅读计数，如果更新失败，则记录错误日志
	//		err := c.redisCache.IncrReadCntIfPresent(ctx, bizs[i], bizIds[i])
	//		if err != nil {
	//			// 记录缓存更新失败的日志
	//			c.l.Error("更新缓存阅读计数失败",
	//				logger.Int64("bizId", bizIds[i]),
	//				logger.String("biz", bizs[i]),
	//				logger.Error(err))
	//		}
	//	}
	//}()
	// 调用数据库层的方法批量更新数据库中的阅读计数
	return c.dao.BatchIncrReadCnt(ctx, bizs, bizIds)
}

// IncrLike 增加点赞操作：首先插入数据库中的点赞信息，然后更新缓存中的点赞计数
// 参数:
//
//	ctx: 上下文对象，用于控制请求生命周期（例如取消或超时）
//	biz: 业务标识，用于区分不同业务（例如：文章、视频等）
//	bizId: 业务ID，用于标识特定的业务对象（例如：文章ID、视频ID等）
//	uid: 用户ID，用于标识哪个用户进行了点赞操作
func (c *CachedReadCntRepository) IncrLike(ctx context.Context, biz string, bizId, uid int64) error {
	// 调用 DAO 层插入点赞信息到数据库
	err := c.dao.InsertLikeInfo(ctx, biz, bizId, uid)
	if err != nil {
		// 如果插入失败，返回错误
		return err
	}

	// 插入数据库成功后，调用缓存层更新缓存中的点赞计数
	return c.cache.IncrLikeCntIfPresent(ctx, biz, bizId)
}

// DecrLike 执行减少点赞操作：首先删除数据库中的点赞信息，然后更新缓存中的点赞计数
// 参数:
//
//	ctx: 上下文对象，用于控制请求生命周期（例如取消或超时）
//	biz: 业务标识，用于区分不同业务（例如：文章、视频等）
//	bizId: 业务ID，用于标识特定的业务对象（例如：文章ID、视频ID等）
//	uid: 用户ID，用于标识哪个用户进行了取消点赞操作
func (c *CachedReadCntRepository) DecrLike(ctx context.Context, biz string, bizId, uid int64) error {
	// 调用 DAO 层删除点赞信息从数据库
	err := c.dao.DeleteLikeInfo(ctx, biz, bizId, uid)
	if err != nil {
		// 如果删除失败，返回错误
		return err
	}

	// 删除数据库记录成功后，调用缓存层更新缓存中的点赞计数
	return c.cache.DecrLikeCntIfPresent(ctx, biz, bizId)
}

// AddCollectionItem 增加收藏项，首先插入数据库，然后更新缓存中的收藏计数
// 参数:
//
//	ctx: 上下文对象，用于控制请求的生命周期（如超时、取消等）
//	biz: 业务类型（例如：文章、视频等）
//	bizId: 业务对象 ID（例如：文章 ID 或视频 ID）
//	cid: 收藏记录 ID（例如：用户收藏的标识）
//	uid: 用户 ID（表示哪个用户执行了收藏操作）
func (c *CachedReadCntRepository) AddCollectionItem(ctx context.Context, biz string, bizId, cid int64, uid int64) error {
	// 1. 插入用户收藏信息到数据库
	err := c.dao.InsertCollectionBiz(ctx, dao2.UserCollectionBiz{
		Biz:   biz,   // 业务类型
		Cid:   cid,   // 收藏记录 ID
		BizId: bizId, // 业务对象 ID
		Uid:   uid,   // 用户 ID
	})
	if err != nil {
		return err // 如果数据库操作失败，则直接返回错误
	}

	// 2. 更新缓存中的收藏计数
	return c.cache.IncrCollectCntIfPresent(ctx, biz, bizId)
}

func (c *CachedReadCntRepository) Get(ctx context.Context, biz string, bizId int64) (domain.Interactive, error) {
	intr, err := c.cache.Get(ctx, biz, bizId)
	if err == nil {
		// 缓存只缓存了具体的数字，但是没有缓存自身有没有点赞的信息
		// 因为一个人反复刷，重复刷一篇文章是小概率的事情
		// 也就是说，你缓存了某个用户是否点赞的数据，命中率会很低
		return intr, nil
	}
	ie, err := c.dao.Get(ctx, biz, bizId)
	if err == nil {
		res := c.toDomain(ie)
		if er := c.cache.Set(ctx, biz, bizId, res); er != nil {
			c.l.Error("回写缓存失败",
				logger.Int64("bizId", bizId),
				logger.String("biz", biz),
				logger.Error(er))
		}
		return res, nil
	}
	return domain.Interactive{}, err
}

func (c *CachedReadCntRepository) Liked(ctx context.Context, biz string, id int64, uid int64) (bool, error) {
	_, err := c.dao.GetLikeInfo(ctx, biz, id, uid)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrDataNotFound):
		return false, nil
	default:
		return false, err
	}
}

func (c *CachedReadCntRepository) Collected(ctx context.Context, biz string, id int64, uid int64) (bool, error) {
	_, err := c.dao.GetCollectionInfo(ctx, biz, id, uid)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrDataNotFound):
		return false, nil
	default:
		return false, err
	}
}

func (c *CachedReadCntRepository) toDomain(intr dao2.Interactive) domain.Interactive {
	return domain.Interactive{
		BizId:      intr.BizId,
		LikeCnt:    intr.LikeCnt,
		CollectCnt: intr.CollectCnt,
		ReadCnt:    intr.ReadCnt,
	}
}
