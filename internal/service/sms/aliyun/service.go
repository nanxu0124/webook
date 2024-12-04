package aliyun

import (
	"context"
	"fmt"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"strings"
)

type Service struct {
	client   *dysmsapi.Client
	signName string
}

func NewService(client *dysmsapi.Client, signName string) *Service {
	return &Service{
		client:   client,
		signName: signName,
	}
}

func (s *Service) Send(ctx context.Context, tpl string, param []string, numbers ...string) error {
	// ali云要求最多一次发1000个
	if len(numbers) > 1000 {
		return fmt.Errorf("phone numbers 超过1000")
	}
	// 多个号码格式要求为 133,137,150 用逗号分隔的字符串
	numberStr := strings.Join(numbers, ",")

	req := dysmsapi.SendSmsRequest{
		SignName:      tea.String(s.signName),
		PhoneNumbers:  tea.String(numberStr),
		TemplateCode:  tea.String(tpl),
		TemplateParam: tea.String(param[0]),
	}
	resp, err := s.client.SendSms(&req)
	if err != nil {
		return err
	}
	if *(resp.StatusCode) != 200 {
		return fmt.Errorf("发送短信失败,code:%s,%s", string(*(resp.StatusCode)), *(resp.Body.Message))
	}
	if *(resp.Body.Code) != "OK" {
		return fmt.Errorf("发送短信失败,code:%s,%s", *(resp.Body.Code), *(resp.Body.Message))
	}
	return nil
}
