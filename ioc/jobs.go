package ioc

import (
	"webook/internal/job"
	"webook/internal/service"
	"webook/pkg/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
	"time"
)

func InitRankingJob(svc service.RankingService, l logger.Logger) *job.RankingJob {
	return job.NewRankingJob(svc, l, time.Second*30)
}

func InitJobs(l logger.Logger, rankingJob *job.RankingJob) *cron.Cron {
	bd := job.NewCronJobBuilder(l, prometheus.SummaryOpts{
		Namespace: "webook_server",
		Subsystem: "webook",
		Name:      "cron_job",
		Help:      "榜单定时任务",
	})
	expr := cron.New(cron.WithSeconds())
	_, err := expr.AddJob("@every 1m", bd.Build(rankingJob))
	if err != nil {
		panic(err)
	}
	return expr
}
