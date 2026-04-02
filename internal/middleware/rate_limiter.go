package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"finance-dashboard/pkg/response"
)

type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

var (
	rateMu  sync.Mutex
	rateMap = make(map[string]*rateLimitEntry)
)

func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rateMu.Lock()
			now := time.Now()
			for k, e := range rateMap {
				if now.After(e.windowEnd) {
					delete(rateMap, k)
				}
			}
			rateMu.Unlock()
		}
	}()
}

func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		rateMu.Lock()
		entry, exists := rateMap[ip]
		if !exists || now.After(entry.windowEnd) {
			rateMap[ip] = &rateLimitEntry{count: 1, windowEnd: now.Add(time.Minute)}
			rateMu.Unlock()
			c.Next()
			return
		}
		entry.count++
		if entry.count > 100 {
			rateMu.Unlock()
			response.Error(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many requests")
			return
		}
		rateMu.Unlock()
		c.Next()
	}
}
