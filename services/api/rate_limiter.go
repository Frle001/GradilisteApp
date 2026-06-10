package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ipBucket struct {
	mu        sync.Mutex
	tokens    float64
	lastRefil time.Time
}

type rateLimiterStore struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
	rate    float64 // tokens added per second
	burst   float64 // maximum token capacity
}

func newRateLimiterStore(rate, burst float64) *rateLimiterStore {
	s := &rateLimiterStore{
		buckets: make(map[string]*ipBucket),
		rate:    rate,
		burst:   burst,
	}
	go s.pruneLoop()
	return s
}

func (s *rateLimiterStore) allow(ip string) bool {
	s.mu.Lock()
	b, ok := s.buckets[ip]
	if !ok {
		b = &ipBucket{tokens: s.burst, lastRefil: time.Now()}
		s.buckets[ip] = b
	}
	s.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefil).Seconds()
	b.tokens += elapsed * s.rate
	if b.tokens > s.burst {
		b.tokens = s.burst
	}
	b.lastRefil = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (s *rateLimiterStore) pruneLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		s.mu.Lock()
		for ip, b := range s.buckets {
			b.mu.Lock()
			idle := b.lastRefil.Before(cutoff)
			b.mu.Unlock()
			if idle {
				delete(s.buckets, ip)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimitMiddleware limits requests per IP using a token bucket.
// rate = tokens/second (sustained throughput), burst = max tokens (peak).
// Note: in-memory only — does not coordinate across multiple instances.
// Use Redis or a managed gateway rate limiter for multi-instance production.
func RateLimitMiddleware(rate, burst float64) gin.HandlerFunc {
	store := newRateLimiterStore(rate, burst)
	return func(c *gin.Context) {
		if !store.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests — please try again later",
			})
			return
		}
		c.Next()
	}
}
