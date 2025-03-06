package job

// Job 为了便于控制（方便扩展），我们使用自己的接口
// 在这个基础上，
// 你可以考虑引入重试、监控和告警等扩展实现（都是装饰器）
type Job interface {
	Name() string
	Run() error
}
