package web

import (
	"github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"net/http"
	"webook/internal/domain"
	"webook/internal/service"
	ijwt "webook/internal/web/jwt"
)

type UserHandler struct {
	svc              service.UserService
	codeSvc          service.CodeService
	emailRegexExp    *regexp2.Regexp
	passwordRegexExp *regexp2.Regexp

	cmd redis.Cmdable
	ijwt.Handler
}

const (
	emailRegexPattern    = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
	passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
	bizLogin             = "login"
)

func NewUserHandler(svc service.UserService, smsSvc service.CodeService, jwtHdl ijwt.Handler) *UserHandler {

	return &UserHandler{
		svc:              svc,
		codeSvc:          smsSvc,
		emailRegexExp:    regexp2.MustCompile(emailRegexPattern, regexp2.None),
		passwordRegexExp: regexp2.MustCompile(passwordRegexPattern, regexp2.None),
		Handler:          jwtHdl,
	}
}

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {

	g := server.Group("/users")

	g.POST("/signup", u.SignUp)
	//server.POST("/users/login", u.Login)
	g.POST("/login", u.LoginJWT)
	g.POST("/edit", u.Edit)
	//server.GET("/users/profile", u.Profile)
	g.GET("/profile", u.ProfileJWT)

	g.POST("/login_sms/code/send", u.SendSMSLoginCode)
	g.POST("/login_sms", u.LoginSMS)

	g.POST("/refresh_token", u.RefreshToken)
	g.POST("/logout_jwt", u.LogoutJWT)
}

func (u *UserHandler) LogoutJWT(ctx *gin.Context) {

	err := u.ClearToken(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "退出登录失败",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "退出登录成功",
	})
}

func (u *UserHandler) RefreshToken(ctx *gin.Context) {

	refreshToken := u.ExtractToken(ctx)
	rclaims := &ijwt.RefreshClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, rclaims, func(token *jwt.Token) (interface{}, error) {
		return ijwt.RtKey, nil
	})
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if token == nil || !token.Valid {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = u.CheckSession(ctx, rclaims.Ssid)
	if err != nil {
		// 系统错误或者用户主动退出
		// 这里可以服务降级 就是redis崩溃的时候直接return 不去给redis压力
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if err := u.SetJWTToken(ctx, rclaims.Uid, rclaims.Ssid); err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "ok",
	})
}

func (u *UserHandler) SignUp(ctx *gin.Context) {
	type SignReq struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	var req SignReq
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	ok, err := u.emailRegexExp.MatchString(req.Email)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "邮箱格式不正确")
		return
	}

	if req.Password != req.ConfirmPassword {
		ctx.String(http.StatusOK, "两次输入的密码不相同")
		return
	}
	ok, err = u.passwordRegexExp.MatchString(req.Password)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "密码必须包含数字、特殊字符，并且长度不能小于8位")
		return
	}
	err = u.svc.SignUp(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	})
	if err == service.ErrUserDuplicateEmail {
		ctx.String(http.StatusOK, "邮箱冲突")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统异常")
		return
	}
	ctx.String(http.StatusOK, "注册成功")
}

func (u *UserHandler) LoginJWT(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	user, err := u.svc.Login(ctx, req.Email, req.Password)
	if err == service.ErrInvalidUserOrPassword {
		ctx.String(http.StatusOK, "用户名或密码不对")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 登录成功之后设置JWT
	if err := u.SetLoginToken(ctx, user.Id); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "登录成功")
}

func (u *UserHandler) Login(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	user, err := u.svc.Login(ctx, req.Email, req.Password)
	if err == service.ErrInvalidUserOrPassword {
		ctx.String(http.StatusOK, "用户名或密码不对")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 登录成功之后设置session的值
	sess := sessions.Default(ctx)
	sess.Set("userId", user.Id)
	sess.Options(sessions.Options{
		//Secure: true,
		//HttpOnly: true,
		MaxAge: 20,
	})
	sess.Save()

	ctx.String(http.StatusOK, "登录成功")
}

func (u *UserHandler) Edit(ctx *gin.Context) {
	type Req struct {
		Nickname        string `json:"nickname"`
		BirthDay        string `json:"birthDay"`
		PersonalProfile string `json:"personalProfile"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	c, ok := ctx.Get("claims")
	if !ok {
		// 正常情况不会没有
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	claims, ok := c.(*ijwt.UserClaims)
	if !ok {
		// 不ok代表断言结果不是 *UserClaims
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	err := u.svc.Edit(ctx, domain.User{
		Id:              claims.Uid,
		Nickname:        req.Nickname,
		BirthDay:        req.BirthDay,
		PersonalProfile: req.PersonalProfile,
	})
	if err != nil {
		ctx.String(http.StatusOK, "更新失败")
		return
	}
	ctx.String(http.StatusOK, "更新成功")
}

func (u *UserHandler) Profile(ctx *gin.Context) {
	// session实现
	sess := sessions.Default(ctx)
	userId, _ := sess.Get("userId").(int64)
	user, err := u.svc.GetProfile(ctx, userId)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.JSON(http.StatusOK, user)
}

func (u *UserHandler) ProfileJWT(ctx *gin.Context) {
	// JWT实现
	c, ok := ctx.Get("claims")
	if !ok {
		// 正常情况不会没有
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	claims, ok := c.(*ijwt.UserClaims)
	if !ok {
		// 不ok代表断言结果不是 *UserClaims
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	user, err := u.svc.GetProfile(ctx, claims.Uid)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.JSON(http.StatusOK, user)
}

func (u *UserHandler) SendSMSLoginCode(ctx *gin.Context) {
	type SMSReq struct {
		Phone string `json:"phone"`
	}
	var req SMSReq
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	err := u.codeSvc.Send(ctx, bizLogin, req.Phone)
	switch err {
	case nil:
		ctx.String(http.StatusOK, "发送成功")
	case service.ErrCodeSendTooMany:
		ctx.String(http.StatusOK, "发送验证码太频繁,请稍后再试")
	default:
		ctx.String(http.StatusOK, "系统错误")
	}
}

func (u *UserHandler) LoginSMS(ctx *gin.Context) {
	type LoginSMSReq struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	var req LoginSMSReq
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	ok, err := u.codeSvc.Verify(ctx, bizLogin, req.Phone, req.Code)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "验证码错误")
		return
	}

	user, err := u.svc.FindOrCreate(ctx, req.Phone)

	// 登录成功之后设置JWT
	if err := u.SetLoginToken(ctx, user.Id); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	ctx.String(http.StatusOK, "登录成功")
}
