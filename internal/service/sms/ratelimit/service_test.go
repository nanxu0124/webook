package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"webook/internal/service/sms"
	smsmocks "webook/internal/service/sms/mocks"
	"webook/pkg/ratelimit"
	limitermocks "webook/pkg/ratelimit/mocks"
)

func TestRatelimitSMSService_Send(t *testing.T) {
	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) (sms.Service, ratelimit.Limiter)

		wantErr error
	}{
		{
			name: "正常发送",
			mock: func(ctrl *gomock.Controller) (sms.Service, ratelimit.Limiter) {
				svc := smsmocks.NewMockService(ctrl)
				limiter := limitermocks.NewMockLimiter(ctrl)

				limiter.EXPECT().Limit(gomock.Any(), gomock.Any()).Return(false, nil)
				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				return svc, limiter
			},
			wantErr: nil,
		},
		{
			name: "触发限流",
			mock: func(ctrl *gomock.Controller) (sms.Service, ratelimit.Limiter) {
				svc := smsmocks.NewMockService(ctrl)
				limiter := limitermocks.NewMockLimiter(ctrl)

				limiter.EXPECT().Limit(gomock.Any(), gomock.Any()).Return(true, nil)

				return svc, limiter
			},
			wantErr: errors.New("触发了限流"),
		},
		{
			name: "限流器异常",
			mock: func(ctrl *gomock.Controller) (sms.Service, ratelimit.Limiter) {
				svc := smsmocks.NewMockService(ctrl)
				limiter := limitermocks.NewMockLimiter(ctrl)

				limiter.EXPECT().Limit(gomock.Any(), gomock.Any()).Return(true, errors.New("限流器异常"))

				return svc, limiter
			},
			wantErr: fmt.Errorf("短信服务判断是否限流出现问题，%w", errors.New("限流器异常")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc, limiter := tc.mock(ctrl)
			limitSvc := NewRatelimitSMSService(svc, limiter)
			err := limitSvc.Send(context.Background(), "testtpl", []string{"123456"}, []string{"151****"})
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
