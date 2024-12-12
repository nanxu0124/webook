package service

import (
	"context"
	"webook/internal/domain"
	"webook/internal/repository"
)

type ArticleService interface {
	Save(ctx context.Context, art domain.Article) (int64, error)
	Publish(ctx context.Context, art domain.Article) (int64, error)
	Withdraw(ctx context.Context, uid, id int64) error

	List(ctx context.Context, author int64, offset, limit int) ([]domain.Article, error)
	GetById(ctx context.Context, id int64) (domain.Article, error)

	// GetPublishedById 查找已经发表的
	// 正常来说在微服务架构下，读者服务和创作者服务会是两个独立的服务
	// 单体应用下可以混在一起
	GetPublishedById(ctx context.Context, id int64) (domain.Article, error)
}

type articleService struct {
	repo repository.ArticleRepository
}

func NewArticleService(authorRepo repository.ArticleRepository) ArticleService {
	return &articleService{
		repo: authorRepo,
	}
}

func (svc *articleService) GetPublishedById(ctx context.Context, id int64) (domain.Article, error) {
	return svc.repo.GetPublishedById(ctx, id)
}

func (svc *articleService) GetById(ctx context.Context, id int64) (domain.Article, error) {
	return svc.repo.GetById(ctx, id)
}

func (svc *articleService) List(ctx context.Context, author int64, offset, limit int) ([]domain.Article, error) {
	return svc.repo.List(ctx, author, offset, limit)
}

func (svc *articleService) Withdraw(ctx context.Context, uid, id int64) error {
	return svc.repo.SyncStatus(ctx, uid, id, domain.ArticleStatusPrivate)
}

func (svc *articleService) Publish(ctx context.Context, art domain.Article) (int64, error) {
	art.Status = domain.ArticleStatusPublished
	return svc.repo.Sync(ctx, art)
}

func (svc *articleService) Save(ctx context.Context, art domain.Article) (int64, error) {
	art.Status = domain.ArticleStatusUnpublished
	if art.Id > 0 {
		err := svc.repo.Update(ctx, art)
		return art.Id, err
	}
	return svc.repo.Create(ctx, art)
}
