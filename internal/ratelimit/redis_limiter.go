// Package ratelimit provides Redis-backed rate limiting with sliding window algorithm
package ratelimit

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter implements Redis-backed rate limiting
type RedisLimiter struct {
	client  *redis.Client
	config  *Config
	scripts *redisScripts
}

// redisScripts contains precompiled Redis Lua scripts
type redisScripts struct {
	slidingWindow *redis.Script
	tokenBucket   *redis.Script
	fixedWindow   *redis.Script
	cleanup       *redis.Script
}

// LimitResult represents the result of a rate limit check
type LimitResult struct {
	Allowed    bool          `json:"allowed"`
	Count      int           `json:"count"`
	Limit      int           `json:"limit"`
	Remaining  int           `json:"remaining"`
	RetryAfter time.Duration `json:"retry_after"`
	ResetTime  time.Time     `json:"reset_time"`
	Algorithm  Algorithm     `json:"algorithm"`
	Key        string        `json:"key"`
	Window     time.Duration `json:"window"`
	Burst      int           `json:"burst"`

	// Additional metadata
	IsFirstRequest bool                   `json:"is_first_request"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewRedisLimiter creates a new Redis-backed rate limiter
func NewRedisLimiter(config *Config) (*RedisLimiter, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:            config.RedisAddr,
		Password:        config.RedisPassword,
		DB:              config.RedisDB,
		MaxRetries:      config.MaxRetries,
		MinRetryBackoff: config.MinRetryBackoff,
		MaxRetryBackoff: config.MaxRetryBackoff,
		DialTimeout:     config.DialTimeout,
		ReadTimeout:     config.ReadTimeout,
		WriteTimeout:    config.WriteTimeout,
		PoolSize:        config.PoolSize,
		MinIdleConns:    config.MinIdleConns,
		MaxIdleConns:    config.MaxIdleConns,
		ConnMaxLifetime: config.ConnMaxLifetime,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	limiter := &RedisLimiter{
		client: rdb,
		config: config,
		scripts: &redisScripts{
			slidingWindow: redis.NewScript(slidingWindowScript),
			tokenBucket:   redis.NewScript(tokenBucketScript),
			fixedWindow:   redis.NewScript(fixedWindowScript),
			cleanup:       redis.NewScript(cleanupScript),
		},
	}

	log.Printf("Connected to Redis at %s for rate limiting", config.RedisAddr)

	return limiter, nil
}

// Check performs a rate limit check for the given key
func (rl *RedisLimiter) Check(ctx context.Context, key string, limit *EndpointLimit) (*LimitResult, error) {
	if limit == nil {
		return nil, fmt.Errorf("endpoint limit configuration is required")
	}

	// Build the full key with prefix
	fullKey := rl.buildKey(key)

	// Execute the appropriate algorithm
	switch limit.Algorithm {
	case AlgorithmSlidingWindow:
		return rl.checkSlidingWindow(ctx, fullKey, limit)
	case AlgorithmTokenBucket:
		return rl.checkTokenBucket(ctx, fullKey, limit)
	case AlgorithmFixedWindow:
		return rl.checkFixedWindow(ctx, fullKey, limit)
	case AlgorithmLeakyBucket:
		return rl.checkLeakyBucket(ctx, fullKey, limit)
	default:
		return rl.checkSlidingWindow(ctx, fullKey, limit)
	}
}

// CheckMultiple performs rate limit checks for multiple keys
func (rl *RedisLimiter) CheckMultiple(ctx context.Context, keys []string, limits []*EndpointLimit) ([]*LimitResult, error) {
	if len(keys) != len(limits) {
		return nil, fmt.Errorf("keys and limits slices must have the same length")
	}

	results := make([]*LimitResult, len(keys))

	// Use pipeline for efficiency
	pipe := rl.client.Pipeline()
	var commands []*redis.Cmd

	for i, key := range keys {
		fullKey := rl.buildKey(key)
		limit := limits[i]

		if limit == nil {
			results[i] = &LimitResult{
				Allowed: false,
				Key:     key,
			}
			continue
		}

		// Add command to pipeline based on algorithm
		var cmd *redis.Cmd
		switch limit.Algorithm {
		case AlgorithmSlidingWindow:
			cmd = pipe.EvalSha(ctx, rl.scripts.slidingWindow.Hash(), []string{fullKey},
				limit.Limit, limit.Window.Milliseconds(), time.Now().UnixMilli(), limit.Burst)
		case AlgorithmTokenBucket:
			cmd = pipe.EvalSha(ctx, rl.scripts.tokenBucket.Hash(), []string{fullKey},
				limit.Limit, limit.Window.Milliseconds(), time.Now().UnixMilli(), limit.Burst)
		case AlgorithmFixedWindow:
			cmd = pipe.EvalSha(ctx, rl.scripts.fixedWindow.Hash(), []string{fullKey},
				limit.Limit, limit.Window.Milliseconds(), time.Now().UnixMilli())
		default:
			cmd = pipe.EvalSha(ctx, rl.scripts.slidingWindow.Hash(), []string{fullKey},
				limit.Limit, limit.Window.Milliseconds(), time.Now().UnixMilli(), limit.Burst)
		}
		commands = append(commands, cmd)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	// Process results
	cmdIndex := 0
	for i, key := range keys {
		limit := limits[i]
		if limit == nil {
			continue
		}

		cmd := commands[cmdIndex]
		cmdIndex++

		result, err := rl.parseScriptResult(cmd.Val(), key, limit)
		if err != nil {
			result = &LimitResult{
				Allowed: false,
				Key:     key,
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
		}
		results[i] = result
	}

	return results, nil
}

// checkSlidingWindow implements sliding window rate limiting
func (rl *RedisLimiter) checkSlidingWindow(ctx context.Context, key string, limit *EndpointLimit) (*LimitResult, error) {
	now := time.Now().UnixMilli()

	// Ensure script is loaded
	if err := rl.loadScript(ctx, rl.scripts.slidingWindow); err != nil {
		return nil, fmt.Errorf("failed to load sliding window script: %w", err)
	}

	// Execute script
	result, err := rl.scripts.slidingWindow.Run(ctx, rl.client, []string{key},
		limit.Limit, limit.Window.Milliseconds(), now, limit.Burst).Result()
	if err != nil {
		return nil, fmt.Errorf("sliding window script failed: %w", err)
	}

	return rl.parseScriptResult(result, key, limit)
}

// checkTokenBucket implements token bucket rate limiting
func (rl *RedisLimiter) checkTokenBucket(ctx context.Context, key string, limit *EndpointLimit) (*LimitResult, error) {
	now := time.Now().UnixMilli()

	// Ensure script is loaded
	if err := rl.loadScript(ctx, rl.scripts.tokenBucket); err != nil {
		return nil, fmt.Errorf("failed to load token bucket script: %w", err)
	}

	// Execute script
	result, err := rl.scripts.tokenBucket.Run(ctx, rl.client, []string{key},
		limit.Limit, limit.Window.Milliseconds(), now, limit.Burst).Result()
	if err != nil {
		return nil, fmt.Errorf("token bucket script failed: %w", err)
	}

	return rl.parseScriptResult(result, key, limit)
}

// checkFixedWindow implements fixed window rate limiting
func (rl *RedisLimiter) checkFixedWindow(ctx context.Context, key string, limit *EndpointLimit) (*LimitResult, error) {
	now := time.Now().UnixMilli()

	// Ensure script is loaded
	if err := rl.loadScript(ctx, rl.scripts.fixedWindow); err != nil {
		return nil, fmt.Errorf("failed to load fixed window script: %w", err)
	}

	// Execute script
	result, err := rl.scripts.fixedWindow.Run(ctx, rl.client, []string{key},
		limit.Limit, limit.Window.Milliseconds(), now).Result()
	if err != nil {
		return nil, fmt.Errorf("fixed window script failed: %w", err)
	}

	return rl.parseScriptResult(result, key, limit)
}

// checkLeakyBucket implements leaky bucket rate limiting
func (rl *RedisLimiter) checkLeakyBucket(ctx context.Context, key string, limit *EndpointLimit) (*LimitResult, error) {
	// For simplicity, implement leaky bucket as a special case of token bucket
	return rl.checkTokenBucket(ctx, key, limit)
}

// parseScriptResult parses the result from a Redis Lua script
func (rl *RedisLimiter) parseScriptResult(result interface{}, key string, limit *EndpointLimit) (*LimitResult, error) {
	values, ok := result.([]interface{})
	if !ok || len(values) < 4 {
		return nil, fmt.Errorf("invalid script result format")
	}

	// Parse values from script result
	allowed, err := strconv.ParseBool(fmt.Sprintf("%v", values[0]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse allowed: %w", err)
	}

	count, err := strconv.Atoi(fmt.Sprintf("%v", values[1]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse count: %w", err)
	}

	remaining, err := strconv.Atoi(fmt.Sprintf("%v", values[2]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse remaining: %w", err)
	}

	resetTimeMs, err := strconv.ParseInt(fmt.Sprintf("%v", values[3]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reset time: %w", err)
	}

	resetTime := time.Unix(0, resetTimeMs*int64(time.Millisecond))
	retryAfter := time.Until(resetTime)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return &LimitResult{
		Allowed:        allowed,
		Count:          count,
		Limit:          limit.Limit,
		Remaining:      remaining,
		RetryAfter:     retryAfter,
		ResetTime:      resetTime,
		Algorithm:      limit.Algorithm,
		Key:            key,
		Window:         limit.Window,
		Burst:          limit.Burst,
		IsFirstRequest: count == 1,
		Metadata:       make(map[string]interface{}),
	}, nil
}

// Reset resets the rate limit for a given key
func (rl *RedisLimiter) Reset(ctx context.Context, key string) error {
	fullKey := rl.buildKey(key)
	return rl.client.Del(ctx, fullKey).Err()
}

// ResetMultiple resets the rate limits for multiple keys
func (rl *RedisLimiter) ResetMultiple(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = rl.buildKey(key)
	}

	return rl.client.Del(ctx, fullKeys...).Err()
}

// GetStats returns current statistics for a key
func (rl *RedisLimiter) GetStats(ctx context.Context, key string) (map[string]interface{}, error) {
	fullKey := rl.buildKey(key)

	// Get current value and TTL
	pipe := rl.client.Pipeline()
	getCmd := pipe.Get(ctx, fullKey)
	ttlCmd := pipe.TTL(ctx, fullKey)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := make(map[string]interface{})
	stats["key"] = key
	stats["full_key"] = fullKey

	if getCmd.Err() == nil {
		stats["current_count"] = getCmd.Val()
	} else {
		stats["current_count"] = 0
	}

	if ttlCmd.Err() == nil {
		stats["ttl_seconds"] = ttlCmd.Val().Seconds()
	} else {
		stats["ttl_seconds"] = 0
	}

	return stats, nil
}

// Cleanup removes expired rate limit keys
func (rl *RedisLimiter) Cleanup(ctx context.Context) error {
	// Load and run cleanup script
	if err := rl.loadScript(ctx, rl.scripts.cleanup); err != nil {
		return fmt.Errorf("failed to load cleanup script: %w", err)
	}

	result, err := rl.scripts.cleanup.Run(ctx, rl.client, []string{},
		rl.config.KeyPrefix, 1000).Result() // Clean up to 1000 keys at a time
	if err != nil {
		return fmt.Errorf("cleanup script failed: %w", err)
	}

	cleanedCount, ok := result.(int64)
	if ok && cleanedCount > 0 {
		log.Printf("Cleaned up %d expired rate limit keys", cleanedCount)
	}

	return nil
}

// Close closes the Redis connection
func (rl *RedisLimiter) Close() error {
	return rl.client.Close()
}

// IsHealthy checks if the Redis connection is healthy
func (rl *RedisLimiter) IsHealthy(ctx context.Context) error {
	return rl.client.Ping(ctx).Err()
}

// GetInfo returns information about the Redis connection
func (rl *RedisLimiter) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Get Redis info
	redisInfo, err := rl.client.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	info["redis_info"] = redisInfo
	info["config"] = map[string]interface{}{
		"redis_addr":     rl.config.RedisAddr,
		"redis_db":       rl.config.RedisDB,
		"pool_size":      rl.config.PoolSize,
		"key_prefix":     rl.config.KeyPrefix,
		"default_limit":  rl.config.DefaultLimit,
		"default_window": rl.config.DefaultWindow.String(),
	}

	return info, nil
}

// buildKey builds a full Redis key with prefix
func (rl *RedisLimiter) buildKey(key string) string {
	return rl.config.KeyPrefix + key
}

// loadScript loads a Lua script into Redis if not already loaded
func (rl *RedisLimiter) loadScript(ctx context.Context, script *redis.Script) error {
	exists, err := rl.client.ScriptExists(ctx, script.Hash()).Result()
	if err != nil {
		return err
	}

	if len(exists) == 0 || !exists[0] {
		_, err := script.Load(ctx, rl.client).Result()
		return err
	}

	return nil
}

// Lua scripts for different rate limiting algorithms

const slidingWindowScript = `
-- Sliding window rate limiting
-- KEYS[1]: rate limit key
-- ARGV[1]: limit
-- ARGV[2]: window in milliseconds  
-- ARGV[3]: current time in milliseconds
-- ARGV[4]: burst allowance

local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local burst = tonumber(ARGV[4]) or 0

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Get current count
local current = redis.call('ZCARD', key)

-- Check if request is allowed
local allowed = current < limit
local actualLimit = limit + burst

if current < actualLimit then
    allowed = true
end

if allowed then
    -- Add current request
    redis.call('ZADD', key, now, now .. ':' .. math.random())
    current = current + 1
    -- Set expiration
    redis.call('EXPIRE', key, math.ceil(window / 1000))
end

local remaining = math.max(0, limit - current)
local resetTime = now + window

return {allowed, current, remaining, resetTime}
`

const tokenBucketScript = `
-- Token bucket rate limiting
-- KEYS[1]: rate limit key
-- ARGV[1]: capacity (limit)
-- ARGV[2]: refill time in milliseconds
-- ARGV[3]: current time in milliseconds
-- ARGV[4]: burst allowance

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refillTime = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local burst = tonumber(ARGV[4]) or 0

-- Get current bucket state
local bucket = redis.call('HMGET', key, 'tokens', 'lastRefill')
local tokens = tonumber(bucket[1]) or capacity
local lastRefill = tonumber(bucket[2]) or now

-- Calculate tokens to add based on time passed
local timePassed = now - lastRefill
local tokensToAdd = math.floor(timePassed / refillTime * capacity)
tokens = math.min(capacity, tokens + tokensToAdd)

-- Check if request is allowed
local allowed = tokens >= 1
local actualCapacity = capacity + burst

if tokens < 1 and tokens + burst >= 1 then
    allowed = true
end

if allowed then
    tokens = tokens - 1
    -- Update bucket state
    redis.call('HMSET', key, 'tokens', tokens, 'lastRefill', now)
    redis.call('EXPIRE', key, math.ceil(refillTime / 1000) * 2)
end

local remaining = math.max(0, tokens)
local resetTime = now + (1 - tokens) * (refillTime / capacity)

return {allowed, capacity - tokens, remaining, resetTime}
`

const fixedWindowScript = `
-- Fixed window rate limiting
-- KEYS[1]: rate limit key
-- ARGV[1]: limit
-- ARGV[2]: window in milliseconds
-- ARGV[3]: current time in milliseconds

local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Calculate window start
local windowStart = math.floor(now / window) * window
local windowKey = key .. ':' .. windowStart

-- Get current count
local current = tonumber(redis.call('GET', windowKey)) or 0

-- Check if request is allowed
local allowed = current < limit

if allowed then
    -- Increment counter
    current = redis.call('INCR', windowKey)
    -- Set expiration for window
    redis.call('EXPIRE', windowKey, math.ceil(window / 1000))
end

local remaining = math.max(0, limit - current)
local resetTime = windowStart + window

return {allowed, current, remaining, resetTime}
`

const cleanupScript = `
-- Cleanup expired rate limit keys
-- ARGV[1]: key prefix
-- ARGV[2]: batch size

local prefix = ARGV[1]
local batchSize = tonumber(ARGV[2]) or 100

local cursor = 0
local count = 0

repeat
    local result = redis.call('SCAN', cursor, 'MATCH', prefix .. '*', 'COUNT', batchSize)
    cursor = tonumber(result[1])
    local keys = result[2]
    
    for i = 1, #keys do
        local ttl = redis.call('TTL', keys[i])
        if ttl == -1 then  -- No expiration set
            redis.call('DEL', keys[i])
            count = count + 1
        end
    end
until cursor == 0

return count
`
