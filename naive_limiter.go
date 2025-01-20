package limiters

import (
	"context"
	"sync"
	"time"
)

type naiveLimiter struct {
	maxTokens       int
	refillDuration  time.Duration
	availableTokens int
	tokenLock       sync.Mutex
}

func NewNaiveLimiter(maxTokens int, refillDuration time.Duration) Limiter {
	l := &naiveLimiter{
		maxTokens:       maxTokens,
		refillDuration:  refillDuration,
		availableTokens: maxTokens,
	}
	go l.refillTokens()
	return l
}

func (l *naiveLimiter) Limit(ctx context.Context) error {
	l.tokenLock.Lock()
	if l.availableTokens > 0 {
		l.availableTokens--
		l.tokenLock.Unlock()
		return nil
	}
	l.tokenLock.Unlock()

	ticker := time.NewTicker(l.refillDuration)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			l.tokenLock.Lock()
			if l.availableTokens > 0 {
				l.availableTokens--
				l.tokenLock.Unlock()
				return nil
			}
			l.tokenLock.Unlock()
		}
	}
}

func (l *naiveLimiter) refillTokens() {
	ticker := time.NewTicker(l.refillDuration)
	for range ticker.C {
		l.tokenLock.Lock()
		if l.availableTokens < l.maxTokens {
			l.availableTokens++
		}
		l.tokenLock.Unlock()
	}
}
