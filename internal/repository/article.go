package repository

import (
	"context"
	"errors"
	"github.com/ecodeclub/ekit/slice"
	"webook/internal/domain"
	"webook/internal/repository/cache"
	"webook/internal/repository/dao/article"
	"webook/pkg/logger"
)

type ArticleRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, art domain.Article) error

	// Sync 本身要求先保存到制作库，再同步到线上库
	Sync(ctx context.Context, art domain.Article) (int64, error)

	// SyncStatus 仅仅同步状态
	SyncStatus(ctx context.Context, uid, id int64, status domain.ArticleStatus) error

	List(ctx context.Context, author int64, offset int, limit int) ([]domain.Article, error)
	GetById(ctx context.Context, id int64) (domain.Article, error)

	GetPublishedById(ctx context.Context, id int64) (domain.Article, error)
}

type CachedArticleRepository struct {
	dao      article.ArticleDAO
	userRepo UserRepository
	cache    cache.ArticleCache
	l        logger.ZapLogger
}

func NewArticleRepository(dao article.ArticleDAO, userRepo UserRepository, c cache.ArticleCache, logger logger.ZapLogger) ArticleRepository {
	return &CachedArticleRepository{
		dao:      dao,
		userRepo: userRepo,
		cache:    c,
		l:        logger,
	}
}

func (repo *CachedArticleRepository) GetPublishedById(ctx context.Context, id int64) (domain.Article, error) {
	res, err := repo.cache.GetPub(ctx, id)
	if err == nil {
		return res, err
	}
	art, err := repo.dao.GetPubById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	user, err := repo.userRepo.FindById(ctx, art.AuthorId)
	if err != nil {
		return domain.Article{}, err
	}
	res = domain.Article{
		Id:      art.Id,
		Title:   art.Title,
		Status:  domain.ArticleStatus(art.Status),
		Content: art.Content,
		Author: domain.Author{
			Id:   user.Id,
			Name: user.Nickname,
		},
	}
	go func() {
		if err = repo.cache.SetPub(ctx, res); err != nil {
			repo.l.Error("缓存已发表文章失败",
				logger.Error(err), logger.Int64("aid", res.Id))
		}
	}()
	return res, nil
}

func (repo *CachedArticleRepository) GetById(ctx context.Context, id int64) (domain.Article, error) {
	cachedArt, err := repo.cache.Get(ctx, id)
	if err == nil {
		return cachedArt, nil
	}
	art, err := repo.dao.GetById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	return repo.toDomain(art), nil
}

func (repo *CachedArticleRepository) List(ctx context.Context, author int64, offset int, limit int) ([]domain.Article, error) {
	// 只有第一页才走缓存，并且假定一页只有 100 条
	// 也就是说，如果前端允许创作者调整页的大小
	// 那么只有 100 这个页大小这个默认情况下，会走索引
	if offset == 0 && limit == 100 {
		data, err := repo.cache.GetFirstPage(ctx, author)
		if err == nil {
			go func() {
				repo.preCache(ctx, data)
			}()
			return data, nil
		}
		// 这里记录日志
		if !errors.Is(err, cache.ErrKeyNotExist) {
			repo.l.Error("查询缓存文章失败",
				logger.Int64("author", author), logger.Error(err))
		}
	}
	// 慢路径
	arts, err := repo.dao.GetByAuthor(ctx, author, offset, limit)
	if err != nil {
		return nil, err
	}
	res := slice.Map[article.Article, domain.Article](arts,
		func(idx int, src article.Article) domain.Article {
			return repo.toDomain(src)
		})
	// 一般都是让调用者来控制是否异步
	go func() {
		repo.preCache(ctx, res)
	}()
	// 这个也可以做成异步的
	err = repo.cache.SetFirstPage(ctx, author, res)
	if err != nil {
		repo.l.Error("刷新第一页文章的缓存失败",
			logger.Int64("author", author), logger.Error(err))
	}
	return res, nil
}

func (repo *CachedArticleRepository) preCache(ctx context.Context, arts []domain.Article) {
	// 1MB
	const contentSizeThreshold = 1024 * 1024
	if len(arts) > 0 && len(arts[0].Content) <= contentSizeThreshold {
		// 你也可以记录日志
		if err := repo.cache.Set(ctx, arts[0]); err != nil {
			repo.l.Error("提前准备缓存失败", logger.Error(err))
		}
	}
}

func (repo *CachedArticleRepository) SyncStatus(ctx context.Context, uid, id int64, status domain.ArticleStatus) error {
	return repo.dao.SyncStatus(ctx, uid, id, status.ToUint8())
}

func (repo *CachedArticleRepository) Sync(ctx context.Context, art domain.Article) (int64, error) {
	id, err := repo.dao.Sync(ctx, repo.toEntity(art))
	if err != nil {
		return 0, err
	}
	go func() {
		author := art.Author.Id
		err = repo.cache.DelFirstPage(ctx, author)
		if err != nil {
			repo.l.Error("删除第一页缓存失败",
				logger.Int64("author", author), logger.Error(err))
		}
		user, err := repo.userRepo.FindById(ctx, author)
		if err != nil {
			repo.l.Error("提前设置缓存准备用户信息失败",
				logger.Int64("uid", author), logger.Error(err))
		}
		art.Author = domain.Author{
			Id:   user.Id,
			Name: user.Nickname,
		}
		err = repo.cache.SetPub(ctx, art)
		if err != nil {
			repo.l.Error("提前设置缓存失败",
				logger.Int64("author", author), logger.Error(err))
		}
	}()
	return id, nil
}

func (repo *CachedArticleRepository) Create(ctx context.Context, art domain.Article) (int64, error) {
	id, err := repo.dao.Create(ctx, repo.toEntity(art))
	if err != nil {
		return 0, err
	}
	author := art.Author.Id
	err = repo.cache.DelFirstPage(ctx, author)
	if err != nil {
		repo.l.Error("删除缓存失败",
			logger.Int64("author", author), logger.Error(err))
	}
	return id, nil
}

func (repo *CachedArticleRepository) Update(ctx context.Context, art domain.Article) error {
	err := repo.dao.UpdateById(ctx, repo.toEntity(art))
	if err != nil {
		return err
	}
	author := art.Author.Id
	err = repo.cache.DelFirstPage(ctx, author)
	if err != nil {
		repo.l.Error("删除缓存失败",
			logger.Int64("author", author), logger.Error(err))
	}
	return nil
}

func (repo *CachedArticleRepository) toEntity(art domain.Article) article.Article {
	return article.Article{
		Id:       art.Id,
		Title:    art.Title,
		Content:  art.Content,
		AuthorId: art.Author.Id,
		Status:   art.Status.ToUint8(),
	}
}

func (repo *CachedArticleRepository) toDomain(art article.Article) domain.Article {
	return domain.Article{
		Id:      art.Id,
		Title:   art.Title,
		Status:  domain.ArticleStatus(art.Status),
		Content: art.Content,
		Author: domain.Author{
			Id: art.AuthorId,
		},
	}
}
