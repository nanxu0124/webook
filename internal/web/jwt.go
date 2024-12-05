package web

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

// UserClaims 是一个自定义的结构体，表示JWT中包含的用户信息
// 它继承了 jwt.RegisteredClaims 结构体，用于存储JWT的标准字段（如过期时间、发行者等）
// 另外我们添加了一个 Id 字段，用于存储用户的唯一标识符（如用户ID）
type UserClaims struct {
	// 用户的唯一ID
	Id int64

	// 用户的 UserAgent（通常是浏览器或客户端的标识）
	// 这个字段可以帮助记录发起请求的客户端类型或来源设备，通常用于日志分析或安全审计
	UserAgent string

	// jwt.RegisteredClaims 是一个结构体，包含JWT的标准字段
	// 比如：过期时间（ExpiresAt）、发行者（Issuer）、受众（Audience）等
	// 通过嵌入 RegisteredClaims，我们可以直接访问这些标准字段
	// 例如：UserClaims.ExpiresAt 直接就能访问过期时间
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	Uid int64
	jwt.RegisteredClaims
}

// AccessTokenKey 是用来签署和验证JWT的密钥
// 这个密钥通常不应该硬编码在代码中，实际应用中可以考虑将其存储在环境变量或配置文件中
// 但是在这个示例中，为了简便起见，我们将其写成了常量
var AccessTokenKey = []byte("moyn8y9abnd7q4zkq2m73yw8tu9j5ixm")
var refreshTokenKey = []byte("moyn8y9abnd7q4zkq2m73yw8tu9j5ixA")

type jwtHandler struct {
}

func (h *jwtHandler) setLoginToken(ctx *gin.Context, uid int64) error {
	err := h.setJWTToken(ctx, uid)
	if err != nil {
		return err
	}
	err = h.setRefreshToken(ctx, uid)
	if err != nil {
		return err
	}
	return nil
}
func (h *jwtHandler) setJWTToken(ctx *gin.Context, uid int64) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{
		Id:        uid,                         // 用户 ID
		UserAgent: ctx.GetHeader("User-Agent"), // 从请求头中获取 User-Agent
		RegisteredClaims: jwt.RegisteredClaims{
			//ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)), // 设置过期时间为 30 分钟
		},
	})

	// 使用 AccessTokenKey 对 token 进行签名
	tokenStr, err := token.SignedString(AccessTokenKey)
	if err != nil {
		// 如果签名过程中出错，返回系统异常信息
		return err
	}
	// 将生成的 token 添加到响应头部，使用 x-jwt-token 作为 header 名
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

func (h *jwtHandler) setRefreshToken(ctx *gin.Context, uid int64) error {
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, RefreshClaims{
		Uid: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			// 设置为七天过期
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
	})

	refreshTokenStr, err := refreshToken.SignedString(refreshTokenKey)
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", refreshTokenStr)
	return nil
}

func ExtractToken(ctx *gin.Context) string {
	// 从请求头中获取Authorization字段，格式应该是 "Bearer token"
	authCode := ctx.GetHeader("Authorization")
	if authCode == "" {
		return ""
	}

	// 使用空格将Authorization字段分割为两部分，第一部分为"Bearer"，第二部分为token字符串。
	// 通过 SplitN 将字符串分割为最多两部分。
	authSegments := strings.SplitN(authCode, " ", 2)
	if len(authSegments) != 2 {
		return ""
	}

	// 获取token字符串（即 "Bearer" 后面的部分）
	tokenStr := authSegments[1]
	return tokenStr
}
