package web

import (
	"errors"
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
	"time"
	"webook/internal/domain"
	"webook/internal/service"
	ijwt "webook/internal/web/jwt"
)

const (
	// 用于邮箱格式验证的正则表达式
	emailRegexPattern = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"

	// 用于密码格式验证的正则表达式
	passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`

	// 用于验证码登录
	bizLogin = "login"
)

//  var _ handler = &UserHandler{}

// UserHandler 结构体，用于处理用户相关的HTTP请求
type UserHandler struct {
	svc              service.UserService // 引用service层的UserService，处理具体的业务逻辑
	codeSvc          service.CodeService // 引用service层的CodeService，处理短信服务
	emailRegexExp    *regexp.Regexp      // 用于邮箱格式验证的正则表达式对象
	passwordRegexExp *regexp.Regexp      // 用于密码格式验证的正则表达式对象

	ijwt.Handler // 用于 JWT 鉴权登录
}

// NewUserHandler 构造函数，创建并返回一个新的UserHandler实例
// 接收一个service.UserService对象，用于处理注册、登录等请求
func NewUserHandler(svc service.UserService, codeSvc service.CodeService, jwthdl ijwt.Handler) *UserHandler {
	return &UserHandler{
		svc:              svc,
		codeSvc:          codeSvc,
		emailRegexExp:    regexp.MustCompile(emailRegexPattern, regexp.None),    // 编译邮箱格式正则
		passwordRegexExp: regexp.MustCompile(passwordRegexPattern, regexp.None), // 编译密码格式正则
		Handler:          jwthdl,
	}
}

// RegisterRoutes 方法用于注册用户相关的路由
func (c *UserHandler) RegisterRoutes(server *gin.Engine) {
	// 定义/users相关的路由组
	ug := server.Group("/users")
	ug.POST("/signup", c.SignUp) // 用户注册
	ug.POST("/login", c.Login)   // 用户登录
	ug.POST("/logout", c.Logout)
	ug.POST("/edit", c.Edit)      // 用户信息编辑
	ug.GET("/profile", c.Profile) // 获取用户信息

	ug.POST("/login_sms/code/send", c.SendSMSLoginCode)
	ug.POST("/login_sms", c.LoginSMS)
	ug.POST("/refresh_token", c.RefreshToken)
}

func (c *UserHandler) RefreshToken(ctx *gin.Context) {
	// 假定长 token 也放在这里
	tokenStr := c.ExtractTokenString(ctx)
	var rc ijwt.RefreshClaims
	token, err := jwt.ParseWithClaims(tokenStr, &rc, func(token *jwt.Token) (interface{}, error) {
		return ijwt.RefreshTokenKey, nil
	})
	// 这边要保持和登录校验一直的逻辑，即返回 401 响应
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, Result{Code: 4, Msg: "请登录"})
		return
	}
	if token == nil || !token.Valid {
		ctx.JSON(http.StatusUnauthorized, Result{Code: 4, Msg: "请登录"})
		return
	}

	err = c.CheckSession(ctx, rc.Ssid)
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = c.SetJWTToken(ctx, rc.Ssid, rc.Id)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, Result{Code: 4, Msg: "请登录"})
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "刷新成功"})
}

// LoginSMS 用户通过短信验证码进行登录
func (c *UserHandler) LoginSMS(ctx *gin.Context) {
	type Req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}
	ok, err := c.codeSvc.Verify(ctx, bizLogin, req.Phone, req.Code)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Code: 5, Msg: "系统异常"})
		return
	}
	if !ok {
		ctx.JSON(http.StatusOK, Result{Code: 4, Msg: "验证码错误"})
		return
	}

	// 验证码是对的
	// 登录或者注册用户
	u, err := c.svc.FindOrCreate(ctx, req.Phone)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Code: 4, Msg: "系统错误"})
		return
	}
	// 用 uuid 来标识这一次会话
	ssid := uuid.New().String()
	err = c.SetJWTToken(ctx, ssid, u.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "登录成功"})
}

// SendSMSLoginCode 发送短信验证码
func (c *UserHandler) SendSMSLoginCode(ctx *gin.Context) {
	type Req struct {
		Phone string `json:"phone"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}
	// 你也可以用正则表达式校验是不是合法的手机号
	if req.Phone == "" {
		ctx.JSON(http.StatusOK, Result{Code: 4, Msg: "请输入手机号码"})
		return
	}
	err := c.codeSvc.Send(ctx, bizLogin, req.Phone)
	switch {
	case err == nil:
		ctx.JSON(http.StatusOK, Result{Msg: "发送成功"})
	case errors.Is(err, service.ErrCodeSendTooMany):
		ctx.JSON(http.StatusOK, Result{Code: 4, Msg: "短信发送太频繁，请稍后再试"})
	default:
		ctx.JSON(http.StatusOK, Result{Code: 5, Msg: "系统错误"})
		// 要打印日志
		return
	}
}

// SignUp 用户注册接口
// 处理用户提交的注册信息，并进行验证，最后调用service层完成注册操作
func (c *UserHandler) SignUp(ctx *gin.Context) {
	// 定义一个SignUpReq结构体，用于接收请求中的用户注册信息
	type SignUpReq struct {
		Email           string `json:"email"`           // 用户邮箱
		Password        string `json:"password"`        // 用户密码
		ConfirmPassword string `json:"confirmPassword"` // 用户确认密码
	}

	// 绑定请求体中的数据到SignUpReq结构体
	var req SignUpReq
	if err := ctx.Bind(&req); err != nil {
		// 如果绑定失败，返回错误
		return
	}

	// 验证邮箱格式是否符合正则规则
	isEmail, err := c.emailRegexExp.MatchString(req.Email)
	if err != nil {
		// 如果匹配失败，返回系统错误
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !isEmail {
		// 如果邮箱格式不正确，返回错误信息
		ctx.String(http.StatusOK, "邮箱不正确")
		return
	}

	// 验证两次密码是否一致
	if req.Password != req.ConfirmPassword {
		// 如果两次密码不一致，返回错误信息
		ctx.String(http.StatusOK, "两次输入的密码不相同")
		return
	}

	// 验证密码格式是否符合要求
	isPassword, err := c.passwordRegexExp.MatchString(req.Password)
	if err != nil {
		// 如果匹配失败，返回系统错误
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !isPassword {
		// 如果密码不符合要求（至少8位，包含字母、数字和特殊字符），返回错误信息
		ctx.String(http.StatusOK,
			"密码必须包含数字、特殊字符，并且长度不能小于 8 位")
		return
	}

	// 调用service层的Signup方法，传入用户邮箱和密码
	err = c.svc.Signup(ctx.Request.Context(),
		domain.User{Email: req.Email, Password: req.ConfirmPassword})

	// 如果遇到重复邮箱的错误，返回相应提示
	if errors.Is(err, service.ErrUserDuplicateEmail) {
		ctx.String(http.StatusOK, "重复邮箱，请换一个邮箱")
		return
	}
	if err != nil {
		// 如果发生其他错误，返回服务器异常提示
		ctx.String(http.StatusOK, "服务器异常，注册失败")
		return
	}
	// 如果注册成功，返回成功信息
	ctx.String(http.StatusOK, "hello, 注册成功")
}

// Login 用户登录接口，使用的是 JWT
// 用户通过提供邮箱和密码进行登录，成功后生成一个JWT令牌返回给用户
// JWT令牌会存储在响应头 "x-jwt-token" 中，供前端存储和后续认证使用
func (c *UserHandler) Login(ctx *gin.Context) {
	// 定义请求体结构体，用于接收用户提交的邮箱和密码
	type LoginReq struct {
		Email    string `json:"email"`    // 用户的邮箱
		Password string `json:"password"` // 用户的密码
	}

	var req LoginReq
	// 绑定请求数据到结构体
	// 如果绑定失败（例如字段缺失或格式错误），则直接返回
	if err := ctx.Bind(&req); err != nil {
		return
	}

	// 调用服务层的Login方法进行用户身份验证
	// 如果邮箱和密码匹配成功，返回用户信息；如果验证失败，返回错误
	u, err := c.svc.Login(ctx.Request.Context(), req.Email, req.Password)
	if errors.Is(err, service.ErrInvalidUserOrPassword) {
		// 如果用户名或密码不正确，返回提示信息
		ctx.String(http.StatusOK, "用户名或者密码不正确，请重试")
		return
	}

	err = c.SetLoginToken(ctx, u.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		return
	}
	// 返回登录成功的响应
	ctx.String(http.StatusOK, "登录成功")
}

func (c *UserHandler) Logout(ctx *gin.Context) {
	err := c.ClearToken(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Msg: "系统错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "OK",
	})
}

// Edit 用户编译信息
func (c *UserHandler) Edit(ctx *gin.Context) {
	type Req struct {
		Nickname string `json:"nickname"`
		Birthday string `json:"birthday"`
		AboutMe  string `json:"aboutMe"`
	}

	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}
	if req.Nickname == "" {
		ctx.JSON(http.StatusOK, Result{Code: 4, Msg: "昵称不能为空"})
		return
	}

	if len(req.AboutMe) > 1024 {
		ctx.JSON(http.StatusOK, Result{Code: 4, Msg: "AboutMe过长"})
		return
	}
	birthday, err := time.Parse(time.DateOnly, req.Birthday)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Code: 4, Msg: "日期格式不对"})
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	err = c.svc.UpdateNonSensitiveInfo(ctx, domain.User{
		Id:       uc.Id,
		Nickname: req.Nickname,
		AboutMe:  req.AboutMe,
		Birthday: birthday,
	})
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "OK"})
}

// Profile 用户查看个人信息接口
// 该接口通过JWT令牌中的用户ID来查询当前用户的详细信息
func (c *UserHandler) Profile(ctx *gin.Context) {
	// 定义响应结构体，用于返回用户的邮箱信息
	type Profile struct {
		Email    string // 用户的邮箱
		Phone    string
		Nickname string
		Birthday string
		AboutMe  string
	}

	// 从上下文中获取JWT中的用户信息（UserClaims），通过ctx.MustGet("user")来获取
	// 该操作会返回UserClaims对象，其中包含用户的ID
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	// 调用服务层的Profile方法查询用户的详细信息
	u, err := c.svc.Profile(ctx, uc.Id)
	if err != nil {
		// 如果查询出错，可能是系统问题，返回系统错误信息
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 返回用户的邮箱信息，响应的格式是JSON
	ctx.JSON(http.StatusOK, Profile{
		Email:    u.Email, // 返回用户的邮箱
		Phone:    u.Phone,
		Nickname: u.Nickname,
		Birthday: u.Birthday.Format(time.DateOnly),
		AboutMe:  u.AboutMe,
	})
}
