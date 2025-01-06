package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
	"time"
)

// Callbacks 用于注册数据库操作的 Prometheus 监控指标
type Callbacks struct {
	Namespace  string                 // 指标的命名空间
	Subsystem  string                 // 指标的子系统
	Name       string                 // 指标的名称
	InstanceID string                 // 实例 ID，用于区分不同的实例
	Help       string                 // 指标的帮助信息
	vector     *prometheus.SummaryVec // 用于存储响应时间的 Prometheus SummaryVec 指标
}

// Register 用于在 GORM 的钩子中注册 Prometheus 的监控指标
// 通过这些钩子函数监控数据库的查询、原始操作、增删改等操作的响应时间
func (c *Callbacks) Register(db *gorm.DB) error {
	// 创建一个 SummaryVec 类型的指标，统计数据库操作的响应时间
	vector := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:      c.Name,
			Subsystem: c.Subsystem,
			Namespace: c.Namespace,
			Help:      c.Help,
			ConstLabels: map[string]string{
				"db_name":     db.Name(),    // 数据库名称，作为常量标签
				"instance_id": c.InstanceID, // 实例 ID，作为常量标签
			},
			// Objectives 用于定义不同分位数的目标精度
			Objectives: map[float64]float64{
				0.9:  0.01,
				0.99: 0.001,
			},
		},
		[]string{"type", "table"},
	)

	prometheus.MustRegister(vector)

	c.vector = vector

	// 注册 GORM 回调函数，监控不同数据库操作的时间
	// 查询操作（包括普通查询和原始 SQL 查询）
	err := db.Callback().Query().Before("*").Register("prometheus_query_before", c.before("query"))
	if err != nil {
		return err
	}
	err = db.Callback().Query().After("*").Register("prometheus_query_after", c.after("query"))
	if err != nil {
		return err
	}

	// 原始 SQL 操作的回调函数
	err = db.Callback().Raw().Before("*").Register("prometheus_raw_before", c.before("raw"))
	if err != nil {
		return err
	}
	err = db.Callback().Raw().After("*").Register("prometheus_raw_after", c.after("raw"))
	if err != nil {
		return err
	}

	// 创建操作（INSERT）
	err = db.Callback().Create().Before("*").Register("prometheus_create_before", c.before("create"))
	if err != nil {
		return err
	}
	err = db.Callback().Create().After("*").Register("prometheus_create_after", c.after("create"))
	if err != nil {
		return err
	}

	// 更新操作（UPDATE）
	err = db.Callback().Update().Before("*").Register("prometheus_update_before", c.before("update"))
	if err != nil {
		return err
	}
	err = db.Callback().Update().After("*").Register("prometheus_update_after", c.after("update"))
	if err != nil {
		return err
	}

	// 删除操作（DELETE）
	err = db.Callback().Delete().Before("*").Register("prometheus_delete_before", c.before("delete"))
	if err != nil {
		return err
	}
	err = db.Callback().Delete().After("*").Register("prometheus_delete_after", c.after("delete"))
	if err != nil {
		return err
	}

	// 所有的回调函数都注册完毕，返回 nil 表示成功
	return nil
}

// before 用于在数据库操作之前记录操作开始的时间
// 在执行任何数据库操作前，都会通过此函数来记录当前时间
func (c *Callbacks) before(typ string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		start := time.Now()         // 获取当前时间
		db.Set("start_time", start) // 将开始时间存入 GORM 的上下文中
	}
}

// after 用于在数据库操作之后计算操作的持续时间，并记录到 Prometheus 指标中
// 该函数会计算操作从开始到结束的持续时间，并通过 `SummaryVec` 指标记录下来
func (c *Callbacks) after(typ string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		// 从 GORM 上下文中获取开始时间
		val, _ := db.Get("start_time")
		start, ok := val.(time.Time)
		if !ok {
			// 如果无法获取开始时间，表示系统存在问题，可以在这里记录日志
			return
		}
		duration := time.Since(start) // 计算操作的持续时间
		// 使用 Prometheus 的 SummaryVec 来记录操作的持续时间
		c.vector.WithLabelValues(typ, db.Statement.Table).Observe(float64(duration.Milliseconds()))
	}
}
