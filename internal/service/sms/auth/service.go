package auth

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"webook/internal/service/sms"
)

// SMSService 权限管理
type SMSService struct {
	svc sms.Service
	key string
}

func NewSMSService(svc sms.Service) sms.Service {
	return &SMSService{
		svc: svc,
	}
}

// Send 发送， 其中biz必须是线下申请的token
func (S *SMSService) Send(ctx context.Context, biz string, param []string, numbers []string) error {

	var tc Claims
	// 如果这里能解析成功，说明就是对应的业务方
	token, err := jwt.ParseWithClaims(biz, &tc, func(token *jwt.Token) (interface{}, error) {
		return S.key, nil
	})
	if err != nil {
		return err
	}
	if token.Valid {
		return errors.New("token 不合法")
	}

	return S.svc.Send(ctx, tc.Tpl, param, numbers)
}

type Claims struct {
	jwt.RegisteredClaims
	Tpl string
}

func GenerateToken(ctx context.Context, tpl string) (string, error) {
	claims := Claims{
		Tpl: tpl,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString([]byte("****"))
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}
