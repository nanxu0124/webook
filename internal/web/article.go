package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"webook/internal/domain"
	"webook/internal/service"
	"webook/internal/web/jwt"
	"webook/pkg/logger"
)

type ArticleHandler struct {
	svc service.ArticleService
	l   logger.Logger
}

func NewArticleHandler(svc service.ArticleService, l logger.Logger) *ArticleHandler {
	return &ArticleHandler{
		svc: svc,
		l:   l,
	}
}

func (hdl *ArticleHandler) RegisterRoutes(s *gin.Engine) {
	g := s.Group("/articles")
	g.POST("/edit", hdl.Edit)
	g.POST("/publish", hdl.Publish)
	g.POST("/withdraw", hdl.Withdraw)
}

func (a *ArticleHandler) Withdraw(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		a.l.Error("反序列化请求失败", logger.Error(err))
		return
	}

	usr, ok := ctx.MustGet("user").(jwt.UserClaims)
	if !ok {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		a.l.Error("获得用户会话信息失败")
		return
	}

	if err := a.svc.Withdraw(ctx, usr.Id, req.Id); err != nil {
		a.l.Error("设置为仅自己可见失败", logger.Error(err),
			logger.Field{Key: "id", Value: req.Id})
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "OK",
	})
}

func (hdl *ArticleHandler) Publish(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		hdl.l.Error("反序列化请求失败", logger.Error(err))
		return
	}

	usr, ok := ctx.MustGet("user").(jwt.UserClaims)
	if !ok {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		hdl.l.Error("获得用户会话信息失败")
		return
	}

	id, err := hdl.svc.Publish(ctx, req.toDomain(usr.Id))
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		hdl.l.Error("发表失败", logger.Error(err))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Data: id,
	})
}

func (hdl *ArticleHandler) Edit(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		hdl.l.Error("反序列化请求失败", logger.Error(err))
		return
	}

	usr, ok := ctx.MustGet("user").(jwt.UserClaims)
	if !ok {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		hdl.l.Error("获得用户会话信息失败")
		return
	}

	id, err := hdl.svc.Save(ctx, req.toDomain(usr.Id))
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		hdl.l.Error("保存数据失败", logger.Field{Key: "error", Value: err})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Data: id,
	})
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
