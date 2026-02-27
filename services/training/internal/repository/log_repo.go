package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/domain"
	"github.com/redis/go-redis/v9"
)

// LogRepository 日志仓库接口
type LogRepository interface {
	// Redis Stream 操作
	AppendLog(ctx context.Context, jobID uuid.UUID, entry *domain.LogEntry) error
	ReadLogs(ctx context.Context, jobID uuid.UUID, start string, count int64) ([]domain.LogEntry, error)
	ReadLogsRealtime(ctx context.Context, jobID uuid.UUID, block time.Duration) ([]domain.LogEntry, error)
	GetLogStreamLength(ctx context.Context, jobID uuid.UUID) (int64, error)
	TrimLogStream(ctx context.Context, jobID uuid.UUID, maxLen int64) error
	
	// 文件存储（用于持久化）
	SaveLogToFile(ctx context.Context, jobID uuid.UUID, content string) error
	GetLogFromFile(ctx context.Context, jobID uuid.UUID) (string, error)
}

// logRepository 日志仓库实现
type logRepository struct {
	redis      *redis.Client
	streamPrefix string
	maxLen     int64
}

// NewLogRepository 创建日志仓库实例
func NewLogRepository(redisClient *redis.Client, maxLen int64) LogRepository {
	return &logRepository{
		redis:        redisClient,
		streamPrefix: "training:logs:",
		maxLen:       maxLen,
	}
}

// getStreamKey 获取流键名
func (r *logRepository) getStreamKey(jobID uuid.UUID) string {
	return fmt.Sprintf("%s%s", r.streamPrefix, jobID.String())
}

// AppendLog 追加日志到 Redis Stream
func (r *logRepository) AppendLog(ctx context.Context, jobID uuid.UUID, entry *domain.LogEntry) error {
	streamKey := r.getStreamKey(jobID)
	
	values := map[string]interface{}{
		"level":     entry.Level,
		"source":    entry.Source,
		"message":   entry.Message,
		"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
	}

	_, err := r.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: values,
		MaxLen: r.maxLen,
	}).Result()

	return err
}

// ReadLogs 读取日志（从指定位置）
func (r *logRepository) ReadLogs(ctx context.Context, jobID uuid.UUID, start string, count int64) ([]domain.LogEntry, error) {
	if start == "" {
		start = "0"
	}
	if count == 0 {
		count = 100
	}

	streamKey := r.getStreamKey(jobID)
	messages, err := r.redis.XRange(ctx, streamKey, start, "+").Result()
	if err != nil {
		return nil, err
	}

	var logs []domain.LogEntry
	for i, msg := range messages {
		if int64(i) >= count {
			break
		}
		
		log := parseLogEntry(msg)
		logs = append(logs, log)
	}

	return logs, nil
}

// ReadLogsRealtime 实时读取日志（阻塞式）
func (r *logRepository) ReadLogsRealtime(ctx context.Context, jobID uuid.UUID, block time.Duration) ([]domain.LogEntry, error) {
	streamKey := r.getStreamKey(jobID)
	
	// 使用 $ 表示只读取新消息
	streams, err := r.redis.XRead(ctx, &redis.XReadArgs{
		Streams: []string{streamKey, "$"},
		Count:   100,
		Block:   block,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return []domain.LogEntry{}, nil
		}
		return nil, err
	}

	var logs []domain.LogEntry
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			log := parseLogEntry(msg)
			logs = append(logs, log)
		}
	}

	return logs, nil
}

// GetLogStreamLength 获取日志流长度
func (r *logRepository) GetLogStreamLength(ctx context.Context, jobID uuid.UUID) (int64, error) {
	streamKey := r.getStreamKey(jobID)
	return r.redis.XLen(ctx, streamKey).Result()
}

// TrimLogStream 裁剪日志流
func (r *logRepository) TrimLogStream(ctx context.Context, jobID uuid.UUID, maxLen int64) error {
	streamKey := r.getStreamKey(jobID)
	return r.redis.XTrimMaxLen(ctx, streamKey, maxLen).Err()
}

// SaveLogToFile 保存日志到文件（持久化）
func (r *logRepository) SaveLogToFile(ctx context.Context, jobID uuid.UUID, content string) error {
	// 暂时使用 Redis 存储完整日志，后续可以改为文件存储
	key := fmt.Sprintf("training:log:file:%s", jobID.String())
	return r.redis.Set(ctx, key, content, 7*24*time.Hour).Err()
}

// GetLogFromFile 从文件获取日志
func (r *logRepository) GetLogFromFile(ctx context.Context, jobID uuid.UUID) (string, error) {
	key := fmt.Sprintf("training:log:file:%s", jobID.String())
	return r.redis.Get(ctx, key).Result()
}

// parseLogEntry 解析日志条目
func parseLogEntry(msg redis.XMessage) domain.LogEntry {
	entry := domain.LogEntry{
		Level:  getString(msg.Values, "level"),
		Source: getString(msg.Values, "source"),
		Message: getString(msg.Values, "message"),
	}

	// 解析时间戳
	if tsStr := getString(msg.Values, "timestamp"); tsStr != "" {
		if ts, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
			entry.Timestamp = ts
		} else {
			entry.Timestamp = time.Now()
		}
	} else {
		entry.Timestamp = time.Now()
	}

	return entry
}

// getString 从 map 中获取字符串值
func getString(values map[string]interface{}, key string) string {
	if val, ok := values[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
