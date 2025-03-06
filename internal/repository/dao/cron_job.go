package dao

import (
	"context"
	"gorm.io/gorm"
	"time"
)

type CronJobDAO interface {
	Preempt(ctx context.Context) (Job, error)
	Release(ctx context.Context, id int64, version int64) error
	UpdateUtime(ctx context.Context, id int64, version int64) error
	UpdateNextTime(ctx context.Context, id int64, version int64, t time.Time) error
	Insert(ctx context.Context, j Job) error
}

type GORMJobDAO struct {
	db *gorm.DB
}

func NewGORMJobDAO(db *gorm.DB) CronJobDAO {
	return &GORMJobDAO{db: db}
}

func (dao *GORMJobDAO) UpdateNextTime(ctx context.Context, id int64, version int64, t time.Time) error {
	return dao.db.WithContext(ctx).Model(&Job{}).
		Where("id = ?", id).Updates(map[string]any{
		"utime":     time.Now().UnixMilli(),
		"next_time": t.UnixMilli(),
	}).Error
}

func (dao *GORMJobDAO) Insert(ctx context.Context, j Job) error {
	now := time.Now().UnixMilli()
	j.Ctime = now
	j.Utime = now
	return dao.db.WithContext(ctx).Create(&j).Error
}

func (dao *GORMJobDAO) UpdateUtime(ctx context.Context, id int64, version int64) error {
	return dao.db.WithContext(ctx).Model(&Job{}).
		Where("id = ?", id).Updates(map[string]any{
		"utime": time.Now().UnixMilli(),
	}).Error
}

func (dao *GORMJobDAO) Release(ctx context.Context, id int64, version int64) error {
	return dao.db.WithContext(ctx).Model(&Job{}).
		Where("id = ?", id).Updates(map[string]any{
		"status": jobStatusWaiting,
		"utime":  time.Now().UnixMilli(),
	}).Error
}

func (dao *GORMJobDAO) Preempt(ctx context.Context) (Job, error) {
	db := dao.db.WithContext(ctx)
	for {
		// 每一个循环都重新计算 time.Now，因为之前可能已经花了一些时间了
		now := time.Now().UnixMilli()
		var j Job
		// 到了调度的时间
		err := db.Where(
			"next_time <= ? AND status = ?",
			now, jobStatusWaiting).First(&j).Error
		if err != nil {
			// 数据库有问题
			return Job{}, err
		}
		// 然后要开始抢占
		// 这里 Version 用来解决并发问题
		res := db.Model(&Job{}).
			Where("id = ? AND version=?", j.Id, j.Version).
			Updates(map[string]any{
				"utime":   now,
				"version": j.Version + 1,
				"status":  jobStatusRunning,
			})
		if res.Error != nil {
			// 数据库错误
			return Job{}, err
		}
		// 抢占成功
		if res.RowsAffected == 1 {
			return j, nil
		}
		// 没有抢占到，也就是同一时刻被人抢走了，那么就下一个循环
	}
}

type Job struct {
	Id         int64 `gorm:"primaryKey,autoIncrement"`
	Name       string
	Executor   string
	Cfg        string
	Expression string

	// Status 用来标记哪些任务可以抢、哪些任务已经被人占着
	Status int

	// NextTime 下一次被调度的时间
	// 判断定时任务有没有到时间
	NextTime int64 `gorm:"index"`

	// Version 用来控制并发问题
	Version int64
	Ctime   int64
	Utime   int64
}

const (
	// 等待被调度，意思就是没有人正在调度
	jobStatusWaiting = iota
	// 已经被 goroutine 抢占了
	jobStatusRunning
	// 不再需要调度了，比如说被终止了，或者被删除了。
	jobStatusEnd
)
