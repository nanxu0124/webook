package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
	"webook/internal/web"
)

// JWTLoginMiddlewareBuilder 是一个中间件构建器，用于验证用户请求中的JWT令牌。
type JWTLoginMiddlewareBuilder struct {
}

// Build 方法创建并返回一个Gin的中间件，负责JWT的验证。
func (j *JWTLoginMiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 如果请求的路径是用户注册（/users/signup）或登录（/users/login）
		// 这些接口不需要JWT验证，直接放行
		if ctx.Request.URL.Path == "/users/signup" ||
			ctx.Request.URL.Path == "/users/login" {
			return
		}

		// 从请求头中获取Authorization字段，格式应该是 "Bearer token"
		authCode := ctx.GetHeader("Authorization")
		if authCode == "" {
			// 如果没有提供Authorization头部，返回401 Unauthorized
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 使用空格将Authorization字段分割为两部分，第一部分为"Bearer"，第二部分为token字符串。
		// 通过 SplitN 将字符串分割为最多两部分。
		authSegments := strings.SplitN(authCode, " ", 2)
		if len(authSegments) != 2 {
			// 如果分割结果不是两部分（例如没有空格或没有token），返回401 Unauthorized
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 获取token字符串（即 "Bearer" 后面的部分）
		tokenStr := authSegments[1]

		// 创建UserClaims结构体用于解析token中的claim信息
		uc := web.UserClaims{}

		// 使用jwt.ParseWithClaims解析token并验证其合法性
		// tokenStr是待验证的JWT字符串，uc是用于存放解析后的claims信息
		// web.JWTKey是密钥，用于验证token的签名
		token, err := jwt.ParseWithClaims(tokenStr, &uc, func(token *jwt.Token) (interface{}, error) {
			return web.JWTKey, nil
		})

		if err != nil || !token.Valid {
			// 如果token解析失败或无效，返回401 Unauthorized
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 从UserClaims中获取过期时间（expiresAt）
		expireTime, err := uc.GetExpirationTime()
		if err != nil {
			// 如果无法获取过期时间，返回401 Unauthorized
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 如果token已经过期，返回401 Unauthorized
		if expireTime.Before(time.Now()) {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 每 10 秒刷新一次
		//if expireTime.Sub(time.Now()) < time.Second*50 {
		//	uc.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
		//	newToken, err := token.SignedString(web.JWTKey)
		//	if err != nil {
		//		// 因为刷新这个事情，并不是一定要做的，所以这里可以考虑打印日志
		//		// 暂时这样打印
		//		log.Println(err)
		//	} else {
		//		ctx.Header("x-jwt-token", newToken)
		//	}
		//}

		// 如果token有效且未过期，则将token中的用户信息（claims）存放到Gin的上下文中
		// 这样后续的请求可以通过ctx.Get("user")来获取用户信息，避免重复解析token
		ctx.Set("user", uc)
	}
}
