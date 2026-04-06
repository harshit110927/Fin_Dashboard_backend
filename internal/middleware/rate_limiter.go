package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"finance-dashboard/pkg/response"
)

// RateLimiter creates an IP-based in-memory rate limiter with a
// configurable requests-per-minute ceiling per client IP.
//
// Trade-off: state is in-memory and not shared across instances.
// For multi-instance deployments this should be backed by Redis
// with atomic INCR/EXPIRE. For single-instance deployment this
// is correct and avoids an infrastructure dependency.
func RateLimiter(rpm int) gin.HandlerFunc {
	type bucket struct {
		count int
		reset time.Time
	}
	var (
		mu      sync.Mutex
		buckets = make(map[string]*bucket)
	)

	// Background goroutine cleans up expired buckets every 5 minutes
	// to prevent unbounded memory growth from unique IPs over time.
	go func() {
		for range time.Tick(5 * time.Minute) {
			mu.Lock()
			now := time.Now()
			for ip, b := range buckets {
				if now.After(b.reset) {
					delete(buckets, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		b, ok := buckets[ip]
		if !ok || time.Now().After(b.reset) {
			buckets[ip] = &bucket{count: 1, reset: time.Now().Add(time.Minute)}
			mu.Unlock()
			c.Next()
			return
		}
		b.count++
		if b.count > rpm {
			mu.Unlock()
			response.Error(c, http.StatusTooManyRequests,
				"RATE_LIMIT_EXCEEDED",
				"Too many requests. Try again in a moment.")
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}
