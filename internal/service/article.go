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
}

type articleService struct {
	repo repository.ArticleRepository
}

func NewArticleService(authorRepo repository.ArticleRepository) ArticleService {
	return &articleService{
		repo: authorRepo,
	}
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
