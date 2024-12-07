package accesslog

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"io"
	"time"
)

// MiddlewareBuilder 用于构建中间件，允许配置是否记录请求体和响应体
type MiddlewareBuilder struct {
	logFunc       func(ctx context.Context, al AccessLog) // 日志记录函数
	allowReqBody  bool                                    // 是否允许记录请求体
	allowRespBody bool                                    // 是否允许记录响应体
}

// NewMiddlewareBuilder 创建一个新的 MiddlewareBuilder 实例
func NewMiddlewareBuilder(fn func(ctx context.Context, al AccessLog)) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		logFunc:      fn,    // 使用外部传入的日志记录函数
		allowReqBody: false, // 默认不记录请求体
	}
}

// AllowReqBody 设置是否允许记录请求体
func (b *MiddlewareBuilder) AllowReqBody() *MiddlewareBuilder {
	b.allowReqBody = true
	return b
}

// AllowRespBody 设置是否允许记录响应体
func (b *MiddlewareBuilder) AllowRespBody() *MiddlewareBuilder {
	b.allowRespBody = true
	return b
}

// Build 构建 Gin 的中间件函数
func (b *MiddlewareBuilder) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		start := time.Now()

		// 创建 AccessLog 结构体实例，记录请求方法和路径
		al := AccessLog{
			Method: c.Request.Method,
			Path:   c.Request.URL.Path,
		}

		// 如果允许记录请求体
		if b.allowReqBody && c.Request.Body != nil {
			// 获取请求体的内容
			reqBodyBytes, _ := c.GetRawData()
			// 因为 Request.Body 是一个流对象，读取后会消耗掉，所以需要重置
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
			al.ReqBody = string(reqBodyBytes)
		}

		// 如果允许记录响应体
		if b.allowRespBody {
			// 使用自定义的 responseWriter 来捕获响应内容
			c.Writer = responseWriter{
				ResponseWriter: c.Writer,
				al:             &al,
			}
		}

		// 延迟执行，记录处理时间、状态码等信息
		defer func() {
			// 计算请求处理时间
			duration := time.Since(start)
			al.Duration = duration.String()
			// 调用外部传入的日志记录函数
			b.logFunc(c, al)
		}()

		// 执行后续的业务逻辑
		c.Next()
	}
}

// AccessLog 定义了请求和响应的日志结构体
type AccessLog struct {
	Method     string `json:"method"`      // HTTP 方法
	Path       string `json:"path"`        // 请求路径
	ReqBody    string `json:"req_body"`    // 请求体内容
	Duration   string `json:"duration"`    // 请求处理时间
	StatusCode int    `json:"status_code"` // 响应状态码
	RespBody   string `json:"resp_body"`   // 响应体内容
}

// responseWriter 用于捕获响应体内容并更新 AccessLog
type responseWriter struct {
	al *AccessLog
	gin.ResponseWriter
}

// WriteHeader 重写 WriteHeader 方法，记录响应状态码
func (r responseWriter) WriteHeader(statusCode int) {
	r.al.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// 重写 Write 方法，记录响应体内容
func (r responseWriter) Write(data []byte) (int, error) {
	r.al.RespBody = string(data)
	return r.ResponseWriter.Write(data)
}

// WriteString 重写 WriteString 方法，记录响应体内容
func (r responseWriter) WriteString(data string) (int, error) {
	r.al.RespBody = data
	return r.ResponseWriter.WriteString(data)
}
