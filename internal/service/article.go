package service

import (
	"context"
	"time"
	"webook/internal/domain"
	eventsArticle "webook/internal/events/article"
	"webook/internal/repository"
	"webook/pkg/logger"
)

//go:generate mockgen -source=./article.go -package=svcmocks -destination=mocks/article.mock.go ArticleService
type ArticleService interface {
	Save(ctx context.Context, art domain.Article) (int64, error)
	Publish(ctx context.Context, art domain.Article) (int64, error)
	Withdraw(ctx context.Context, uid, id int64) error

	List(ctx context.Context, author int64, offset, limit int) ([]domain.Article, error)
	GetById(ctx context.Context, id int64) (domain.Article, error)

	// GetPublishedById 查找已经发表的
	// 正常来说在微服务架构下，读者服务和创作者服务会是两个独立的服务
	// 单体应用下可以混在一起
	GetPublishedById(ctx context.Context, id int64, uid int64) (domain.Article, error)

	// ListPub 根据更新时间来分页，更新时间必须小于 startTime
	ListPub(ctx context.Context, startTime time.Time, offset, limit int) ([]domain.Article, error)
}

type articleService struct {
	repo     repository.ArticleRepository
	logger   logger.Logger
	producer eventsArticle.Producer
}

func NewArticleService(authorRepo repository.ArticleRepository, logger logger.Logger, producer eventsArticle.Producer) ArticleService {
	return &articleService{
		repo:     authorRepo,
		logger:   logger,
		producer: producer,
	}
}

func (svc *articleService) ListPub(ctx context.Context, startTime time.Time, offset, limit int) ([]domain.Article, error) {
	return svc.repo.ListPub(ctx, startTime, offset, limit)
}

// GetPublishedById 获取已发布的文章信息，并发送阅读事件
//
//	id: 文章 ID
//	uid: 用户 ID，用于标识阅读该文章的用户
func (svc *articleService) GetPublishedById(ctx context.Context, id int64, uid int64) (domain.Article, error) {
	// 从数据库获取已发布的文章信息
	res, err := svc.repo.GetPublishedById(ctx, id)
	if err == nil {
		// 如果文章信息获取成功，则异步发送阅读事件
		go func() {
			// 构造一个阅读事件对象，包含用户 ID 和文章 ID
			er := svc.producer.ProduceReadEvent(ctx, eventsArticle.ReadEvent{
				Uid: uid, // 用户 ID
				Aid: id,  // 文章 ID
			})
			if er != nil {
				// 如果发送阅读事件失败，记录错误日志
				svc.logger.Error("发送消息失败",
					logger.Int64("uid", uid),
					logger.Int64("aid", id),
					logger.Error(err))
			}
		}()
	}
	// 返回获取的文章信息及错误（如果有）
	return res, err
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
