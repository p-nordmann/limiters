package limiters

import "context"

type Limiter interface {
	Limit(ctx context.Context) error
}
