package failover

import (
	"context"
	"errors"
	"webook/internal/service/sms"
)

type FailoverSMSService struct {
	// 一大堆可供选择的 SMS Service 实现
	svcs []sms.Service

	idx uint64
}

func NewFailoverSMSService(svcs []sms.Service) *FailoverSMSService {
	return &FailoverSMSService{
		svcs: svcs,
	}
}

func (f *FailoverSMSService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	for _, svc := range f.svcs {
		err := svc.Send(ctx, tplId, args, numbers...)
		if err == nil {
			return nil
		}
		// TODO 考虑打日志
	}
	return errors.New("发送失败，所有服务商都尝试过了")
}
