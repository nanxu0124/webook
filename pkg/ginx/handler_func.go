package ginx

import (
	"github.com/gin-gonic/gin"
	"net/http"
	ijwt "webook/internal/web/jwt"
	"webook/pkg/logger"
)

var log logger.Logger

func SetLogger(l logger.Logger) {
	log = l
}

// WrapClaimsAndReq 如果做成中间件来源出去，那么直接耦合 UserClaims 也是不好的
func WrapClaimsAndReq[Req any](fn func(*gin.Context, Req, UserClaims) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.Bind(&req); err != nil {
			log.Error("解析请求失败", logger.Error(err))
			return
		}
		// 用包变量来配置，因为泛型的限制，这里只能用包变量
		rawVal, ok := ctx.Get("user")
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("无法获得 claims",
				logger.String("path", ctx.Request.URL.Path))
			return
		}
		// 注意，这里要求放进去 ctx 的不能是*UserClaims
		claims, ok := rawVal.(UserClaims)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("无法获得 claims",
				logger.String("path", ctx.Request.URL.Path))
			return
		}
		res, err := fn(ctx, req, claims)
		if err != nil {
			log.Error("执行业务逻辑失败",
				logger.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

func WrapClaims(fn func(*gin.Context, UserClaims) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 用包变量来配置，因为泛型的限制，这里只能用包变量
		rawVal, ok := ctx.Get("user")
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("无法获得 claims",
				logger.String("path", ctx.Request.URL.Path))
			return
		}
		// 注意，这里要求放进去 ctx 的不能是*UserClaims
		claims := UserClaims{}
		if rawClaims, ok := rawVal.(ijwt.UserClaims); ok {
			claims = UserClaims{
				Id:               rawClaims.Id,
				UserAgent:        rawClaims.UserAgent,
				Ssid:             rawClaims.Ssid,
				RegisteredClaims: rawClaims.RegisteredClaims,
			}
		} else {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("无法获得 claims",
				logger.String("path", ctx.Request.URL.Path))
			return
		}
		res, err := fn(ctx, claims)
		if err != nil {
			log.Error("执行业务逻辑失败",
				logger.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}
