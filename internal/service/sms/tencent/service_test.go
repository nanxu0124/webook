package tencent

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"os"
	"testing"
)

func TestService_Send(t *testing.T) {

	secretId, ok := os.LookupEnv("Tencent_SMS_Secret_Id")
	if !ok {
		panic("没有找到环境变量 Tencent_SMS_Secret_Id ")
	}
	secretKey, ok := os.LookupEnv("Tencent_SMS_Secret_Key")
	if !ok {
		panic("没有找到环境变量 Tencent_SMS_Secret_Key ")
	}

	c, err := sms.NewClient(common.NewCredential(secretId, secretKey),
		"ap-beijing",
		profile.NewClientProfile())
	if err != nil {
		t.Fatal(err)
	}

	s := NewService(c, "1400952398", "南絮0124公众号")

	testCases := []struct {
		name    string
		tplId   string
		params  []string
		numbers []string
		wantErr error
	}{
		{
			name:   "发送验证码",
			tplId:  "2320764",
			params: []string{"666777"},
			// 改成你的手机号码
			numbers: []string{"151***"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			er := s.Send(context.Background(), tc.tplId, tc.params, tc.numbers...)
			assert.Equal(t, tc.wantErr, er)
		})
	}
}
