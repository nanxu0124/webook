package service

import (
	"context"
	"time"
	"webook/internal/domain"
	"webook/internal/repository"
	"webook/pkg/logger"
)

//go:generate mockgen -source=./cron_job.go -package=svcmocks -destination=mocks/cron_job.mock.go CronJobService
type CronJobService interface {
	// Preempt 抢占
	Preempt(ctx context.Context) (domain.CronJob, error)
	ResetNextTime(ctx context.Context, job domain.CronJob) error
	AddJob(ctx context.Context, j domain.CronJob) error
}

type cronJobService struct {
	repo            repository.CronJobRepository
	l               logger.Logger
	refreshInterval time.Duration
}

func NewCronJobService(repo repository.CronJobRepository, l logger.Logger) CronJobService {
	return &cronJobService{
		repo:            repo,
		l:               l,
		refreshInterval: time.Second * 10,
	}
}

func (c *cronJobService) ResetNextTime(ctx context.Context, job domain.CronJob) error {
	// 计算下一次的时间
	t := job.Next(time.Now())
	// 我们认为这是不需要继续执行了
	if !t.IsZero() {
		return c.repo.UpdateNextTime(ctx, job.Id, job.Version, t)
	}
	return nil
}

func (c *cronJobService) AddJob(ctx context.Context, j domain.CronJob) error {
	j.NextTime = j.Next(time.Now())
	return c.repo.AddJob(ctx, j)
}

func (c *cronJobService) Preempt(ctx context.Context) (domain.CronJob, error) {
	j, err := c.repo.Preempt(ctx)
	if err != nil {
		return domain.CronJob{}, err
	}
	ticker := time.NewTicker(c.refreshInterval)
	go func() {
		// 启动一个 goroutine 开始续约，也就是在持续占有期间
		for range ticker.C {
			c.refresh(j.Id, j.Version)
		}
	}()

	// 只能调用一次，也就是放弃续约。这时候要把状态还原回去
	j.CancelFunc = func() {
		ticker.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err := c.repo.Release(ctx, j.Id, j.Version)
		if err != nil {
			c.l.Error("释放任务失败", logger.Error(err), logger.Int64("id", j.Id))
		}
	}
	return j, nil
}

func (c *cronJobService) refresh(id int64, version int64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := c.repo.UpdateUtime(ctx, id, version)
	if err != nil {
		c.l.Error("续约失败",
			logger.Int64("jid", id),
			logger.Error(err))
	}
}
