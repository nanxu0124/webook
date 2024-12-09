package repository

import (
	"context"
	"webook/internal/domain"
	"webook/internal/repository/dao"
)

type ArticleRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, art domain.Article) error

	// Sync 本身要求先保存到制作库，再同步到线上库
	Sync(ctx context.Context, art domain.Article) (int64, error)

	// SyncStatus 仅仅同步状态
	SyncStatus(ctx context.Context, uid, id int64, status domain.ArticleStatus) error
}

type CachedArticleRepository struct {
	dao dao.ArticleDAO
}

func NewArticleRepository(dao dao.ArticleDAO) ArticleRepository {
	return &CachedArticleRepository{
		dao: dao,
	}
}

func (repo *CachedArticleRepository) SyncStatus(ctx context.Context, uid, id int64, status domain.ArticleStatus) error {
	return repo.dao.SyncStatus(ctx, uid, id, status.ToUint8())
}

func (repo *CachedArticleRepository) Sync(ctx context.Context, art domain.Article) (int64, error) {
	return repo.dao.SyncClosure(ctx, repo.toEntity(art))
}

func (repo *CachedArticleRepository) Create(ctx context.Context, art domain.Article) (int64, error) {
	return repo.dao.Create(ctx, repo.toEntity(art))
}

func (repo *CachedArticleRepository) Update(ctx context.Context, art domain.Article) error {
	return repo.dao.UpdateById(ctx, repo.toEntity(art))
}

func (repo *CachedArticleRepository) toEntity(art domain.Article) dao.Article {
	return dao.Article{
		Id:       art.Id,
		Title:    art.Title,
		Content:  art.Content,
		AuthorId: art.Author.Id,
		Status:   art.Status.ToUint8(),
	}
}
