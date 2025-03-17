package web

import (
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"net/http"
	"strconv"
	"time"
	domain2 "webook/interactive/domain"
	service2 "webook/interactive/service"
	"webook/internal/domain"
	"webook/internal/service"
	"webook/internal/web/jwt"
	"webook/pkg/ginx"
	"webook/pkg/logger"
)

type ArticleHandler struct {
	svc     service.ArticleService
	intrSvc service2.InteractiveService
	biz     string
	l       logger.Logger
}

func NewArticleHandler(svc service.ArticleService, intrSvc service2.InteractiveService, l logger.Logger) *ArticleHandler {
	return &ArticleHandler{
		svc:     svc,
		l:       l,
		biz:     "article",
		intrSvc: intrSvc,
	}
}

func (hdl *ArticleHandler) RegisterRoutes(s *gin.Engine) {
	g := s.Group("/articles")
	g.POST("/edit", hdl.Edit)
	g.POST("/publish", hdl.Publish)
	g.POST("/withdraw", hdl.Withdraw)

	g.POST("/list", hdl.List)
	g.GET("/detail/:id", hdl.Detail)

	pub := g.Group("/pub")
	//pub.GET("/pub", a.PubList)
	pub.GET("/:id", ginx.WrapClaims(hdl.PubDetail))
	pub.POST("/like", ginx.WrapClaimsAndReq[LikeReq](hdl.Like))
	pub.POST("/collect", ginx.WrapClaimsAndReq[CollectReq](hdl.Collect))
}

func (hdl *ArticleHandler) Collect(ctx *gin.Context, req CollectReq, uc ginx.UserClaims) (Result, error) {
	err := hdl.intrSvc.Collect(ctx, hdl.biz, req.Id, req.Cid, uc.Id)
	if err != nil {
		return Result{
			Code: 5,
			Msg:  "系统错误",
		}, err
	}
	return Result{Msg: "OK"}, nil
}

func (hdl *ArticleHandler) Like(ctx *gin.Context, req LikeReq, uc ginx.UserClaims) (Result, error) {
	var err error
	if req.Like {
		err = hdl.intrSvc.Like(ctx, hdl.biz, req.Id, uc.Id)
	} else {
		err = hdl.intrSvc.CancelLike(ctx, hdl.biz, req.Id, uc.Id)
	}

	if err != nil {
		return Result{
			Code: 5,
			Msg:  "系统错误",
		}, err
	}
	return Result{Msg: "OK"}, nil
}

func (hdl *ArticleHandler) PubDetail(ctx *gin.Context, uc ginx.UserClaims) (Result, error) {
	idstr := ctx.Param("id")
	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		hdl.l.Error("前端输入的 ID 不对", logger.Error(err))
		return Result{
			Code: 4,
			Msg:  "参数错误",
		}, fmt.Errorf("查询文章详情的 ID %s 不正确, %w", idstr, err)
	}

	// 使用 error group 来同时查询数据
	var (
		eg   errgroup.Group
		art  domain.Article
		intr domain2.Interactive
	)
	eg.Go(func() error {
		var er error
		art, er = hdl.svc.GetPublishedById(ctx, id, uc.Id)
		return er
	})

	eg.Go(func() error {
		var er error
		intr, er = hdl.intrSvc.Get(ctx, hdl.biz, id, uc.Id)
		return er
	})

	err = eg.Wait()

	if err != nil {
		return Result{
			Code: 5,
			Msg:  "系统错误",
		}, fmt.Errorf("获取文章信息失败 %w", err)
	}

	// 现在 service 接入了 kafka， 所以这里不需要异步去操作了
	//
	// 直接异步操作，在确定我们获取到了数据之后再来操作
	//go func() {
	//	err = hdl.intrSvc.IncrReadCnt(ctx, hdl.biz, art.Id)
	//	if err != nil {
	//		hdl.l.Error("增加文章阅读数失败", logger.Error(err))
	//	}
	//}()

	return Result{
		Data: ArticleVo{
			Id:      art.Id,
			Title:   art.Title,
			Status:  art.Status.ToUint8(),
			Content: art.Content,
			// 要把作者信息带出去
			Author:     art.Author.Name,
			Ctime:      art.Ctime.Format(time.DateTime),
			Utime:      art.Utime.Format(time.DateTime),
			ReadCnt:    intr.ReadCnt,
			CollectCnt: intr.CollectCnt,
			LikeCnt:    intr.LikeCnt,
			Liked:      intr.Liked,
			Collected:  intr.Collected,
		},
	}, nil
}

func (hdl *ArticleHandler) Detail(ctx *gin.Context) {
	idstr := ctx.Param("id")
	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "参数错误",
		})
		hdl.l.Error("前端输入的 ID 不对", logger.Error(err))
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
	art, err := hdl.svc.GetById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		hdl.l.Error("获得文章信息失败", logger.Error(err))
		return
	}
	if art.Author.Id != usr.Id {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "输入有误",
		})
		hdl.l.Error("非法访问文章，创作者 ID 不匹配", logger.Int64("uid", usr.Id))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Data: ArticleVo{
			Id:      art.Id,
			Title:   art.Title,
			Status:  art.Status.ToUint8(),
			Content: art.Content,
			Ctime:   art.Ctime.Format(time.DateTime),
			Utime:   art.Utime.Format(time.DateTime),
		},
	})
}

func (hdl *ArticleHandler) List(ctx *gin.Context) {
	type Req struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	}

	var req Req
	if err := ctx.Bind(&req); err != nil {
		hdl.l.Error("反序列化请求失败", logger.Error(err))
		return
	}

	// 对于批量接口来说，要小心批次大小
	if req.Limit > 100 {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "请求有误",
		})
		hdl.l.Error("获得用户会话信息失败")
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
	arts, err := hdl.svc.List(ctx, usr.Id, req.Offset, req.Limit)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		hdl.l.Error("获得用户会话信息失败")
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Data: slice.Map[domain.Article, ArticleVo](arts,
			func(idx int, src domain.Article) ArticleVo {
				return ArticleVo{
					Id:       src.Id,
					Title:    src.Title,
					Abstract: src.Abstract(),
					Status:   src.Status.ToUint8(),
					Ctime:    src.Ctime.Format(time.DateTime),
					Utime:    src.Utime.Format(time.DateTime),
				}
			}),
	})
}

func (hdl *ArticleHandler) Withdraw(ctx *gin.Context) {
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

	if err := hdl.svc.Withdraw(ctx, usr.Id, req.Id); err != nil {
		hdl.l.Error("设置为仅自己可见失败", logger.Error(err),
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
