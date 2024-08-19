package failover

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"webook/internal/service/sms"
	"webook/internal/service/sms/ratelimit"
)

// FailoverSMSService 最简单的轮询，基本上第一个smsSvc就成功了，所以负载会很不均衡
type FailoverSMSService struct {
	smsSvc []sms.Service
}

func NewFailoverSMSService(smsSvc []sms.Service) sms.Service {
	return &FailoverSMSService{
		smsSvc: smsSvc,
	}
}

func (f *FailoverSMSService) Send(ctx context.Context, tpl string, param []string, numbers []string) error {
	for _, svc := range f.smsSvc {
		err := svc.Send(ctx, tpl, param, numbers)
		if err == nil {
			// 发送成功
			return nil
		}
		// 输出日志
		// 做好监控
		log.Println(err)
	}

	return errors.New("全部服务商都失败了")
}

// FailoverSMSServiceV2 第二种实现
// smsSvc是动态计算的
type FailoverSMSServiceV2 struct {
	smsSvc []sms.Service
	idx    uint64
}

func NewFailoverSMSServiceV2(smsSvc []sms.Service) sms.Service {
	return &FailoverSMSServiceV2{
		smsSvc: smsSvc,
	}
}

func (f *FailoverSMSServiceV2) Send(ctx context.Context, tpl string, param []string, numbers []string) error {
	idx := atomic.AddUint64(&f.idx, 1)
	length := uint64(len(f.smsSvc))

	for i := idx; i < idx+length; i++ {
		svc := f.smsSvc[i%length]
		err := svc.Send(ctx, tpl, param, numbers)
		switch err {
		case nil:
			return nil
		case context.DeadlineExceeded, context.Canceled:
			return err
		case ratelimit.ErrLimited:
			return err
		default:
			log.Println(err)
		}
	}
	return errors.New("全部服务商都失败了")
}
