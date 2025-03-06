package repository

import (
	"context"
	"time"
	"webook/internal/domain"
	"webook/internal/repository/dao"
)

//go:generate mockgen -source=./cron_job.go -package=repomocks -destination=mocks/cron_job.mock.go CronJobRepository
type CronJobRepository interface {
	Preempt(ctx context.Context) (domain.CronJob, error)
	Release(ctx context.Context, id int64, version int64) error
	UpdateUtime(ctx context.Context, id int64, version int64) error
	UpdateNextTime(ctx context.Context, id int64, version int64, t time.Time) error
	AddJob(ctx context.Context, j domain.CronJob) error
}

type cronJobRepositoryImpl struct {
	dao dao.CronJobDAO
}

func NewCronJobRepositoryImpl(dao dao.CronJobDAO) CronJobRepository {
	return &cronJobRepositoryImpl{dao: dao}
}

func (c *cronJobRepositoryImpl) UpdateNextTime(ctx context.Context, id int64, version int64, t time.Time) error {
	return c.dao.UpdateNextTime(ctx, id, version, t)
}

func (c *cronJobRepositoryImpl) AddJob(ctx context.Context, j domain.CronJob) error {
	return c.dao.Insert(ctx, c.toEntity(j))
}

func (c *cronJobRepositoryImpl) UpdateUtime(ctx context.Context, id int64, version int64) error {
	return c.dao.UpdateUtime(ctx, id, version)
}

func (c *cronJobRepositoryImpl) Release(ctx context.Context, id int64, version int64) error {
	return c.dao.Release(ctx, id, version)
}

func (c *cronJobRepositoryImpl) Preempt(ctx context.Context) (domain.CronJob, error) {
	j, err := c.dao.Preempt(ctx)
	if err != nil {
		return domain.CronJob{}, err
	}
	return c.toDomain(j), nil
}

func (c *cronJobRepositoryImpl) toEntity(j domain.CronJob) dao.Job {
	return dao.Job{
		Id:         j.Id,
		Name:       j.Name,
		Expression: j.Expression,
		Cfg:        j.Cfg,
		Executor:   j.Executor,
		NextTime:   j.NextTime.UnixMilli(),
	}
}

func (c *cronJobRepositoryImpl) toDomain(j dao.Job) domain.CronJob {
	return domain.CronJob{
		Id:         j.Id,
		Name:       j.Name,
		Expression: j.Expression,
		Cfg:        j.Cfg,
		Executor:   j.Executor,
		NextTime:   time.UnixMilli(j.NextTime),
	}
}
