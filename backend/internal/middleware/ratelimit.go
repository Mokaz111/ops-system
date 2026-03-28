package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter 按客户端 IP 限流，带 TTL 自动清理过期条目。
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit
	burst    int
	ttl      time.Duration
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	if rps <= 0 {
		rps = 100
	}
	if burst <= 0 {
		burst = 200
	}
	i := &IPRateLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        rate.Limit(rps),
		burst:    burst,
		ttl:      10 * time.Minute,
	}
	go i.cleanupLoop()
	return i
}

func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.limiters[ip]
	if !ok {
		entry = &limiterEntry{
			limiter:  rate.NewLimiter(i.r, i.burst),
			lastSeen: time.Now(),
		}
		i.limiters[ip] = entry
	} else {
		entry.lastSeen = time.Now()
	}
	return entry.limiter
}

func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(i.ttl)
	defer ticker.Stop()
	for range ticker.C {
		i.mu.Lock()
		cutoff := time.Now().Add(-i.ttl)
		for ip, entry := range i.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(i.limiters, ip)
			}
		}
		i.mu.Unlock()
	}
}

// RateLimit 返回 Gin 中间件：超过配额返回 429。
func (i *IPRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !i.getLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    http.StatusTooManyRequests,
				"message": "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}
