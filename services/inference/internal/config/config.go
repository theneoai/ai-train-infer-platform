package config

import (
	"os"
	"time"
)

// Config 推理服务配置
type Config struct {
	Environment string
	Port        string

	// Database
	DatabaseURL string

	// Logging
	LogLevel  string
	LogFormat string // json or console
	LogOutput string // stdout, file, or both
	LogPath   string

	// Docker
	DockerHost       string
	DockerNetwork    string
	ModelCachePath   string

	// MinIO (模型存储)
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool

	// 推理服务配置
	DefaultInferencePort int
	MaxConcurrentServices int
	HealthCheckInterval  time.Duration
	ModelDownloadTimeout time.Duration
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnv("PORT", "8082"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable"),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "console"),
		LogOutput: getEnv("LOG_OUTPUT", "stdout"),
		LogPath:   getEnv("LOG_PATH", "logs/inference.log"),

		DockerHost:       getEnv("DOCKER_HOST", ""),
		DockerNetwork:    getEnv("DOCKER_NETWORK", "aitip-network"),
		ModelCachePath:   getEnv("MODEL_CACHE_PATH", "/var/aitip/models"),

		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "models"),
		MinIOUseSSL:    getEnvBool("MINIO_USE_SSL", false),

		DefaultInferencePort:  getEnvInt("DEFAULT_INFERENCE_PORT", 8000),
		MaxConcurrentServices: getEnvInt("MAX_CONCURRENT_SERVICES", "10"),
		HealthCheckInterval:   parseDuration(getEnv("HEALTH_CHECK_INTERVAL", "30s")),
		ModelDownloadTimeout:  parseDuration(getEnv("MODEL_DOWNLOAD_TIMEOUT", "10m")),
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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		for _, c := range value {
			if c >= '0' && c <= '9' {
				result = result*10 + int(c-'0')
			} else {
				break
			}
		}
		if result > 0 {
			return result
		}
	}
	return defaultValue
}

func parseDuration(value string) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return time.Minute * 5
	}
	return d
}
