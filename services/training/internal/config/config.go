package config

import (
	"os"
	"time"
)

// Config 训练服务配置
type Config struct {
	Environment string
	Port        string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Logging
	LogLevel  string
	LogFormat string // json or console
	LogOutput string // stdout, file, or both
	LogPath   string

	// Docker
	DockerHost        string
	DockerNetwork     string
	DockerVolumeBase  string

	// MinIO (模型存储)
	MinIOEndpoint     string
	MinIOAccessKey    string
	MinIOSecretKey    string
	MinIOBucket       string
	MinIOUseSSL       bool

	// 训练配置
	DefaultTimeout    time.Duration
	MaxConcurrentJobs int
	LogStreamMaxLen   int64  // Redis Stream 最大长度
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnv("PORT", "8081"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "console"),
		LogOutput: getEnv("LOG_OUTPUT", "stdout"),
		LogPath:   getEnv("LOG_PATH", "logs/training.log"),

		DockerHost:       getEnv("DOCKER_HOST", ""),
		DockerNetwork:    getEnv("DOCKER_NETWORK", "aitip-network"),
		DockerVolumeBase: getEnv("DOCKER_VOLUME_BASE", "/var/aitip/training"),

		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "models"),
		MinIOUseSSL:    getEnvBool("MINIO_USE_SSL", false),

		DefaultTimeout:    parseDuration(getEnv("DEFAULT_TIMEOUT", "24h")),
		MaxConcurrentJobs: parseInt(getEnv("MAX_CONCURRENT_JOBS", "5")),
		LogStreamMaxLen:   parseInt64(getEnv("LOG_STREAM_MAX_LEN", "10000")),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func parseInt(value string) int {
	var result int
	for _, c := range value {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			break
		}
	}
	if result == 0 {
		return 10
	}
	return result
}

func parseInt64(value string) int64 {
	var result int64
	for _, c := range value {
		if c >= '0' && c <= '9' {
			result = result*10 + int64(c-'0')
		} else {
			break
		}
	}
	if result == 0 {
		return 10000
	}
	return result
}

func parseDuration(value string) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return time.Hour * 24
	}
	return d
}
