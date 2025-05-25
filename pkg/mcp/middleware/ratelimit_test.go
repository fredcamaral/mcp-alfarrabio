package middleware

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucket(t *testing.T) {
	t.Run("basic allow", func(t *testing.T) {
		tb := NewTokenBucket(10, 5) // 10 tokens/sec, burst of 5
		
		// Should allow initial burst
		for i := 0; i < 5; i++ {
			assert.True(t, tb.Allow(1), "request %d should be allowed", i+1)
		}
		
		// 6th request should be denied
		assert.False(t, tb.Allow(1))
	})
	
	t.Run("token refill", func(t *testing.T) {
		tb := NewTokenBucket(10, 5) // 10 tokens/sec, burst of 5
		
		// Use all tokens
		for i := 0; i < 5; i++ {
			assert.True(t, tb.Allow(1))
		}
		assert.False(t, tb.Allow(1))
		
		// Wait for tokens to refill
		time.Sleep(200 * time.Millisecond) // Should get ~2 tokens
		assert.True(t, tb.Allow(1))
		assert.True(t, tb.Allow(1))
		assert.False(t, tb.Allow(1))
	})
	
	t.Run("allow multiple tokens", func(t *testing.T) {
		tb := NewTokenBucket(10, 10)
		
		// Should allow taking multiple tokens at once
		assert.True(t, tb.Allow(5))
		assert.True(t, tb.Allow(5))
		assert.False(t, tb.Allow(1))
	})
	
	t.Run("wait for tokens", func(t *testing.T) {
		tb := NewTokenBucket(10, 5)
		
		// Use all tokens
		for i := 0; i < 5; i++ {
			tb.Allow(1)
		}
		
		// Wait should succeed after delay
		ctx := context.Background()
		start := time.Now()
		err := tb.Wait(ctx, 2)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Greater(t, duration, 100*time.Millisecond)
		assert.Less(t, duration, 300*time.Millisecond)
	})
	
	t.Run("wait context cancellation", func(t *testing.T) {
		tb := NewTokenBucket(1, 1) // Very slow rate
		
		// Use all tokens
		tb.Allow(1)
		
		// Cancel context during wait
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		
		err := tb.Wait(ctx, 1)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
	
	t.Run("reservation", func(t *testing.T) {
		tb := NewTokenBucket(10, 5)
		
		// Make a reservation
		r := tb.Reserve(3)
		assert.True(t, r.OK())
		assert.Zero(t, r.Delay())
		
		// Should have 2 tokens left
		assert.True(t, tb.Allow(2))
		assert.False(t, tb.Allow(1))
		
		// Reserve more than available
		r2 := tb.Reserve(10)
		assert.True(t, r2.OK())
		assert.Greater(t, r2.Delay(), time.Duration(0))
	})
	
	t.Run("reservation cancellation", func(t *testing.T) {
		tb := NewTokenBucket(10, 5)
		
		// Make and cancel a reservation
		r := tb.Reserve(3)
		assert.True(t, r.OK())
		
		r.Cancel()
		assert.False(t, r.OK()) // Should be marked as cancelled
		
		// Tokens should be returned
		assert.True(t, tb.Allow(5))
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("global rate limiting", func(t *testing.T) {
		config := &RateLimitConfig{
			Rate:   5,
			Burst:  2,
			Global: true,
		}
		middleware := NewRateLimitMiddleware(config)
		defer middleware.Stop()
		
		successCount := int32(0)
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			atomic.AddInt32(&successCount, 1)
			return "response", nil
		}
		
		// First 2 requests should succeed (burst)
		for i := 0; i < 2; i++ {
			resp, err := middleware.Process(context.Background(), "request", handler)
			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
		}
		
		// 3rd request should fail
		_, err := middleware.Process(context.Background(), "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrRateLimitExceeded, err)
		assert.Equal(t, int32(2), atomic.LoadInt32(&successCount))
	})
	
	t.Run("per-user rate limiting", func(t *testing.T) {
		config := &RateLimitConfig{
			Rate:    5,
			Burst:   2,
			PerUser: true,
		}
		middleware := NewRateLimitMiddleware(config)
		defer middleware.Stop()
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}
		
		// Create contexts for different users
		user1Ctx := context.WithValue(context.Background(), ContextKeyUser, &User{ID: "user1"})
		user2Ctx := context.WithValue(context.Background(), ContextKeyUser, &User{ID: "user2"})
		
		// Each user should have their own limit
		for i := 0; i < 2; i++ {
			resp, err := middleware.Process(user1Ctx, "request", handler)
			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
			
			resp, err = middleware.Process(user2Ctx, "request", handler)
			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
		}
		
		// User 1 should be rate limited
		_, err := middleware.Process(user1Ctx, "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrRateLimitExceeded, err)
		
		// User 2 should also be rate limited
		_, err = middleware.Process(user2Ctx, "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrRateLimitExceeded, err)
	})
	
	t.Run("per-IP rate limiting", func(t *testing.T) {
		config := &RateLimitConfig{
			Rate:  5,
			Burst: 2,
			PerIP: true,
		}
		middleware := NewRateLimitMiddleware(config)
		defer middleware.Stop()
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}
		
		// Create contexts for different IPs
		ip1Ctx := context.WithValue(context.Background(), "RemoteAddr", "192.168.1.1")
		ip2Ctx := context.WithValue(context.Background(), "RemoteAddr", "192.168.1.2")
		
		// Each IP should have their own limit
		for i := 0; i < 2; i++ {
			resp, err := middleware.Process(ip1Ctx, "request", handler)
			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
			
			resp, err = middleware.Process(ip2Ctx, "request", handler)
			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
		}
		
		// Both IPs should be rate limited
		_, err := middleware.Process(ip1Ctx, "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrRateLimitExceeded, err)
		
		_, err = middleware.Process(ip2Ctx, "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrRateLimitExceeded, err)
	})
	
	t.Run("combined per-user and per-IP", func(t *testing.T) {
		config := &RateLimitConfig{
			Rate:    5,
			Burst:   2,
			PerUser: true,
			PerIP:   true,
		}
		middleware := NewRateLimitMiddleware(config)
		defer middleware.Stop()
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}
		
		// Create context with both user and IP
		ctx := context.WithValue(context.Background(), ContextKeyUser, &User{ID: "user1"})
		ctx = context.WithValue(ctx, "RemoteAddr", "192.168.1.1")
		
		// Should create a combined identifier
		for i := 0; i < 2; i++ {
			resp, err := middleware.Process(ctx, "request", handler)
			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
		}
		
		_, err := middleware.Process(ctx, "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrRateLimitExceeded, err)
	})
	
	t.Run("no identifier", func(t *testing.T) {
		config := &RateLimitConfig{
			Rate:    5,
			Burst:   10,
			PerUser: true, // But no user in context
		}
		middleware := NewRateLimitMiddleware(config)
		defer middleware.Stop()
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}
		
		// Should allow requests when no identifier is found
		for i := 0; i < 5; i++ {
			resp, err := middleware.Process(context.Background(), "request", handler)
			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
		}
	})
}

func TestRateLimitMiddlewareCleanup(t *testing.T) {
	config := &RateLimitConfig{
		Rate:            10,
		Burst:           5,
		PerUser:         true,
		CleanupInterval: 100 * time.Millisecond,
		TTL:             200 * time.Millisecond,
	}
	middleware := NewRateLimitMiddleware(config)
	defer middleware.Stop()
	
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}
	
	// Create requests for multiple users
	for i := 0; i < 5; i++ {
		ctx := context.WithValue(context.Background(), ContextKeyUser, &User{ID: fmt.Sprintf("user%d", i)})
		middleware.Process(ctx, "request", handler)
	}
	
	// Verify limiters were created
	middleware.mu.RLock()
	initialCount := len(middleware.limiters)
	middleware.mu.RUnlock()
	assert.Equal(t, 5, initialCount)
	
	// Wait for cleanup to run
	time.Sleep(400 * time.Millisecond)
	
	// Limiters should be cleaned up
	middleware.mu.RLock()
	finalCount := len(middleware.limiters)
	middleware.mu.RUnlock()
	assert.Equal(t, 0, finalCount)
}

func TestRateLimitMiddlewareWaitN(t *testing.T) {
	config := &RateLimitConfig{
		Rate:    10,
		Burst:   5,
		PerUser: true,
	}
	middleware := NewRateLimitMiddleware(config)
	defer middleware.Stop()
	
	identifier := "user:test"
	
	// Use all tokens
	limiter := middleware.getLimiter(identifier)
	for i := 0; i < 5; i++ {
		limiter.Allow(1)
	}
	
	// WaitN should succeed after delay
	ctx := context.Background()
	start := time.Now()
	err := middleware.WaitN(ctx, identifier, 2)
	duration := time.Since(start)
	
	assert.NoError(t, err)
	assert.Greater(t, duration, 100*time.Millisecond)
}

func TestRateLimitMiddlewareReserveN(t *testing.T) {
	config := &RateLimitConfig{
		Rate:    10,
		Burst:   5,
		PerUser: true,
	}
	middleware := NewRateLimitMiddleware(config)
	defer middleware.Stop()
	
	identifier := "user:test"
	
	// Make a reservation
	reservation := middleware.ReserveN(identifier, 3)
	assert.True(t, reservation.OK())
	assert.Zero(t, reservation.Delay())
	
	// Cancel the reservation
	reservation.Cancel()
	
	// Should be able to reserve 5 tokens now
	reservation2 := middleware.ReserveN(identifier, 5)
	assert.True(t, reservation2.OK())
	assert.Zero(t, reservation2.Delay())
}

func TestRateLimitMiddlewareStats(t *testing.T) {
	config := &RateLimitConfig{
		Rate:    10,
		Burst:   5,
		PerUser: true,
		Global:  true,
	}
	middleware := NewRateLimitMiddleware(config)
	defer middleware.Stop()
	
	// Create some activity
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}
	
	ctx1 := context.WithValue(context.Background(), ContextKeyUser, &User{ID: "user1"})
	ctx2 := context.WithValue(context.Background(), ContextKeyUser, &User{ID: "user2"})
	
	middleware.Process(ctx1, "request", handler)
	middleware.Process(ctx2, "request", handler)
	
	// Get stats
	stats := middleware.Stats()
	
	assert.Equal(t, 2, stats["active_limiters"])
	
	configStats := stats["config"].(map[string]interface{})
	assert.Equal(t, 10.0, configStats["rate"])
	assert.Equal(t, 5, configStats["burst"])
	assert.True(t, configStats["per_user"].(bool))
	assert.True(t, configStats["global"].(bool))
	
	limiters := stats["limiters"].(map[string]interface{})
	assert.Len(t, limiters, 2)
}

func TestConcurrentRateLimiting(t *testing.T) {
	config := &RateLimitConfig{
		Rate:    100,
		Burst:   10,
		PerUser: true,
	}
	middleware := NewRateLimitMiddleware(config)
	defer middleware.Stop()
	
	successCount := int32(0)
	failCount := int32(0)
	
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		atomic.AddInt32(&successCount, 1)
		return "response", nil
	}
	
	// Run concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		userID := fmt.Sprintf("user%d", i)
		for j := 0; j < 20; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				ctx := context.WithValue(context.Background(), ContextKeyUser, &User{ID: userID})
				_, err := middleware.Process(ctx, "request", handler)
				if err == ErrRateLimitExceeded {
					atomic.AddInt32(&failCount, 1)
				}
			}()
		}
	}
	
	wg.Wait()
	
	// Each user should have at most burst successful requests
	assert.LessOrEqual(t, atomic.LoadInt32(&successCount), int32(5*10))
	assert.Greater(t, atomic.LoadInt32(&failCount), int32(0))
}

func BenchmarkTokenBucket(b *testing.B) {
	tb := NewTokenBucket(1000, 100)
	
	b.Run("Allow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tb.Allow(1)
		}
	})
	
	b.Run("AllowN", func(b *testing.B) {
		now := time.Now()
		for i := 0; i < b.N; i++ {
			tb.AllowN(now, 1)
		}
	})
}

func BenchmarkRateLimitMiddleware(b *testing.B) {
	config := &RateLimitConfig{
		Rate:    1000,
		Burst:   100,
		PerUser: true,
	}
	middleware := NewRateLimitMiddleware(config)
	defer middleware.Stop()
	
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}
	
	ctx := context.WithValue(context.Background(), ContextKeyUser, &User{ID: "user1"})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.Process(ctx, "request", handler)
	}
}

func BenchmarkConcurrentRateLimiting(b *testing.B) {
	config := &RateLimitConfig{
		Rate:    10000,
		Burst:   1000,
		PerUser: true,
	}
	middleware := NewRateLimitMiddleware(config)
	defer middleware.Stop()
	
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}
	
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.WithValue(context.Background(), ContextKeyUser, &User{ID: "user1"})
		for pb.Next() {
			middleware.Process(ctx, "request", handler)
		}
	})
}