package web

import (
	"bytes"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"
	"webook/internal/domain"
	"webook/internal/service"
	svcmocks "webook/internal/service/mocks"
)

func TestUserHandler_SignUp(t *testing.T) {
	testCases := []struct {
		name string

		mock func(ctrl *gomock.Controller) service.UserService

		reqbody string

		wantCode int
		wantBody string
	}{
		{
			name: "注册成功",
			mock: func(ctrl *gomock.Controller) service.UserService {
				usersvc := svcmocks.NewMockUserService(ctrl)
				usersvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "123@qq.com",
					Password: "hello#word123",
				}).Return(nil)
				return usersvc
			},
			reqbody: `
{
	"email":"123@qq.com",
	"password":"hello#word123",
	"confirmPassword":"hello#word123"
}
`,
			wantCode: http.StatusOK,
			wantBody: "注册成功",
		},
		{
			name: "参数不对，bind 失败",
			mock: func(ctrl *gomock.Controller) service.UserService {
				usersvc := svcmocks.NewMockUserService(ctrl)
				return usersvc
			},
			reqbody: `
{
	"email":"123@qq.com",
`,
			wantCode: http.StatusBadRequest,
			wantBody: "系统错误",
		},
		{
			name: "邮箱格式不对",
			mock: func(ctrl *gomock.Controller) service.UserService {
				usersvc := svcmocks.NewMockUserService(ctrl)
				return usersvc
			},
			reqbody: `
{
	"email":"123qq.com",
	"password":"hello#word123",
	"confirmPassword":"hello#word123"
}
`,
			wantCode: http.StatusOK,
			wantBody: "邮箱格式不正确",
		},
		{
			name: "两次输入密码不匹配",
			mock: func(ctrl *gomock.Controller) service.UserService {
				usersvc := svcmocks.NewMockUserService(ctrl)
				return usersvc
			},
			reqbody: `
{
	"email":"123@qq.com",
	"password":"hello#word1234",
	"confirmPassword":"hello#word123"
}
`,
			wantCode: http.StatusOK,
			wantBody: "两次输入的密码不相同",
		},
		{
			name: "密码格式不对",
			mock: func(ctrl *gomock.Controller) service.UserService {
				usersvc := svcmocks.NewMockUserService(ctrl)
				return usersvc
			},
			reqbody: `
{
	"email":"123@qq.com",
	"password":"helloword123",
	"confirmPassword":"helloword123"
}
`,
			wantCode: http.StatusOK,
			wantBody: "密码必须包含数字、特殊字符，并且长度不能小于8位",
		},
		{
			name: "邮箱冲突",
			mock: func(ctrl *gomock.Controller) service.UserService {
				usersvc := svcmocks.NewMockUserService(ctrl)
				usersvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "123@qq.com",
					Password: "hello#word123",
				}).Return(service.ErrUserDuplicateEmail)
				return usersvc
			},
			reqbody: `
{
	"email":"123@qq.com",
	"password":"hello#word123",
	"confirmPassword":"hello#word123"
}
`,
			wantCode: http.StatusOK,
			wantBody: "邮箱冲突",
		},
		{
			name: "系统异常",
			mock: func(ctrl *gomock.Controller) service.UserService {
				usersvc := svcmocks.NewMockUserService(ctrl)
				usersvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "123@qq.com",
					Password: "hello#word123",
				}).Return(errors.New("随便一个error"))
				return usersvc
			},
			reqbody: `
{
	"email":"123@qq.com",
	"password":"hello#word123",
	"confirmPassword":"hello#word123"
}
`,
			wantCode: http.StatusOK,
			wantBody: "系统异常",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			server := gin.Default()
			h := NewUserHandler(tc.mock(ctrl), nil, nil)
			h.RegisterRoutes(server)

			req, err := http.NewRequest(http.MethodPost, "/users/signup",
				bytes.NewBuffer([]byte(tc.reqbody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			server.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resp.Body.String())

		})
	}
}

func TestMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersvc := svcmocks.NewMockUserService(ctrl)
	usersvc.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(errors.New("mock error"))

	//usersvc := svcmocks.NewMockUserService(ctrl)
	//usersvc.EXPECT().SignUp(gomock.Any(), domain.User{
	//	Email: "124@qq.com",
	//}).Return(errors.New("mock error"))

	err := usersvc.SignUp(context.Background(), domain.User{
		Email: "123@qq.com",
	})
	t.Log(err)
}
