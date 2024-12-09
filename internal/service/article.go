package service

import (
	"context"
	"webook/internal/domain"
	"webook/internal/repository"
)

type ArticleService interface {
	Save(ctx context.Context, art domain.Article) (int64, error)
	Publish(ctx context.Context, art domain.Article) (int64, error)
}

type articleService struct {
	repo repository.ArticleRepository
}

func NewArticleService(authorRepo repository.ArticleRepository) ArticleService {
	return &articleService{
		repo: authorRepo,
	}
}

func (svc *articleService) Publish(ctx context.Context, art domain.Article) (int64, error) {
	return svc.repo.Sync(ctx, art)
}

func (svc *articleService) Save(ctx context.Context, art domain.Article) (int64, error) {
	if art.Id > 0 {
		err := svc.repo.Update(ctx, art)
		return art.Id, err
	}
	return svc.repo.Create(ctx, art)
}
