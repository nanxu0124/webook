package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

// PrometheusBuilder 用于构建 Prometheus 指标，主要用于统计响应时间和活跃请求数等信息。
type PrometheusBuilder struct {
	// Namespace 是 Prometheus 指标的命名空间（可选），可以用来区分不同的服务或模块。
	Namespace string

	// Subsystem 是 Prometheus 指标的子系统（可选），用于进一步组织不同的指标。
	Subsystem string

	// Name 是指当前指标的名称（必选），每个指标必须有一个唯一的名字。
	Name string

	// Help 是该指标的描述信息（可选），有助于说明该指标的用途。
	Help string

	// InstanceID 是该实例的唯一标识，可以考虑使用本地 IP 或在启动时配置一个唯一 ID。
	InstanceID string
}

// BuildResponseTime 用于创建一个 Gin 中间件，统计 HTTP 请求的响应时间。
// 这个指标是一个总结类型（Summary），适用于统计响应时间，并按不同的百分位数进行统计。
func (p *PrometheusBuilder) BuildResponseTime() gin.HandlerFunc {
	// 定义统计指标的标签：method、pattern、status
	labels := []string{"method", "pattern", "status"}

	// 创建一个 SummaryVec 类型的指标，用于统计响应时间
	SummaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		// 指标的命名空间、子系统和名称
		Namespace: p.Namespace,
		Subsystem: p.Subsystem,
		Name:      p.Name + "_resp_time",
		Help:      p.Help,
		// 固定标签，表示该指标的实例 ID
		ConstLabels: map[string]string{
			"instance_id": p.InstanceID,
		},
		// 定义响应时间的百分位数目标：例如 50%、75%、90%、99% 和 99.9% 响应时间
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.75:  0.01,
			0.90:  0.01,
			0.99:  0.001,
			0.999: 0.0001,
		},
	}, labels)

	// 注册该指标到 Prometheus
	prometheus.MustRegister(SummaryVec)

	// 返回一个 Gin 中间件函数
	return func(ctx *gin.Context) {
		// 记录请求的 HTTP 方法
		method := ctx.Request.Method

		// 记录请求开始时间
		start := time.Now()

		// 使用 defer 来确保请求结束后统计响应时间
		defer func() {
			// 根据请求方法、路径和响应状态码，统计响应时间
			SummaryVec.WithLabelValues(method, ctx.FullPath(),
				strconv.Itoa(ctx.Writer.Status())).
				Observe(float64(time.Since(start).Milliseconds()))
		}()

		// 继续处理请求
		ctx.Next()
	}
}

// BuildActiveRequest 用于创建一个 Gin 中间件，统计当前活跃的请求数。
// 这个指标是一个 Gauge 类型的指标，适用于统计实时的请求数量。
func (p *PrometheusBuilder) BuildActiveRequest() gin.HandlerFunc {
	// 创建一个 Gauge 类型的指标，用于统计活跃请求数
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		// 指标的命名空间、子系统和名称
		Namespace: p.Namespace,
		Subsystem: p.Subsystem,
		Name:      p.Name + "_active_req",
		Help:      p.Help,
		// 固定标签，表示该指标的实例 ID
		ConstLabels: map[string]string{
			"instance_id": p.InstanceID,
		},
	})

	// 注册该指标到 Prometheus
	prometheus.MustRegister(gauge)

	// 返回一个 Gin 中间件函数
	return func(ctx *gin.Context) {
		// 每当有请求进入时，活跃请求数加 1
		gauge.Inc()

		// 在请求完成后，活跃请求数减 1
		defer gauge.Dec()

		// 继续处理请求
		ctx.Next()
	}
}
