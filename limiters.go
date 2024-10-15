package limiters

import "context"

type Limiter interface {
	Limit(ctx context.Context) error
}

type Scheduler interface {
	Schedule(ctx context.Context, f func(ctx context.Context) error) error
}
