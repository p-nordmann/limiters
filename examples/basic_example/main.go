package main

import (
	"context"
	"time"

	"github.com/p-nordmann/limiters/reservoir"
)

func main() {
	limiter := reservoir.NewLimiter(4, time.Second)
	ctx := context.Background()

	for i := 0; i < 6; i++ {
		go func(id int) {
			if err := limiter.Limit(ctx); err == nil {
				println("Goroutine", id, "proceeded at", time.Now().Format("15:04:05"))
			} else {
				println("Goroutine", id, "was canceled")
			}
		}(i)
	}

	time.Sleep(5 * time.Second) // Wait to observe the output and refill 3 tokens

	for i := 0; i < 6; i++ {
		go func(id int) {
			if err := limiter.Limit(ctx); err == nil {
				println("Goroutine", id, "proceeded at", time.Now().Format("15:04:05"))
			} else {
				println("Goroutine", id, "was canceled")
			}
		}(i)
	}

	time.Sleep(15 * time.Second) // Wait to observe the output
}
