package httpapi

import (
	"sync"
	"time"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type rateLimiter struct {
	mu    sync.Mutex
	rps   float64
	burst int
	bkts  map[string]*bucket // key: ip
}

func newRateLimiter(rps float64, burst int) *rateLimiter {
	return &rateLimiter{rps: rps, burst: burst, bkts: make(map[string]*bucket)}
}

func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bkt, ok := rl.bkts[key]
	if !ok {
		bkt = &bucket{tokens: float64(rl.burst), lastRefill: now}
		rl.bkts[key] = bkt
	}

	elapsed := now.Sub(bkt.lastRefill).Seconds()
	bkt.tokens = minFloat(float64(rl.burst), bkt.tokens+elapsed*rl.rps)
	bkt.lastRefill = now

	if bkt.tokens >= 1 {
		bkt.tokens -= 1
		return true
	}
	return false
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
