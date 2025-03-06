package job

import (
	"context"
	"time"
	"webook/internal/service"
	"webook/pkg/logger"
)

type RankingJob struct {
	svc     service.RankingService
	timeout time.Duration
	l       logger.Logger
	key     string
}

func NewRankingJob(svc service.RankingService, l logger.Logger, timeout time.Duration) *RankingJob {
	return &RankingJob{
		svc:     svc,
		timeout: timeout,
		key:     "job:ranking",
		l:       l,
	}
}

func (r *RankingJob) Name() string {
	return "ranking"
}

func (r *RankingJob) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	return r.svc.RankTopN(ctx)
}
