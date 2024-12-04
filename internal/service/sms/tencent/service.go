package tencent

import (
	"context"
	"fmt"
	"github.com/ecodeclub/ekit"
	"github.com/ecodeclub/ekit/slice"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

// Service 是实现了 sms.Service 接口的具体类型，用于调用腾讯云的短信服务
// 它封装了调用腾讯云 SMS API 的相关操作
type Service struct {
	client   *sms.Client // 腾讯云短信服务的客户端
	appId    *string     // 腾讯云短信应用的 ID
	signName *string     // 短信签名名称
}

// NewService 创建并返回一个新的 Service 实例
// client 是腾讯云短信服务的客户端，appId 和 signName 分别是短信应用 ID 和签名名称
func NewService(c *sms.Client, appId string,
	signName string) *Service {
	return &Service{
		client:   c,
		appId:    ekit.ToPtr[string](appId),
		signName: ekit.ToPtr[string](signName),
	}
}

// Send 实现了 sms.Service 接口的 Send 方法，调用腾讯云的 API 发送短信
// tplId 是短信模板 ID，args 是模板中占位符的参数，numbers 是目标手机号
func (s *Service) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	// 创建短信请求对象
	req := sms.NewSendSmsRequest()

	// 将手机号列表转换为指针切片
	req.PhoneNumberSet = toStringPtrSlice(numbers)

	// 设置短信应用 ID
	req.SmsSdkAppId = s.appId

	// 传递上下文
	req.SetContext(ctx)

	// 设置短信模板参数
	req.TemplateParamSet = toStringPtrSlice(args)

	// 设置短信模板 ID
	req.TemplateId = ekit.ToPtr[string](tplId)

	// 设置短信签名名称
	req.SignName = s.signName

	// 调用腾讯云短信服务的 API 发送短信
	resp, err := s.client.SendSms(req)
	if err != nil {
		return err // 如果发生错误，直接返回
	}

	// 检查发送状态，确保每一条短信都发送成功
	for _, status := range resp.Response.SendStatusSet {
		if status.Code == nil || *(status.Code) != "Ok" {
			// 如果有任何一条短信发送失败，返回错误信息
			return fmt.Errorf("发送失败，code: %s, 原因：%s", *status.Code, *status.Message)
		}
	}

	// 如果没有错误，返回 nil 表示发送成功
	return nil
}

// toStringPtrSlice 将字符串切片转换为字符串指针切片
// 用于将手机号列表和模板参数转换为腾讯云 API 所需的格式
func toStringPtrSlice(src []string) []*string {
	return slice.Map[string, *string](src, func(idx int, src string) *string {
		return &src
	})
}
