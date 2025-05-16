package limiters

import (
	"context"
	"sync"
	"time"
)

type token struct{}

// Struct implementing the Limiter interface.
type reservoirLimiter struct {
	maxTokens      int
	refillDuration time.Duration
	in             chan token
	out            chan token

	numListeners     int
	numListenersLock sync.Mutex

	numManagers     int
	numManagersLock sync.Mutex
}

// Creates a new reservoir limiter.
func NewReservoirLimiter(maxTokens int, refillDuration time.Duration) Limiter {
	return &reservoirLimiter{
		maxTokens:      maxTokens,
		refillDuration: refillDuration,
		in:             make(chan token),
		out:            make(chan token),
	}
}

func (l *reservoirLimiter) incrementNumListeners() {
	l.numListenersLock.Lock()
	l.numListeners++
	l.numListenersLock.Unlock()
}

func (l *reservoirLimiter) decrementNumListeners() {
	l.numListenersLock.Lock()
	l.numListeners--
	l.numListenersLock.Unlock()
}

func (l *reservoirLimiter) readNumListeners() int {
	l.numListenersLock.Lock()
	defer l.numListenersLock.Unlock()
	return l.numListeners
}

// Blocks until a token is available or the context is canceled.
func (l *reservoirLimiter) Limit(ctx context.Context) error {
	l.incrementNumListeners()
	defer l.decrementNumListeners()

	l.launchManager()

	select {
	case <-l.out:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *reservoirLimiter) launchManager() {
	l.numManagersLock.Lock()
	defer l.numManagersLock.Unlock()

	if l.numManagers < 0 {
		panic("numManagers is expected to be positive or null")
	}

	if l.numManagers > 0 {
		return
	}

	go l.manageTokens()
	l.numManagers = 1
}

// Manages the tokens in the reservoir (distribution and refill).
func (l *reservoirLimiter) manageTokens() {
	tokenCount := l.maxTokens
	for {
		switch tokenCount {
		case 0:
			<-l.in
			tokenCount++
		case l.maxTokens:
			// If no one is waiting, free resources.
			if l.readNumListeners() == 0 {
				l.numManagersLock.Lock()
				l.numManagers--
				l.numManagersLock.Unlock()
				return
			}
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
func (l *reservoirLimiter) refillTokens() {
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
