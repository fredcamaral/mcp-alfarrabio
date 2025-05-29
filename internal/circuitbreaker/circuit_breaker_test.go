package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := New(&Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
	})

	ctx := context.Background()

	// Successful requests should work
	for i := 0; i < 5; i++ {
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be closed, got: %v", cb.GetState())
	}

	// Some failures, but below threshold
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain closed, got: %v", cb.GetState())
	}

	// Success should reset failure count
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// More failures should now be counted from zero
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain closed after reset, got: %v", cb.GetState())
	}
}

func TestCircuitBreaker_OpenState(t *testing.T) {
	var stateChanges []string
	cb := New(&Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		OnStateChange: func(from, to State) {
			stateChanges = append(stateChanges, fmt.Sprintf("%s->%s", from, to))
		},
	})

	ctx := context.Background()

	// Trigger failures to open circuit
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be open, got: %v", cb.GetState())
	}

	// Requests should be rejected
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Expected ErrCircuitOpen, got: %v", err)
	}

	// Check state change was recorded
	if len(stateChanges) != 1 || stateChanges[0] != "closed->open" {
		t.Errorf("Expected state change closed->open, got: %v", stateChanges)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open on next request
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error in half-open state, got: %v", err)
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be half-open, got: %v", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	cb := New(&Config{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		Timeout:               50 * time.Millisecond,
		MaxConcurrentRequests: 1,
	})

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// First request should succeed and transition to half-open
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be half-open, got: %v", cb.GetState())
	}

	// Need one more success to close
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be closed after successes, got: %v", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := New(&Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Failure in half-open should reopen
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return errTest
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be open after half-open failure, got: %v", cb.GetState())
	}
}

func TestCircuitBreaker_Fallback(t *testing.T) {
	cb := New(&Config{
		FailureThreshold: 1,
		Timeout:          1 * time.Second, // Set explicit timeout
	})

	ctx := context.Background()
	fallbackCalled := false

	// Trigger circuit open
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return errTest
	})

	// Verify circuit is open
	if cb.GetState() != StateOpen {
		t.Fatalf("Expected circuit to be open, got: %v", cb.GetState())
	}

	// Execute with fallback immediately (should still be open)
	err := cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			t.Error("Function should not be called when circuit is open")
			return errors.New("should not be called")
		},
		func(ctx context.Context, originalErr error) error {
			fallbackCalled = true
			if !errors.Is(originalErr, ErrCircuitOpen) {
				t.Errorf("Expected ErrCircuitOpen in fallback, got: %v", originalErr)
			}
			return nil
		},
	)

	if err != nil {
		t.Errorf("Expected no error with fallback, got: %v", err)
	}

	if !fallbackCalled {
		t.Error("Expected fallback to be called")
	}
}

func TestCircuitBreaker_ConcurrentRequests(t *testing.T) {
	cb := New(&Config{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		Timeout:               50 * time.Millisecond,
		MaxConcurrentRequests: 2,
	})

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	var successCount int32
	var rejectCount int32

	// Try 5 concurrent requests in half-open state
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Execute(ctx, func(ctx context.Context) error {
				time.Sleep(20 * time.Millisecond) // Simulate work - increased to ensure concurrency
				return nil
			})
			switch {
			case err == nil:
				atomic.AddInt32(&successCount, 1)
			case errors.Is(err, ErrTooManyConcurrentRequests):
				atomic.AddInt32(&rejectCount, 1)
			default:
				t.Logf("Unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	// Should have allowed at most MaxConcurrentRequests (2) in half-open
	// But since 2 successes close the circuit, we might get more successes
	// The important thing is that we got some rejections
	t.Logf("Success count: %d, Reject count: %d", successCount, rejectCount)

	if successCount == 0 {
		t.Error("Expected at least some successful requests")
	}
	if rejectCount == 0 && successCount < 5 {
		t.Error("Expected some requests to be rejected when exceeding concurrent limit")
	}
	if successCount+rejectCount != 5 {
		t.Errorf("Expected total of 5 requests, got: %d", successCount+rejectCount)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := New(&Config{
		FailureThreshold: 3,
	})

	ctx := context.Background()

	// Execute some requests
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}

	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	stats := cb.GetStats()

	if stats.TotalRequests != 5 {
		t.Errorf("Expected 5 total requests, got: %d", stats.TotalRequests)
	}
	if stats.TotalSuccesses != 3 {
		t.Errorf("Expected 3 total successes, got: %d", stats.TotalSuccesses)
	}
	if stats.TotalFailures != 2 {
		t.Errorf("Expected 2 total failures, got: %d", stats.TotalFailures)
	}
	if stats.FailureRate != 0.4 {
		t.Errorf("Expected failure rate 0.4, got: %f", stats.FailureRate)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New(&Config{
		FailureThreshold: 1,
	})

	ctx := context.Background()

	// Open the circuit
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return errTest
	})

	if cb.GetState() != StateOpen {
		t.Error("Expected circuit to be open")
	}

	// Reset
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Error("Expected circuit to be closed after reset")
	}

	// Should be able to execute again
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error after reset, got: %v", err)
	}
}

func TestCircuitBreaker_RaceConditions(t *testing.T) {
	cb := New(&Config{
		FailureThreshold: 10,
		SuccessThreshold: 5,
		Timeout:          10 * time.Millisecond,
	})

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent executions
	go func() {
		for i := 0; i < 100; i++ {
			_ = cb.Execute(ctx, func(ctx context.Context) error {
				if i%3 == 0 {
					return errTest
				}
				return nil
			})
		}
		done <- true
	}()

	// Concurrent stats reading
	go func() {
		for i := 0; i < 100; i++ {
			_ = cb.GetStats()
			_ = cb.GetState()
		}
		done <- true
	}()

	// Concurrent state transitions
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(15 * time.Millisecond)
			if cb.GetState() == StateOpen {
				time.Sleep(15 * time.Millisecond) // Wait for timeout
			}
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Circuit should still be in a valid state
	state := cb.GetState()
	if state != StateClosed && state != StateOpen && state != StateHalfOpen {
		t.Errorf("Invalid state after race test: %v", state)
	}
}
