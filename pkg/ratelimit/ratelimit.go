package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/plucky-groove3/ai-train-infer-platform/pkg/redis"
)

// Limiter 限流器接口
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	AllowN(ctx context.Context, key string, n int) (bool, error)
	Reset(ctx context.Context, key string) error
}

// Config 限流配置
type Config struct {
	Rate       int           // 每秒请求数
	Burst      int           // 突发请求数
	WindowSize time.Duration // 窗口大小
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Rate:       100,
		Burst:      150,
		WindowSize: time.Second,
	}
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	redis  *redis.Client
	config *Config
	prefix string
}

// NewTokenBucket 创建令牌桶限流器
func NewTokenBucket(rdb *redis.Client, config *Config, prefix string) *TokenBucket {
	if config == nil {
		config = DefaultConfig()
	}
	if prefix == "" {
		prefix = "ratelimit"
	}
	return &TokenBucket{
		redis:  rdb,
		config: config,
		prefix: prefix,
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	return tb.AllowN(ctx, key, 1)
}

// AllowN 检查是否允许 N 个请求
func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int) (bool, error) {
	fullKey := fmt.Sprintf("%s:%s", tb.prefix, key)

	// 使用 Redis Lua 脚本实现令牌桶算法
	script := `
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])
		local ttl = math.ceil((burst / rate) * 2)
		
		local bucket = redis.call('HMGET', key, 'tokens', 'last_updated')
		local tokens = tonumber(bucket[1]) or burst
		local last_updated = tonumber(bucket[2]) or now
		
		local elapsed = math.max(0, now - last_updated)
		tokens = math.min(burst, tokens + (elapsed * rate))
		
		local allowed = tokens >= requested
		local new_tokens = tokens
		if allowed then
			new_tokens = tokens - requested
		end
		
		redis.call('HMSET', key, 'tokens', new_tokens, 'last_updated', now)
		redis.call('EXPIRE', key, ttl)
		
		return allowed and 1 or 0
	`

	now := time.Now().Unix()
	result, err := tb.redis.GetClient().Eval(ctx, script, []string{fullKey},
		float64(tb.config.Rate),
		float64(tb.config.Burst),
		float64(now),
		n,
	).Result()

	if err != nil {
		return false, err
	}

	allowed, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type from redis script")
	}

	return allowed == 1, nil
}

// Reset 重置限流器
func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	fullKey := fmt.Sprintf("%s:%s", tb.prefix, key)
	return tb.redis.Del(ctx, fullKey)
}

// SlidingWindow 滑动窗口限流器
type SlidingWindow struct {
	redis      *redis.Client
	windowSize time.Duration
	maxRequest int
	prefix     string
}

// NewSlidingWindow 创建滑动窗口限流器
func NewSlidingWindow(rdb *redis.Client, windowSize time.Duration, maxRequest int, prefix string) *SlidingWindow {
	if prefix == "" {
		prefix = "sliding"
	}
	return &SlidingWindow{
		redis:      rdb,
		windowSize: windowSize,
		maxRequest: maxRequest,
		prefix:     prefix,
	}
}

// Allow 检查是否允许请求
func (sw *SlidingWindow) Allow(ctx context.Context, key string) (bool, error) {
	return sw.AllowN(ctx, key, 1)
}

// AllowN 检查是否允许 N 个请求
func (sw *SlidingWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	fullKey := fmt.Sprintf("%s:%s", sw.prefix, key)
	now := time.Now().UnixMilli()
	windowStart := now - sw.windowSize.Milliseconds()

	// 使用 Redis 有序集合实现滑动窗口
	pipe := sw.redis.GetClient().Pipeline()

	// 移除窗口外的请求记录
	pipe.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", windowStart))

	// 获取当前窗口内的请求数
	pipe.ZCard(ctx, fullKey)

	// 添加当前请求
	for i := 0; i < n; i++ {
		pipe.ZAdd(ctx, fullKey, redis.ZAddArgs{
			Members: []redis.Z{
				{Score: float64(now + int64(i)), Member: fmt.Sprintf("%d-%d", now, i)},
			},
		})
	}

	// 设置过期时间
	pipe.Expire(ctx, fullKey, sw.windowSize)

	results, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	// 获取当前请求数
	currentCount := results[1].(*redis.IntCmd).Val()

	return int(currentCount)+n <= sw.maxRequest, nil
}

// Reset 重置限流器
func (sw *SlidingWindow) Reset(ctx context.Context, key string) error {
	fullKey := fmt.Sprintf("%s:%s", sw.prefix, key)
	return sw.redis.Del(ctx, fullKey)
}

// FixedWindow 固定窗口限流器
type FixedWindow struct {
	redis      *redis.Client
	windowSize time.Duration
	maxRequest int
	prefix     string
}

// NewFixedWindow 创建固定窗口限流器
func NewFixedWindow(rdb *redis.Client, windowSize time.Duration, maxRequest int, prefix string) *FixedWindow {
	if prefix == "" {
		prefix = "fixed"
	}
	return &FixedWindow{
		redis:      rdb,
		windowSize: windowSize,
		maxRequest: maxRequest,
		prefix:     prefix,
	}
}

// Allow 检查是否允许请求
func (fw *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	return fw.AllowN(ctx, key, 1)
}

// AllowN 检查是否允许 N 个请求
func (fw *FixedWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	window := time.Now().Unix() / int64(fw.windowSize.Seconds())
	fullKey := fmt.Sprintf("%s:%s:%d", fw.prefix, key, window)

	count, err := fw.redis.Incr(ctx, fullKey)
	if err != nil {
		return false, err
	}

	// 如果是第一次请求，设置过期时间
	if count == 1 {
		fw.redis.Expire(ctx, fullKey, fw.windowSize)
	}

	return int(count) <= fw.maxRequest, nil
}

// Reset 重置限流器
func (fw *FixedWindow) Reset(ctx context.Context, key string) error {
	window := time.Now().Unix() / int64(fw.windowSize.Seconds())
	fullKey := fmt.Sprintf("%s:%s:%d", fw.prefix, key, window)
	return fw.redis.Del(ctx, fullKey)
}
