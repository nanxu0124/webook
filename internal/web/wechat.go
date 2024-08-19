package web

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	uuid "github.com/lithammer/shortuuid/v4"
	"net/http"
	"time"
	"webook/internal/service"
	"webook/internal/service/oauth2/wechat"
	ijwt "webook/internal/web/jwt"
)

type OAuth2WechatHandler struct {
	svc     wechat.Service
	userSvc service.UserService
	ijwt.Handler
}

func NewOAuth2WechatHandler(svc wechat.Service, userSvc service.UserService, jwtHdl ijwt.Handler) *OAuth2WechatHandler {
	return &OAuth2WechatHandler{
		svc:     svc,
		userSvc: userSvc,
		Handler: jwtHdl,
	}
}

func (h *OAuth2WechatHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/oauth2/wechat")
	g.GET("/authurl", h.AuthURL)
	g.Any("/callback", h.Callback)
}

type StateClaims struct {
	State string
	jwt.RegisteredClaims
}

func (h *OAuth2WechatHandler) AuthURL(ctx *gin.Context) {
	url, err := h.svc.AuthURL(ctx, "")
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  fmt.Sprintf("构造URL失败:%s", err),
		})
		return
	}

	state := uuid.New()
	err = h.setStateCookie(ctx, state)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  fmt.Sprintf("setStateCookie:%s", err),
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Data: url,
	})
}

func (h *OAuth2WechatHandler) setStateCookie(ctx *gin.Context, state string) error {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, StateClaims{
		State: state,
		RegisteredClaims: jwt.RegisteredClaims{
			// 过期时间，你预期中一个用户完成登录的时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 3)),
		},
	})
	tokenStr, err := token.SignedString([]byte("HjVq4YPaFoMywaPqKrhCbLIxpzGHA0Vf"))
	if err != nil {
		return err
	}
	ctx.SetCookie("jwt-state", tokenStr, 180, "/oauth2/wechat/callback", "", false, false)
	return nil
}

func (h *OAuth2WechatHandler) Callback(ctx *gin.Context) {

	err := h.verifyState(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  fmt.Sprintf("系统错误:%s", err),
		})
		return
	}

	code := ctx.Query("code")
	info, err := h.svc.VerifyCode(ctx, code)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  fmt.Sprintf("系统错误:%s", err),
		})
		return
	}

	u, err := h.userSvc.FindOrCreateByWechat(ctx, info)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  fmt.Sprintf("系统错误:%s", err),
		})
		return
	}
	err = h.SetLoginToken(ctx, u.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  fmt.Sprintf("系统错误:%s", err),
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "OK",
	})
}

func (h *OAuth2WechatHandler) verifyState(ctx *gin.Context) error {
	state := ctx.Query("state")
	ck, err := ctx.Cookie("jwt-state")
	if err != nil {
		return fmt.Errorf("拿不到 state 的 cookie, %w", err)
	}
	var sc StateClaims
	token, err := jwt.ParseWithClaims(ck, &sc, func(token *jwt.Token) (interface{}, error) {
		return []byte("HjVq4YPaFoMywaPqKrhCbLIxpzGHA0Vf"), nil
	})
	if err != nil || !token.Valid {
		return fmt.Errorf("token 已经过期了, %w", err)
	}
	if sc.State != state {
		return errors.New("state 不相等")
	}
	return nil
}
