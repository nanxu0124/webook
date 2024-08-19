package failover

import (
	"context"
	"sync/atomic"
	"webook/internal/service/sms"
)

// TimeoutFailoverSMSService 基于超时响应的判断
// 大多数情况服务商都没有问题
// 当服务商出现问题时，cnt数量会一下子增加
// 当cnt超过threshold的时候就切换服务商，并将cnt置0
type TimeoutFailoverSMSService struct {
	svcSms []sms.Service
	idx    int32

	// 连续超时的个数
	cnt int32
	// 阈值 连续超时超过这个数字就要切换
	threshold int32
}

func NewTimeoutFailoverSMSService(svcSms []sms.Service) sms.Service {
	return &TimeoutFailoverSMSService{
		svcSms: svcSms,
	}
}

func (t *TimeoutFailoverSMSService) Send(ctx context.Context, tpl string, param []string, numbers []string) error {
	idx := atomic.LoadInt32(&t.idx)
	cnt := atomic.LoadInt32(&t.cnt)

	if cnt > t.threshold {
		// 超过阈值 切换服务商
		newIdx := (idx + 1) % int32(len(t.svcSms))
		if atomic.CompareAndSwapInt32(&t.idx, idx, newIdx) {
			// 下标切换成功
			atomic.StoreInt32(&t.cnt, 0)
		}
		// else 就是出现并发，别人换成功了
		idx = atomic.LoadInt32(&t.idx)
	}
	svc := t.svcSms[idx]
	err := svc.Send(ctx, tpl, param, numbers)
	switch err {
	case context.DeadlineExceeded:
		// 超时，超时的标记加1
		atomic.AddInt32(&t.cnt, 1)
		return err
	case nil:
		// 连续状态被打断，cnt置0
		atomic.StoreInt32(&t.cnt, 0)
		return nil
	default:
		return err
	}
}
