package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewarePipeline(t *testing.T) {
	logger := slog.Default()
	
	t.Run("empty pipeline", func(t *testing.T) {
		pipeline := NewPipeline(logger)
		
		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "response", nil
		}
		
		resp, err := pipeline.Execute(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.True(t, called)
	})
	
	t.Run("single middleware", func(t *testing.T) {
		pipeline := NewPipeline(logger)
		
		var order []string
		middleware := MiddlewareFunc(func(ctx context.Context, req interface{}, next Handler) (interface{}, error) {
			order = append(order, "before")
			resp, err := next(ctx, req)
			order = append(order, "after")
			return resp, err
		})
		
		pipeline.Use(middleware)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			order = append(order, "handler")
			return "response", nil
		}
		
		resp, err := pipeline.Execute(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.Equal(t, []string{"before", "handler", "after"}, order)
	})
	
	t.Run("multiple middleware", func(t *testing.T) {
		pipeline := NewPipeline(logger)
		
		var order []string
		
		// Create multiple middleware
		for i := 1; i <= 3; i++ {
			index := i
			middleware := MiddlewareFunc(func(ctx context.Context, req interface{}, next Handler) (interface{}, error) {
				order = append(order, fmt.Sprintf("before-%d", index))
				resp, err := next(ctx, req)
				order = append(order, fmt.Sprintf("after-%d", index))
				return resp, err
			})
			pipeline.Use(middleware)
		}
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			order = append(order, "handler")
			return "response", nil
		}
		
		resp, err := pipeline.Execute(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.Equal(t, []string{
			"before-1", "before-2", "before-3",
			"handler",
			"after-3", "after-2", "after-1",
		}, order)
	})
	
	t.Run("middleware error handling", func(t *testing.T) {
		pipeline := NewPipeline(logger)
		
		expectedErr := errors.New("middleware error")
		middleware := MiddlewareFunc(func(ctx context.Context, req interface{}, next Handler) (interface{}, error) {
			return nil, expectedErr
		})
		
		pipeline.Use(middleware)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		}
		
		resp, err := pipeline.Execute(context.Background(), "request", handler)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)
	})
	
	t.Run("context propagation", func(t *testing.T) {
		pipeline := NewPipeline(logger)
		
		key := "test-key"
		value := "test-value"
		
		middleware := MiddlewareFunc(func(ctx context.Context, req interface{}, next Handler) (interface{}, error) {
			ctx = context.WithValue(ctx, key, value)
			return next(ctx, req)
		})
		
		pipeline.Use(middleware)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			val := ctx.Value(key)
			assert.Equal(t, value, val)
			return "response", nil
		}
		
		_, err := pipeline.Execute(context.Background(), "request", handler)
		assert.NoError(t, err)
	})
}

func TestLoggingMiddleware(t *testing.T) {
	// Create a custom logger to capture logs
	logger := slog.Default()
	
	t.Run("successful request", func(t *testing.T) {
		middleware := NewLoggingMiddleware(logger)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
	
	t.Run("failed request", func(t *testing.T) {
		middleware := NewLoggingMiddleware(logger)
		
		expectedErr := errors.New("handler error")
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, expectedErr
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := slog.Default()
	
	t.Run("no panic", func(t *testing.T) {
		middleware := NewRecoveryMiddleware(logger)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
	
	t.Run("panic with error", func(t *testing.T) {
		middleware := NewRecoveryMiddleware(logger)
		
		panicErr := errors.New("panic error")
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			panic(panicErr)
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "panic")
		assert.Contains(t, err.Error(), panicErr.Error())
		assert.Nil(t, resp)
	})
	
	t.Run("panic with string", func(t *testing.T) {
		middleware := NewRecoveryMiddleware(logger)
		
		panicMsg := "panic message"
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			panic(panicMsg)
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "panic")
		assert.Contains(t, err.Error(), panicMsg)
		assert.Nil(t, resp)
	})
}

func TestMetricsMiddleware(t *testing.T) {
	logger := slog.Default()
	
	t.Run("collect metrics", func(t *testing.T) {
		middleware := NewMetricsMiddleware(logger)
		
		// Make multiple requests
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "response", nil
		}
		
		for i := 0; i < 5; i++ {
			resp, err := middleware.Process(context.Background(), "request", handler)
			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
		}
		
		// Check metrics
		assert.Equal(t, int64(5), middleware.requestCount["string"])
		assert.Len(t, middleware.requestLatency["string"], 5)
		
		// Check average latency calculation
		avgLatency := middleware.calculateAvgLatency("string")
		assert.Greater(t, avgLatency, time.Duration(0))
	})
}

func TestContextMiddleware(t *testing.T) {
	t.Run("add single value", func(t *testing.T) {
		middleware := NewContextMiddleware()
		middleware.WithValue("key1", "value1")
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			val := ctx.Value("key1")
			assert.Equal(t, "value1", val)
			return "response", nil
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
	
	t.Run("add multiple values", func(t *testing.T) {
		middleware := NewContextMiddleware()
		middleware.WithValue("key1", "value1")
		middleware.WithValue("key2", 42)
		middleware.WithValue("key3", true)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			assert.Equal(t, "value1", ctx.Value("key1"))
			assert.Equal(t, 42, ctx.Value("key2"))
			assert.Equal(t, true, ctx.Value("key3"))
			return "response", nil
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
}

func TestTimeoutMiddleware(t *testing.T) {
	logger := slog.Default()
	
	t.Run("request completes before timeout", func(t *testing.T) {
		middleware := NewTimeoutMiddleware(100*time.Millisecond, logger)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "response", nil
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
	
	t.Run("request times out", func(t *testing.T) {
		middleware := NewTimeoutMiddleware(10*time.Millisecond, logger)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return "response", nil
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		if err == nil {
			// If no error, it should be a timeout response
			assert.NotEqual(t, "response", resp)
		} else {
			// Or we get a context error
			assert.True(t, errors.Is(err, context.DeadlineExceeded))
		}
	})
	
	t.Run("context cancellation", func(t *testing.T) {
		middleware := NewTimeoutMiddleware(100*time.Millisecond, logger)
		
		ctx, cancel := context.WithCancel(context.Background())
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		
		// Cancel context after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		
		_, err := middleware.Process(ctx, "request", handler)
		assert.Error(t, err)
	})
}

func TestMiddlewareIntegration(t *testing.T) {
	logger := slog.Default()
	
	t.Run("complete pipeline", func(t *testing.T) {
		pipeline := NewPipeline(logger)
		
		// Add all middleware
		pipeline.Use(
			NewRecoveryMiddleware(logger),
			NewLoggingMiddleware(logger),
			NewTimeoutMiddleware(1*time.Second, logger),
			NewMetricsMiddleware(logger),
			NewContextMiddleware().WithValue("test", "value"),
		)
		
		callCount := int32(0)
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			atomic.AddInt32(&callCount, 1)
			
			// Verify context value
			assert.Equal(t, "value", ctx.Value("test"))
			
			return map[string]interface{}{
				"request": req,
				"count":   atomic.LoadInt32(&callCount),
			}, nil
		}
		
		// Execute multiple requests
		for i := 0; i < 3; i++ {
			resp, err := pipeline.Execute(context.Background(), i, handler)
			require.NoError(t, err)
			
			respMap, ok := resp.(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, i, respMap["request"])
			assert.Equal(t, int32(i+1), respMap["count"])
		}
	})
	
	t.Run("error propagation", func(t *testing.T) {
		pipeline := NewPipeline(logger)
		
		var executionOrder []string
		
		// Add middleware that tracks execution
		for i := 1; i <= 3; i++ {
			index := i
			middleware := MiddlewareFunc(func(ctx context.Context, req interface{}, next Handler) (interface{}, error) {
				executionOrder = append(executionOrder, fmt.Sprintf("before-%d", index))
				resp, err := next(ctx, req)
				if err != nil {
					executionOrder = append(executionOrder, fmt.Sprintf("error-%d", index))
				} else {
					executionOrder = append(executionOrder, fmt.Sprintf("after-%d", index))
				}
				return resp, err
			})
			pipeline.Use(middleware)
		}
		
		// Handler that returns an error
		expectedErr := errors.New("handler error")
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			executionOrder = append(executionOrder, "handler-error")
			return nil, expectedErr
		}
		
		resp, err := pipeline.Execute(context.Background(), "request", handler)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resp)
		
		// Verify execution order - error should propagate back through middleware
		assert.Equal(t, []string{
			"before-1", "before-2", "before-3",
			"handler-error",
			"error-3", "error-2", "error-1",
		}, executionOrder)
	})
}

// Benchmark tests
func BenchmarkPipeline(b *testing.B) {
	logger := slog.Default()
	
	b.Run("empty pipeline", func(b *testing.B) {
		pipeline := NewPipeline(logger)
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return req, nil
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pipeline.Execute(context.Background(), i, handler)
		}
	})
	
	b.Run("single middleware", func(b *testing.B) {
		pipeline := NewPipeline(logger)
		pipeline.Use(NewLoggingMiddleware(logger))
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return req, nil
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pipeline.Execute(context.Background(), i, handler)
		}
	})
	
	b.Run("multiple middleware", func(b *testing.B) {
		pipeline := NewPipeline(logger)
		pipeline.Use(
			NewRecoveryMiddleware(logger),
			NewLoggingMiddleware(logger),
			NewMetricsMiddleware(logger),
			NewContextMiddleware().WithValue("key", "value"),
		)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return req, nil
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pipeline.Execute(context.Background(), i, handler)
		}
	})
}