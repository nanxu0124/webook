package ratelimit

import "golang.org/x/net/context"

type Limiter interface {
	Limit(ctx context.Context, key string) (bool, error)
}
