package service

import (
	"context"
	"fmt"
	"math/rand"
	"webook/internal/repository"
	"webook/internal/service/sms"
)

const codeTplId = "2044585"

var (
	ErrCodeSendTooMany        = repository.ErrCodeSendTooMany
	ErrCodeVerifyTooManyTimes = repository.ErrCodeVerifyTooManyTimes
)

type CodeService interface {
	Send(ctx context.Context, biz string, phone string) error
	Verify(ctx context.Context, biz string, phone string, code string) (bool, error)
}

type SMSCodeService struct {
	smsSvc sms.Service
	repo   repository.CodeRepository
}

func NewCodeService(repo repository.CodeRepository, smsSvc sms.Service) CodeService {
	return &SMSCodeService{
		repo:   repo,
		smsSvc: smsSvc,
	}
}
func (svc *SMSCodeService) Send(ctx context.Context, biz string, phone string) error {
	code := svc.generate()
	err := svc.repo.Store(ctx, biz, phone, code)
	if err != nil {
		return err
	}
	err = svc.smsSvc.Send(ctx, codeTplId, []string{code}, []string{phone})
	return err
}

func (svc *SMSCodeService) Verify(ctx context.Context, biz string, phone string, code string) (bool, error) {
	return svc.repo.Verify(ctx, biz, phone, code)
}

func (svc *SMSCodeService) generate() string {
	// 用随机数生成一个
	num := rand.Intn(1000000)
	return fmt.Sprintf("%06d", num)
}
