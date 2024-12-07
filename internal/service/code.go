package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"webook/internal/repository"
	"webook/internal/service/sms"
	"webook/pkg/logger"
)

var (
	ErrCodeSendTooMany = repository.ErrCodeSendTooMany
)

const codeTplId = "2320764" // 短信模板 ID，用于发送验证码短信

// CodeService 是处理验证码相关业务逻辑的接口
// 提供了发送验证码和验证验证码的功能
type CodeService interface {

	// Send 用于发送验证码
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - biz: 验证码的业务类型，区分不同的业务场景，如登录、注册等
	//   - phone: 用户的手机号码，接收验证码的对象
	// 返回:
	//   - error: 如果发送成功返回 nil；如果发送失败（如验证码发送失败、发送频率过高等），则返回相应的错误
	Send(ctx context.Context, biz string, phone string) error

	// Verify 用于验证用户输入的验证码
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - biz: 验证码的业务类型，用于区分不同场景的验证码
	//   - phone: 用户的手机号码，验证码验证的对象
	//   - inputCode: 用户输入的验证码
	// 返回:
	//   - bool: 返回是否验证成功。如果验证码正确，返回 true；否则返回 false
	//   - error: 如果验证过程中发生错误，返回错误信息
	Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error)
}

// SMSCodeService 负责处理验证码的相关业务逻辑：生成验证码、存储验证码、发送短信以及验证验证码
type SMSCodeService struct {
	sms    sms.Service               // 短信服务接口，用于发送验证码短信
	repo   repository.CodeRepository // 数据库操作对象，用于存储和验证验证码
	logger logger.Logger
}

// NewSMSCodeService 实现 CodeService 接口
func NewSMSCodeService(svc sms.Service, repo repository.CodeRepository, l logger.Logger) CodeService {
	return &SMSCodeService{
		sms:    svc,
		repo:   repo,
		logger: l,
	}
}

func (c *SMSCodeService) Send(ctx context.Context, biz string, phone string) error {
	code := c.generate() // 生成一个随机验证码
	// 存储验证码到缓存中
	err := c.repo.Store(ctx, biz, phone, code)
	if err != nil {
		return err // 存储失败，返回错误
	}
	// 发送验证码短信
	err = c.sms.Send(ctx, codeTplId, []string{code}, phone)
	// TODO 这里考虑返回 err 之后是否要删除 redis 里边的验证码
	if err != nil {
		c.logger.Warn("发送验证码短信失败: ", logger.Field{
			Key:   "SMSCodeService",
			Value: err.Error(),
		})
	}
	return err // 返回发送短信的错误
}

func (c *SMSCodeService) Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error) {
	// 调用 repository 层的 Verify 方法验证验证码
	ok, err := c.repo.Verify(ctx, biz, phone, inputCode)
	// 处理特殊的错误：验证码验证次数超限
	if errors.Is(err, repository.ErrCodeVerifyTooManyTimes) {
		// 如果验证次数超过限制，表示可能存在异常行为（例如恶意攻击）
		// 在接入告警系统后，可以在这里进行告警处理
		c.logger.Error("验证次数超过限制: ", logger.Field{
			Key:   "SMSCodeService",
			Value: err.Error(),
		})
		return false, nil // 返回 false，表示验证失败
	}
	// 返回验证码验证的结果
	return ok, err
}

// generate 生成一个随机的 6 位验证码
// 使用随机数生成一个介于 0 到 999999 之间的验证码
func (c *SMSCodeService) generate() string {
	num := rand.Intn(999999) // 生成随机数
	// 将随机数格式化为 6 位字符串，前面补零
	return fmt.Sprintf("%06d", num)
}
