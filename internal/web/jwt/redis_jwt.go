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

var (
	AtKey = []byte("HjVq4YPaFoMywaPfKrhCbLIxpzGHA0Vf")
	RtKey = []byte("HjVq4YPaFoMywaPfKrhCbLIxpzGHAqVq")
)

type RedisJWTHandler struct {
	cmd redis.Cmdable
}

func NewRedisJWT(cmd redis.Cmdable) Handler {
	return &RedisJWTHandler{
		cmd: cmd,
	}
}

func (h *RedisJWTHandler) SetLoginToken(ctx *gin.Context, uid int64) error {
	ssid := uuid.New().String()
	err := h.SetJWTToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	err = h.SetRefreshToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	return nil
}

func (h *RedisJWTHandler) ClearToken(ctx *gin.Context) error {
	// 对于普通用户来说，可以直接把token换成一个不可用的值，前端直接覆盖掉, 这样就可以直接401从而不用走redis
	ctx.Header("x-jwt-token", "")
	ctx.Header("x-refresh-token", "")

	// 对于攻击者来说，需要到redis里边改ssid的状态
	c, ok := ctx.Get("claims")
	if !ok {
		return errors.New("系统错误")
	}
	claims, ok := c.(*UserClaims)
	if !ok {
		return errors.New("系统错误")
	}

	// 退出登录的时候才在redis里边写入一个值，只要 users:ssid:ssid 这个key存在就表明这个claim的用户退出登录了
	err := h.cmd.Set(ctx, fmt.Sprintf("users:ssid:%s", claims.Ssid), "", time.Minute*30).Err()
	if err != nil {
		return err
	}
	return nil
}

func (h *RedisJWTHandler) CheckSession(ctx *gin.Context, ssid string) error {
	// 要么redis有问题 要么已经退出登录
	// users:ssid:%s 这个key存在表明已经退出登录了
	cnt, err := h.cmd.Exists(ctx, fmt.Sprintf("users:ssid:%s", ssid)).Result()
	if cnt != 0 {
		return errors.New("已经退出登录")
	}
	return err
}

func (h *RedisJWTHandler) ExtractToken(ctx *gin.Context) string {
	tokenHeader := ctx.GetHeader("Authorization")
	segs := strings.Split(tokenHeader, " ")
	if len(segs) != 2 { // 正常都是2段
		return ""
	}
	return segs[1]
}

func (h *RedisJWTHandler) SetJWTToken(ctx *gin.Context, UId int64, ssid string) error {
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 1)), // 过期时间
		},
		Uid:       UId,
		Ssid:      ssid,
		UserAgent: ctx.Request.UserAgent(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString(AtKey)
	if err != nil {
		return err
	}
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

func (h *RedisJWTHandler) SetRefreshToken(ctx *gin.Context, UId int64, ssid string) error {
	claims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)), // 过期时间
		},
		Uid:  UId,
		Ssid: ssid,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString(RtKey)
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", tokenStr)
	return nil
}
