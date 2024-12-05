package jwt

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

// AccessTokenKey 因为 JWT Key 不太可能变，所以可以直接写成常量
var AccessTokenKey = []byte("moyn8y9abnd7q4zkq2m73yw8tu9j5ixm")
var RefreshTokenKey = []byte("moyn8y9abnd7q4zkq2m73yw8tu9j5ixA")

type RedisHandler struct {
	cmd redis.Cmdable
	// 长 token 的过期时间
	rtExpiration time.Duration
}

func NewRedisHandler(cmd redis.Cmdable) Handler {
	return &RedisHandler{
		cmd:          cmd,
		rtExpiration: time.Hour * 24 * 7,
	}
}

func (h *RedisHandler) SetJWTToken(ctx *gin.Context, ssid string, uid int64) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{
		Id:        uid, // 用户 ID
		Ssid:      ssid,
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

// ClearToken 清除 token
func (h *RedisHandler) ClearToken(ctx *gin.Context) error {
	// 正常用户的这两个 token 都会被前端更新
	// 也就是说在登录校验里面，走不到 redis 那一步就返回了
	ctx.Header("x-jwt-token", "")
	ctx.Header("x-refresh-token", "")
	// 这里不可能拿不到
	uc := ctx.MustGet("user").(UserClaims)
	return h.cmd.Set(ctx, h.key(uc.Ssid), "", h.rtExpiration).Err()
}

func (h *RedisHandler) key(ssid string) string {
	return fmt.Sprintf("users:Ssid:%s", ssid)
}

// SetLoginToken 设置登录后的 token
func (h *RedisHandler) SetLoginToken(ctx *gin.Context, uid int64) error {
	ssid := uuid.New().String()
	err := h.SetJWTToken(ctx, ssid, uid)
	if err != nil {
		return err
	}
	err = h.setRefreshToken(ctx, ssid, uid)
	return err
}

func (h *RedisHandler) setRefreshToken(ctx *gin.Context, ssid string, uid int64) error {
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, RefreshClaims{
		Id: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			// 设置为七天过期
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
	})

	refreshTokenStr, err := refreshToken.SignedString(RefreshTokenKey)
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", refreshTokenStr)
	return nil
}

func (h *RedisHandler) CheckSession(ctx *gin.Context, ssid string) error {
	logout, err := h.cmd.Exists(ctx, fmt.Sprintf("users:Ssid:%s", ssid)).Result()
	if err != nil {
		// 系统错误或者用户已经主动退出登录了
		// TODO 可以考虑降级措施，如果在 Redis 已经崩溃的时候就不要去校验是不是已经主动退出登录了
		return err
	}
	if logout > 0 {
		return errors.New("用户已经退出登录")
	}
	return nil
}

func (h *RedisHandler) ExtractTokenString(ctx *gin.Context) string {
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
