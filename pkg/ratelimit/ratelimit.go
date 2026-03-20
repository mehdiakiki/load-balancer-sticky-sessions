package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters      map[string]*rate.Limiter
	mux           sync.RWMutex
	rateLimit     rate.Limit
	burst         int
	cleanupTicker *time.Ticker
}

func NewRateLimiter(requestsPerSecond float64, burst int, cleanupInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limiters:  make(map[string]*rate.Limiter),
		rateLimit: rate.Limit(requestsPerSecond),
		burst:     burst,
	}

	rl.cleanupTicker = time.NewTicker(cleanupInterval)
	go rl.cleanup(cleanupInterval)

	return rl
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mux.Lock()
	defer rl.mux.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rateLimit, rl.burst)
		rl.limiters[key] = limiter
	}

	return limiter
}

func (rl *RateLimiter) cleanup(interval time.Duration) {
	for range rl.cleanupTicker.C {
		rl.mux.Lock()
		now := time.Now()
		for key, limiter := range rl.limiters {
			if limiter.Allow() {
				limiter.AllowN(now, limiter.Burst())
			} else {
				delete(rl.limiters, key)
			}
		}
		rl.mux.Unlock()
	}
}

func (rl *RateLimiter) Stop() {
	rl.cleanupTicker.Stop()
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr

		limiter := rl.getLimiter(key)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
