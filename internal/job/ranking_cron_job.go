package job

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
	"strconv"
	"time"
	"webook/pkg/logger"
)

type CronJobBuilder struct {
	vector *prometheus.SummaryVec
	l      logger.Logger
}

func NewCronJobBuilder(l logger.Logger, opt prometheus.SummaryOpts) *CronJobBuilder {
	vector := prometheus.NewSummaryVec(opt, []string{"name", "success"})
	prometheus.MustRegister()
	return &CronJobBuilder{vector: vector, l: l}
}

func (m *CronJobBuilder) Build(job Job) cron.Job {
	name := job.Name()
	return cronJobAdapterFunc(func() {
		start := time.Now()
		m.l.Debug("任务开始", logger.String("name", name), logger.String("time", start.String()))
		err := job.Run()
		duration := time.Since(start)
		if err != nil {
			m.l.Error("任务执行失败", logger.String("name", name), logger.Error(err))
		}
		m.l.Debug("任务结束", logger.String("name", name))
		m.vector.WithLabelValues(name, strconv.FormatBool(err == nil)).Observe(float64(duration.Milliseconds()))
	})
}

var _ cron.Job = (*cronJobAdapterFunc)(nil)

type cronJobAdapterFunc func()

func (c cronJobAdapterFunc) Run() {
	c()
}
