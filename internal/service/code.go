package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"webook/internal/repository"
	"webook/internal/service/sms"
)

var ErrCodeSendTooMany = repository.ErrCodeSendTooMany

const codeTplId = "2320764" // 短信模板 ID，用于发送验证码短信

// CodeService 负责处理验证码的相关业务逻辑：生成验证码、存储验证码、发送短信以及验证验证码
type CodeService struct {
	sms  sms.Service                // 短信服务接口，用于发送验证码短信
	repo *repository.CodeRepository // 数据库操作对象，用于存储和验证验证码
}

// NewCodeService 创建并返回一个新的 CodeService 实例
func NewCodeService(svc sms.Service, repo *repository.CodeRepository) *CodeService {
	return &CodeService{
		sms:  svc,  // 初始化短信服务
		repo: repo, // 初始化验证码仓库
	}
}

// Send 生成一个随机验证码，并发送给指定的手机号
// biz 是业务场景，phone 是接收验证码的手机号
func (c *CodeService) Send(ctx context.Context, biz string, phone string) error {
	code := c.generate() // 生成一个随机验证码
	// 存储验证码到缓存中
	err := c.repo.Store(ctx, biz, phone, code)
	if err != nil {
		return err // 存储失败，返回错误
	}
	// 发送验证码短信
	err = c.sms.Send(ctx, codeTplId, []string{code}, phone)
	return err // 返回发送短信的错误
}

// Verify 验证输入的验证码是否正确
// biz 是业务场景，phone 是接收验证码的手机号，inputCode 是用户输入的验证码
func (c *CodeService) Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error) {
	// 调用 repository 层的 Verify 方法验证验证码
	ok, err := c.repo.Verify(ctx, biz, phone, inputCode)
	// 处理特殊的错误：验证码验证次数超限
	if errors.Is(err, repository.ErrCodeVerifyTooManyTimes) {
		// 如果验证次数超过限制，表示可能存在异常行为（例如恶意攻击）
		// 在接入告警系统后，可以在这里进行告警处理
		return false, nil // 返回 false，表示验证失败
	}
	// 返回验证码验证的结果
	return ok, err
}

// generate 生成一个随机的 6 位验证码
// 使用随机数生成一个介于 0 到 999999 之间的验证码
func (c *CodeService) generate() string {
	num := rand.Intn(999999) // 生成随机数
	// 将随机数格式化为 6 位字符串，前面补零
	return fmt.Sprintf("%06d", num)
}
