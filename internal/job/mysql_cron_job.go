package job

import (
	"context"
	"fmt"
	"golang.org/x/sync/semaphore"
	"time"
	"webook/internal/domain"
	"webook/internal/service"
	"webook/pkg/logger"
)

type Executor interface {
	Name() string
	// Exec ctx 是整个任务调度的上下文
	// 当从 ctx.Done 有信号的时候，就需要考虑结束执行
	// 具体实现来控制
	Exec(ctx context.Context, j domain.CronJob) error
}

type LocalFuncExecutor struct {
	funcs map[string]func(ctx context.Context, j domain.CronJob) error
}

func NewLocalFuncExecutor() *LocalFuncExecutor {
	return &LocalFuncExecutor{
		funcs: make(map[string]func(ctx context.Context, j domain.CronJob) error),
	}
}

func (l *LocalFuncExecutor) AddLocalFunc(name string, fn func(ctx context.Context, j domain.CronJob) error) {
	l.funcs[name] = fn
}

func (l *LocalFuncExecutor) Name() string {
	return "local"
}

func (l *LocalFuncExecutor) Exec(ctx context.Context, j domain.CronJob) error {
	fn, ok := l.funcs[j.Name]
	if !ok {
		return fmt.Errorf("未知任务：没有注册本地方法 %s", j.Name)
	}
	return fn(ctx, j)
}

type Scheduler struct {
	execs     map[string]Executor
	svc       service.CronJobService
	interval  time.Duration
	dbTimeout time.Duration
	l         logger.Logger
	limiter   *semaphore.Weighted
}

func NewScheduler(svc service.CronJobService, l logger.Logger) *Scheduler {
	return &Scheduler{
		execs:     make(map[string]Executor, 8),
		svc:       svc,
		interval:  time.Second,
		dbTimeout: time.Second,
		l:         l,
		// 最多只有 100 个goroutine
		limiter: semaphore.NewWeighted(100),
	}
}

type CronJob = domain.CronJob

func (s *Scheduler) RegisterJob(ctx context.Context, j CronJob) error {
	return s.svc.AddJob(ctx, j)
}

func (s *Scheduler) RegisterExecutor(exec Executor) {
	s.execs[exec.Name()] = exec
}

// Schedule 开始调度。当被取消，或者超时的时候，就会结束调度
func (s *Scheduler) Schedule(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			// 已经超时了，或者被取消运行，大多数时候，都是被取消了，或者说关闭了
			return ctx.Err()
		}

		// 超过规定数目就会阻塞在这里
		err := s.limiter.Acquire(ctx, 1)
		if err != nil {
			// 正常来说，只有 ctx 超时或者取消才会进来这里
			return err
		}

		// 抢占，获得可以运行的资格
		// 数据库查询的时候，dbTimeout 控制超时
		dbCtx, cancel := context.WithTimeout(ctx, s.dbTimeout)
		j, err := s.svc.Preempt(dbCtx)
		cancel()
		if err != nil {
			// 没有抢占到，进入下一个循环
			// 这里可以考虑睡眠一段时间
			// 也可以进一步细分不同的错误，如果是可以容忍的错误，就继续，不然就直接 return
			time.Sleep(s.interval)
			continue
		}

		exec, ok := s.execs[j.Executor]
		if !ok {
			// 不支持的执行方式。
			s.l.Error("未找到对应的执行器", logger.String("executor", j.Executor))
			j.CancelFunc()
			continue
		}

		go func() {
			// 异步执行具体任务
			// 不要阻塞主循环
			defer func() {
				j.CancelFunc()
				s.limiter.Release(1)
			}()

			err1 := exec.Exec(ctx, j)
			if err1 != nil {
				s.l.Error("调度任务执行失败", logger.Int64("id", j.Id), logger.Error(err1))
				return
			}
			err1 = s.svc.ResetNextTime(ctx, j)
			if err1 != nil {
				s.l.Error("更新下一次的执行失败", logger.Error(err1))
			}
		}()
	}
}
