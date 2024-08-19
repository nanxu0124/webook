package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"net/http"
	ijwt "webook/internal/web/jwt"
)

type LoginJWTMiddlewareBuilder struct {
	paths []string
	cmd   redis.Cmdable
	ijwt.Handler
}

func NewLoginJWTMiddlewareBuilder(jwtHdl ijwt.Handler) *LoginJWTMiddlewareBuilder {
	return &LoginJWTMiddlewareBuilder{
		Handler: jwtHdl,
	}
}

func (l *LoginJWTMiddlewareBuilder) IgnorePaths(path string) *LoginJWTMiddlewareBuilder {
	l.paths = append(l.paths, path)
	return l
}

func (l *LoginJWTMiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 不需要登录校验
		for _, path := range l.paths {
			if ctx.Request.URL.Path == path {
				return
			}
		}

		// 用JWT来进行登录校验
		tokenStr := l.ExtractToken(ctx)
		claims := &ijwt.UserClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return ijwt.AtKey, nil
		})
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if token == nil || !token.Valid || claims.Uid == 0 {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if claims.UserAgent != ctx.Request.UserAgent() {
			// 登录设备不一样 可能 token 被获取
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 判断是否退出登录
		err = l.CheckSession(ctx, claims.Ssid)
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 因为使用了长短token机制，这里的刷新机制用不上了
		// 每 10s 刷新一次
		//if claims.ExpiresAt.Sub(time.Now()) < time.Second*50 {
		//	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
		//	tokenStr, err = token.SignedString([]byte("HjVq4YPaFoMywaPfKrhCbLIxpzGHA0Vf"))
		//	if err != nil {
		//		// 记录日志
		//		log.Println("jwt 续约失败", err)
		//	}
		//	ctx.Header("x-jwt-token", tokenStr)
		//}
		ctx.Set("claims", claims)
	}
}
