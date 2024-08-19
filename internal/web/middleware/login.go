package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type LoginMiddlewareBuilder struct {
	paths []string
}

func NewLoginMiddlewareBuilder() *LoginMiddlewareBuilder {
	return &LoginMiddlewareBuilder{}
}

func (l *LoginMiddlewareBuilder) IgnorePaths(path string) *LoginMiddlewareBuilder {
	l.paths = append(l.paths, path)
	return l
}
func (l *LoginMiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 不需要登录校验的
		for _, path := range l.paths {
			if ctx.Request.URL.Path == path {
				return
			}
		}

		// 用session来校验
		sess := sessions.Default(ctx)
		if sess == nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		id := sess.Get("userId")
		if id == nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		now := time.Now().UnixMilli()
		UpdateTime := sess.Get("Update_time")
		if UpdateTime == nil { // nil表示还没刷新过 刚登录 还没刷新过
			sess.Set("Update_time", now)
			sess.Set("userId", id)
			sess.Options(sessions.Options{
				MaxAge: 20,
			})
			if err := sess.Save(); err != nil {
				panic(err)
			}
		}
		UpdateTimeVal, ok := UpdateTime.(int64)
		if !ok {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if now-UpdateTimeVal > 10*1000 {
			sess.Set("Update_time", now)
			sess.Set("userId", id)
			sess.Options(sessions.Options{
				MaxAge: 20,
			})
			if err := sess.Save(); err != nil {
				panic(err)
			}
		}
	}
}
