package reservoir

import (
	"context"
	"sync"
	"time"

	"github.com/p-nordmann/limiters"
)

type token struct{}

// Struct implementing the Limiter interface.
type limiter struct {
	maxTokens      int
	refillDuration time.Duration
	in             chan token
	out            chan token
	numConcurrent  int
	mutex          sync.Mutex
}

// Creates a new reservoir limiter.
func NewLimiter(maxTokens int, refillDuration time.Duration) limiters.Limiter {
	l := &limiter{
		maxTokens:      maxTokens,
		refillDuration: refillDuration,
		in:             make(chan token),
		out:            make(chan token),
	}
	return l
}

// Blocks until a token is available or the context is canceled.
func (l *limiter) Limit(ctx context.Context) error {
	l.mutex.Lock()
	if l.numConcurrent == 0 {
		l.numConcurrent = 2
		go l.manageTokens()
	} else {
		l.numConcurrent++
	}
	l.mutex.Unlock()
	select {
	case <-l.out:
		l.mutex.Lock()
		l.numConcurrent--
		l.mutex.Unlock()
		return nil
	case <-ctx.Done():
		l.mutex.Lock()
		l.numConcurrent--
		l.mutex.Unlock()
		return ctx.Err()
	}
}

// Manages the tokens in the reservoir (distribution and refill).
func (l *limiter) manageTokens() {
	tokenCount := l.maxTokens
	for {
		switch tokenCount {
		case 0:
			<-l.in
			tokenCount++
		case l.maxTokens:
			l.mutex.Lock()
			if l.numConcurrent == 1 {
				// No one is waiting: free resources.
				l.numConcurrent = 0
				l.mutex.Unlock()
				return
			}
			l.mutex.Unlock()
			l.out <- token{}
			tokenCount--
			go l.refillTokens()
		default:
			select {
			case <-l.in:
				tokenCount++
			case l.out <- token{}:
				tokenCount--
			}
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
