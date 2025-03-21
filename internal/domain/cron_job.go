package domain

import (
	"github.com/robfig/cron/v3"
	"time"
)

type CronJob struct {
	Id int64
	// Job 的名称，必须唯一
	Name string
	// 用来控制同一实例的并发问题
	Version int64
	// 用什么来运行
	Executor   string
	Cfg        string
	Expression string
	NextTime   time.Time

	// 放弃抢占状态
	CancelFunc func()
}

var expr = cron.NewParser(cron.Second | cron.Minute |
	cron.Hour | cron.Dom |
	cron.Month | cron.Dow |
	cron.Descriptor)

func (j CronJob) Next(t time.Time) time.Time {
	s, _ := expr.Parse(j.Expression)
	return s.Next(t)
}
