package web

import (
	"errors"
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"net/http"
	"webook/internal/domain"
	"webook/internal/service"
)

// 用于邮箱格式验证的正则表达式
const emailRegexPattern = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"

// 用于密码格式验证的正则表达式
const passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`

// UserHandler 结构体，用于处理用户相关的HTTP请求
type UserHandler struct {
	svc              *service.UserService // 引用service层的UserService，处理具体的业务逻辑
	emailRegexExp    *regexp.Regexp       // 用于邮箱格式验证的正则表达式对象
	passwordRegexExp *regexp.Regexp       // 用于密码格式验证的正则表达式对象
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

// Login 用户登录接口（尚未实现）
func (c *UserHandler) Login(ctx *gin.Context) {
	// 这里可以实现用户登录的相关逻辑，例如验证用户名密码、生成JWT等
}

// Edit 用户编辑个人信息接口（尚未实现）
func (c *UserHandler) Edit(ctx *gin.Context) {
	// 这里可以实现用户编辑个人信息的逻辑
}

// Profile 获取用户个人信息接口（尚未实现）
func (c *UserHandler) Profile(ctx *gin.Context) {
	// 这里可以实现获取用户信息的逻辑
	ctx.JSON(http.StatusOK, "这是测试信息。")
}
