package reservoir

import (
	"context"
	"time"

	"github.com/p-nordmann/limiters"
)

// TODO find a way to be garbage collected
//	right now, there is an infinite loop in a goroutine

type token struct{}

// Struct implementing the Limiter interface.
type limiter struct {
	maxTokens      int
	refillDuration time.Duration
	in             chan token
	out            chan token
}

// Creates a new reservoir limiter.
func NewLimiter(maxTokens int, refillDuration time.Duration) limiters.Limiter {
	l := &limiter{
		maxTokens:      maxTokens,
		refillDuration: refillDuration,
		in:             make(chan token),
		out:            make(chan token),
	}
	go l.manageTokens()
	return l
}

// Blocks until a token is available or the context is canceled.
func (l *limiter) Limit(ctx context.Context) error {
	select {
	case <-l.out:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Manages the tokens in the reservoir (distribution and refill).
func (l *limiter) manageTokens() {
	tokenCount := l.maxTokens
	for {
		// Reservoir empty.
		if tokenCount == 0 {
			<-l.in
			tokenCount++
			continue
		}

		// Reservoir full.
		if tokenCount == l.maxTokens {
			l.out <- token{}
			tokenCount--
			go l.refillTokens()
			continue
		}

		// In between.
		select {
		case <-l.in:
			tokenCount++
		case l.out <- token{}:
			tokenCount--
		}
	}
}

// Starts a ticker to refill missing tokens.
func (l *limiter) refillTokens() {
	ticker := time.NewTicker(l.refillDuration)
	for range ticker.C {
		select {
		case l.in <- token{}:
			// Token refilled, wait for next tick.
		default:
			// Reservoir full, stop ticking.
			ticker.Stop()
			return
		}
	}
}
