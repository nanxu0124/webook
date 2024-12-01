package web

import (
	"errors"
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
	"webook/internal/domain"
	"webook/internal/service"
)

const (
	// 用于邮箱格式验证的正则表达式
	emailRegexPattern = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"

	// 用于密码格式验证的正则表达式
	passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
)

// UserHandler 结构体，用于处理用户相关的HTTP请求
type UserHandler struct {
	svc              *service.UserService // 引用service层的UserService，处理具体的业务逻辑
	emailRegexExp    *regexp.Regexp       // 用于邮箱格式验证的正则表达式对象
	passwordRegexExp *regexp.Regexp       // 用于密码格式验证的正则表达式对象

	jwtKey string // 用于 JWT 鉴权登录
}

// NewUserHandler 构造函数，创建并返回一个新的UserHandler实例
// 接收一个service.UserService对象，用于处理注册、登录等请求
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{
		svc:              svc,
		emailRegexExp:    regexp.MustCompile(emailRegexPattern, regexp.None),    // 编译邮箱格式正则
		passwordRegexExp: regexp.MustCompile(passwordRegexPattern, regexp.None), // 编译密码格式正则
	}
}

// RegisterRoutes 方法用于注册用户相关的路由
func (c *UserHandler) RegisterRoutes(server *gin.Engine) {
	// 定义/users相关的路由组
	ug := server.Group("/users")
	ug.POST("/signup", c.SignUp)  // 用户注册
	ug.POST("/login", c.Login)    // 用户登录
	ug.POST("/edit", c.Edit)      // 用户信息编辑
	ug.GET("/profile", c.Profile) // 获取用户信息
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

	// 登录成功，创建一个新的JWT令牌
	// UserClaims中包含用户ID以及过期时间
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{
		Id:        u.Id, // 用户ID
		UserAgent: ctx.GetHeader("User-Agent"),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 10)), // 设置JWT过期时间
		},
	})

	// 使用密钥对生成的JWT令牌进行签名
	tokenStr, err := token.SignedString(JWTKey)
	if err != nil {
		// 如果签名失败，返回系统异常
		ctx.String(http.StatusOK, "系统异常")
		return
	}

	// 将生成的JWT令牌返回给前端，存储在响应头 x-jwt-token 中
	ctx.Header("x-jwt-token", tokenStr)
	// 返回登录成功的响应
	ctx.String(http.StatusOK, "登录成功")
}

// Edit 用户编辑个人信息的接口（此接口目前未实现）
func (c *UserHandler) Edit(ctx *gin.Context) {
	// 该方法为空，表示目前未实现用户编辑个人信息的功能
}

// Profile 用户查看个人信息接口
// 该接口通过JWT令牌中的用户ID来查询当前用户的详细信息
func (c *UserHandler) Profile(ctx *gin.Context) {
	// 定义响应结构体，用于返回用户的邮箱信息
	type Profile struct {
		Email string // 用户的邮箱
	}

	// 从上下文中获取JWT中的用户信息（UserClaims），通过ctx.MustGet("user")来获取
	// 该操作会返回UserClaims对象，其中包含用户的ID
	uc := ctx.MustGet("user").(UserClaims)

	// 调用服务层的Profile方法查询用户的详细信息
	u, err := c.svc.Profile(ctx, uc.Id)
	if err != nil {
		// 如果查询出错，可能是系统问题，返回系统错误信息
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 返回用户的邮箱信息，响应的格式是JSON
	ctx.JSON(http.StatusOK, Profile{
		Email: u.Email, // 返回用户的邮箱
	})
}
