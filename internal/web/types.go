package web

import (
	"github.com/gin-gonic/gin"
	"webook/internal/domain"
)

// Result  API 响应的统一格式
type Result struct {
	Code int    `json:"code"` // 响应状态码，通常为 HTTP 状态码或自定义错误码
	Msg  string `json:"msg"`  // 响应消息，简短的错误描述或成功信息
	Data any    `json:"data"` // 响应的数据，可以是任意类型，通常是请求的结果
}

// handler 定义了注册路由接口的接口类型
type handler interface {
	RegisterRoutes(s *gin.Engine) // 注册路由的方法，s 是 gin.Engine 实例
}

type ArticleReq struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (req ArticleReq) toDomain(uid int64) domain.Article {
	return domain.Article{
		Id:      req.Id,
		Title:   req.Title,
		Content: req.Content,
		Author: domain.Author{
			Id: uid,
		},
	}
}

type ArticleVo struct {
	Id    int64  `json:"id"`
	Title string `json:"title"`
	// 摘要
	Abstract string `json:"abstract"`
	// 内容
	Content string `json:"content"`
	Status  uint8  `json:"status"`
	Author  string `json:"author"`
	Ctime   string `json:"ctime"`
	Utime   string `json:"utime"`
}
