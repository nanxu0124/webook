package sms

import "context"

type Service interface {
	Send(ctx context.Context, tpl string, param []string, numbers []string) error
}
