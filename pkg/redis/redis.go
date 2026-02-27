package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client Redis 客户端包装
type Client struct {
	client *redis.Client
}

// Config Redis 配置
type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
		PoolSize: 10,
	}
}

// New 创建 Redis 客户端
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{client: rdb}, nil
}

// NewFromURL 从 URL 创建 Redis 客户端
func NewFromURL(redisURL string) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{client: rdb}, nil
}

// GetClient 获取原始 Redis 客户端
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// Get 获取键值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Set 设置键值
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// SetNX 仅当键不存在时才设置
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, expiration).Result()
}

// Del 删除键
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Exists(ctx, keys...).Result()
}

// Expire 设置过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return c.client.Expire(ctx, key, expiration).Result()
}

// TTL 获取剩余过期时间
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Incr 自增
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Decr 自减
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

// HGet 获取哈希字段
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

// HSet 设置哈希字段
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.HSet(ctx, key, values...).Err()
}

// HGetAll 获取所有哈希字段
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// HDel 删除哈希字段
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.HDel(ctx, key, fields...).Err()
}

// LPush 列表左侧推送
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.LPush(ctx, key, values...).Err()
}

// RPush 列表右侧推送
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.RPush(ctx, key, values...).Err()
}

// LPop 列表左侧弹出
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	return c.client.LPop(ctx, key).Result()
}

// RPop 列表右侧弹出
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.client.RPop(ctx, key).Result()
}

// BLPop 阻塞式列表左侧弹出
func (c *Client) BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	return c.client.BLPop(ctx, timeout, keys...).Result()
}

// BRPop 阻塞式列表右侧弹出
func (c *Client) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	return c.client.BRPop(ctx, timeout, keys...).Result()
}

// LRange 获取列表范围
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

// LLen 获取列表长度
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.client.LLen(ctx, key).Result()
}

// SAdd 集合添加
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, key, members...).Err()
}

// SRem 集合移除
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SRem(ctx, key, members...).Err()
}

// SIsMember 检查是否为集合成员
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.client.SIsMember(ctx, key, member).Result()
}

// SMembers 获取所有集合成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, key).Result()
}

// ZAdd 有序集合添加
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return c.client.ZAdd(ctx, key, members...).Err()
}

// ZRange 有序集合范围
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.ZRange(ctx, key, start, stop).Result()
}

// ZRangeByScore 有序集合分数范围
func (c *Client) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return c.client.ZRangeByScore(ctx, key, opt).Result()
}

// ZRem 有序集合移除
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.ZRem(ctx, key, members...).Err()
}

// Publish 发布消息
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.client.Publish(ctx, channel, message).Err()
}

// Subscribe 订阅频道
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

// XAdd 添加流消息
func (c *Client) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	return c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
}

// XRange 流范围查询
func (c *Client) XRange(ctx context.Context, stream, start, stop string) ([]redis.XMessage, error) {
	return c.client.XRange(ctx, stream, start, stop).Result()
}

// XRead 读取流
func (c *Client) XRead(ctx context.Context, streams []string, count int64, block time.Duration) ([]redis.XStream, error) {
	return c.client.XRead(ctx, &redis.XReadArgs{
		Streams: streams,
		Count:   count,
		Block:   block,
	}).Result()
}

// XGroupCreate 创建消费组
func (c *Client) XGroupCreate(ctx context.Context, stream, group, start string) error {
	return c.client.XGroupCreate(ctx, stream, group, start).Err()
}

// XReadGroup 消费组读取
func (c *Client) XReadGroup(ctx context.Context, group, consumer string, streams []string, count int64, block time.Duration) ([]redis.XStream, error) {
	return c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  streams,
		Count:    count,
		Block:    block,
	}).Result()
}

// XAck 确认消息
func (c *Client) XAck(ctx context.Context, stream, group string, ids ...string) error {
	return c.client.XAck(ctx, stream, group, ids...).Err()
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.client.Close()
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
