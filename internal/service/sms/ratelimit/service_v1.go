package ratelimit

import (
	"context"
	"fmt"
	"webook/internal/service/sms"
	"webook/pkg/ratelimit"
)

// RatelimitSMSServiceV1 组合实现sms.Serivce接口
// 优点是当sms.Service有很多功能，但是我只想装饰其中一个功能时候，用组合更好
// 缺点时这个Service的字段时大写的，别人可以绕过去
type RatelimitSMSServiceV1 struct {
	sms.Service
	limiter ratelimit.Limiter
}

func NewRatelimitSMSServiceV1(limiter ratelimit.Limiter) sms.Service {
	return &RatelimitSMSServiceV1{
		limiter: limiter,
	}
}

func (s *RatelimitSMSServiceV1) Send(ctx context.Context, tpl string, param []string, numbers []string) error {

	limited, err := s.limiter.Limit(ctx, "sms:message")
	if err != nil {
		return fmt.Errorf("短信服务判断是否限流出现问题，%w", err)
	}
	if limited {
		return ErrLimited
	}

	err = s.Service.Send(ctx, tpl, param, numbers)
	return err
}
