package tencent

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"testing"
)

func TestService_Send(t *testing.T) {
	//secretId, ok := os.LookupEnv("SMS_SECRET_ID")
	//if !ok {
	//	t.Fatal()
	//}
	//secretKey, ok := os.LookupEnv("SMS_SECRET_KEY")

	c, err := sms.NewClient(common.NewCredential("***", "***"),
		"ap-guangzhou",
		profile.NewClientProfile())
	if err != nil {
		t.Fatal(err)
	}

	s := NewService(c, "1400881894", "南絮0124公众号")

	testCases := []struct {
		name    string
		tplId   string
		params  []string
		numbers []string
		wantErr error
	}{
		{
			name:   "发送验证码",
			tplId:  "2044585",
			params: []string{"666888"},
			// 改成你的手机号码
			numbers: []string{"151****", "197****"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			er := s.Send(context.Background(), tc.tplId, tc.params, tc.numbers)
			assert.Equal(t, tc.wantErr, er)
		})
	}
}
