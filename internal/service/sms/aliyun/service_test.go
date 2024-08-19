package aliyun

import (
	"context"
	"fmt"
	openai "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewService(t *testing.T) {

	c, err := dysmsapi.NewClient(&openai.Config{
		AccessKeyId:     tea.String("****"),
		AccessKeySecret: tea.String("****"),
	})
	if err != nil {
		fmt.Println(err)
	}

	s := NewService(c, "阿里云短信测试")

	testCases := []struct {
		name    string
		tplId   string
		params  []string
		numbers []string
		wantErr error
	}{
		{
			name:   "发送验证码",
			tplId:  "SMS_154950909",
			params: []string{"{\"code\":\"666666\"}"},
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

//func Test_test(t *testing.T) {
//	val := []string{"123"}
//	if len(val) > 1000 {
//		fmt.Println("切片长度超过1000，无法拼接")
//		return
//	}
//	//print(val[0])
//	result := strings.Join(val, ",")
//	fmt.Println(result)
//}
