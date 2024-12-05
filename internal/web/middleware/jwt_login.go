package middleware

import (
	"github.com/ecodeclub/ekit/set"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
	"webook/internal/web"
)

// JWTLoginMiddlewareBuilder 是一个中间件构建器，用于验证用户请求中的JWT令牌。
type JWTLoginMiddlewareBuilder struct {
	publicPaths set.Set[string]
}

func NewJWTLoginMiddlewareBuilder() *JWTLoginMiddlewareBuilder {
	s := set.NewMapSet[string](5)
	// 如果请求的路径是用户注册（/users/signup）或登录（/users/login）
	// 这些接口不需要JWT验证，直接放行
	s.Add("/users/signup")
	s.Add("/users/login_sms/code/send")
	s.Add("/users/login_sms")
	s.Add("/users/login")
	s.Add("/users/refresh_token")
	return &JWTLoginMiddlewareBuilder{
		publicPaths: s,
	}
}

// Build 方法创建并返回一个Gin的中间件，负责JWT的验证。
func (j *JWTLoginMiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 不需要校验
		if j.publicPaths.Exist(ctx.Request.URL.Path) {
			return
		}

		tokenStr := web.ExtractToken(ctx)

		// 创建UserClaims结构体用于解析token中的claim信息
		uc := web.UserClaims{}

		// 使用jwt.ParseWithClaims解析token并验证其合法性
		// tokenStr是待验证的JWT字符串，uc是用于存放解析后的claims信息
		// web.JWTKey是密钥，用于验证token的签名
		token, err := jwt.ParseWithClaims(tokenStr, &uc, func(token *jwt.Token) (interface{}, error) {
			return web.AccessTokenKey, nil
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

		if ctx.GetHeader("User-Agent") != uc.UserAgent {
			// 换了一个 User-Agent，可能是攻击者
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		//// 刷新 token
		//if expireTime.Sub(time.Now()) < time.Second*50 {
		//	// 如果距离过期时间小于 50 秒，进行 token 刷新
		//	uc.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute)) // 设置新的过期时间为当前时间加一分钟
		//	newToken, err := token.SignedString(web.AccessTokenKey)                // 使用原来的 JWT 密钥对新的 claims 生成新的 token
		//	if err != nil {
		//		// 如果生成新的 token 出错，则打印错误日志
		//		log.Println(err)
		//	} else {
		//		// 如果生成新 token 成功，则将新的 token 添加到响应头中
		//		// 这里的 "x-jwt-token" 是自定义的 HTTP 响应头，客户端可以使用该 header 获取新的 token
		//		ctx.Header("x-jwt-token", newToken)
		//	}
		//}

		// 如果token有效且未过期，则将token中的用户信息（claims）存放到Gin的上下文中
		// 这样后续的请求可以通过ctx.Get("user")来获取用户信息，避免重复解析token
		ctx.Set("user", uc)
	}
}
