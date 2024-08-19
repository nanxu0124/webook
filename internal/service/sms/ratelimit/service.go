package ratelimit

import (
	"context"
	"fmt"
	"webook/internal/service/sms"
	"webook/pkg/ratelimit"
)

// RatelimitSMSService 装饰器
// svc 是被装饰的东西
type RatelimitSMSService struct {
	svc     sms.Service
	limiter ratelimit.Limiter
}

var ErrLimited = fmt.Errorf("触发了限流")

func NewRatelimitSMSService(svc sms.Service, limiter ratelimit.Limiter) sms.Service {
	return &RatelimitSMSService{
		svc:     svc,
		limiter: limiter,
	}
}

func (s *RatelimitSMSService) Send(ctx context.Context, tpl string, param []string, numbers []string) error {

	limited, err := s.limiter.Limit(ctx, "sms:message")
	if err != nil {
		return fmt.Errorf("短信服务判断是否限流出现问题，%w", err)
	}
	if limited {
		return ErrLimited
	}

	err = s.svc.Send(ctx, tpl, param, numbers)
	return err
}
