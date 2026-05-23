package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerHour int
	CleanupInterval time.Duration
}

// RateLimiter implements token bucket rate limiting per user
type RateLimiter struct {
	config   RateLimitConfig
	mu       sync.RWMutex
	buckets  map[string]*TokenBucket
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// TokenBucket tracks requests for a single user
type TokenBucket struct {
	Tokens     float64
	LastRefill time.Time
	Capacity   float64
	RefillRate float64 // tokens per second
}

// NewRateLimiter creates a new rate limiter instance
// requestsPerHour: max requests per user per hour (e.g., 10)
func NewRateLimiter(requestsPerHour int) *RateLimiter {
	rl := &RateLimiter{
		config: RateLimitConfig{
			RequestsPerHour: requestsPerHour,
			CleanupInterval: 1 * time.Hour,
		},
		buckets:  make(map[string]*TokenBucket),
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine to remove stale buckets
	rl.wg.Add(1)
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a user can make a request
// Returns true if within rate limit, false otherwise
func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[userID]
	if !exists {
		// New user: create bucket with full capacity
		bucket = &TokenBucket{
			Tokens:     float64(rl.config.RequestsPerHour),
			LastRefill: time.Now(),
			Capacity:   float64(rl.config.RequestsPerHour),
			RefillRate: float64(rl.config.RequestsPerHour) / 3600.0, // tokens per second
		}
		rl.buckets[userID] = bucket
		bucket.Tokens--
		return true
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.LastRefill).Seconds()
	bucket.Tokens = min(bucket.Capacity, bucket.Tokens+elapsed*bucket.RefillRate)
	bucket.LastRefill = now

	// Check if user has tokens available
	if bucket.Tokens >= 1.0 {
		bucket.Tokens--
		return true
	}

	return false
}

// GetRemaining returns the approximate number of remaining requests for a user
func (rl *RateLimiter) GetRemaining(userID string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	bucket, exists := rl.buckets[userID]
	if !exists {
		return rl.config.RequestsPerHour
	}

	return int(bucket.Tokens)
}

// GetResetTime returns when the user's rate limit will be fully reset
func (rl *RateLimiter) GetResetTime(userID string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	bucket, exists := rl.buckets[userID]
	if !exists {
		return time.Now().Add(1 * time.Hour)
	}

	// Estimate reset time based on current tokens and refill rate
	tokensNeeded := bucket.Capacity - bucket.Tokens
	secondsUntilReset := tokensNeeded / bucket.RefillRate
	return bucket.LastRefill.Add(time.Duration(secondsUntilReset) * time.Second)
}

// cleanupLoop removes stale buckets periodically
func (rl *RateLimiter) cleanupLoop() {
	defer rl.wg.Done()
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopChan:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// cleanup removes buckets that haven't been used in the last cleanup interval
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for userID, bucket := range rl.buckets {
		if now.Sub(bucket.LastRefill) > rl.config.CleanupInterval {
			delete(rl.buckets, userID)
		}
	}
}

// Stop gracefully shuts down the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
	rl.wg.Wait()
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
// Applies to authenticated users (requires user_id in context)
// Unauthenticated requests are allowed through
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user_id from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			// Unauthenticated request: allow through
			// (rate limiting applies only to authenticated users)
			c.Next()
			return
		}

		uid, ok := userID.(string)
		if !ok || uid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id"})
			c.Abort()
			return
		}

		// Check rate limit
		if !limiter.Allow(uid) {
			resetTime := limiter.GetResetTime(uid)
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":             "rate limit exceeded",
				"retry_after_unix":  resetTime.Unix(),
				"requests_per_hour": limiter.config.RequestsPerHour,
			})
			c.Abort()
			return
		}

		// Add remaining requests to response headers
		remaining := limiter.GetRemaining(uid)
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerHour))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limiter.GetResetTime(uid).Unix()))

		c.Next()
	}
}

// min returns the minimum of two integers
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
