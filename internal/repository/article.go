package repository

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"webook/internal/domain"
	"webook/internal/repository/dao/article"
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
}

func NewArticleRepository(dao article.ArticleDAO, userRepo UserRepository) ArticleRepository {
	return &CachedArticleRepository{
		dao:      dao,
		userRepo: userRepo,
	}
}

func (repo *CachedArticleRepository) GetPublishedById(ctx context.Context, id int64) (domain.Article, error) {
	art, err := repo.dao.GetPubById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	user, err := repo.userRepo.FindById(ctx, art.AuthorId)
	if err != nil {
		return domain.Article{}, err
	}
	res := domain.Article{
		Id:      art.Id,
		Title:   art.Title,
		Status:  domain.ArticleStatus(art.Status),
		Content: art.Content,
		Author: domain.Author{
			Id:   user.Id,
			Name: user.Nickname,
		},
	}

	return res, nil
}

func (repo *CachedArticleRepository) GetById(ctx context.Context, id int64) (domain.Article, error) {
	art, err := repo.dao.GetById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	return repo.toDomain(art), nil
}

func (repo *CachedArticleRepository) List(ctx context.Context, author int64, offset int, limit int) ([]domain.Article, error) {
	arts, err := repo.dao.GetByAuthor(ctx, author, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map[article.Article, domain.Article](arts,
		func(idx int, src article.Article) domain.Article {
			return repo.toDomain(src)
		}), nil
}

func (repo *CachedArticleRepository) SyncStatus(ctx context.Context, uid, id int64, status domain.ArticleStatus) error {
	return repo.dao.SyncStatus(ctx, uid, id, status.ToUint8())
}

func (repo *CachedArticleRepository) Sync(ctx context.Context, art domain.Article) (int64, error) {
	return repo.dao.Sync(ctx, repo.toEntity(art))
}

func (repo *CachedArticleRepository) Create(ctx context.Context, art domain.Article) (int64, error) {
	return repo.dao.Create(ctx, repo.toEntity(art))
}

func (repo *CachedArticleRepository) Update(ctx context.Context, art domain.Article) error {
	return repo.dao.UpdateById(ctx, repo.toEntity(art))
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
