package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Environment string
	Port        string
	
	// Database
	DatabaseURL string
	
	// Redis
	RedisURL string
	
	// JWT
	JWTSecretKey       string
	JWTIssuer          string
	JWTAccessTokenTTL  time.Duration
	JWTRefreshTokenTTL time.Duration
	
	// Rate Limiting
	RateLimitRate   int
	RateLimitBurst  int
	RateLimitWindow time.Duration
	
	// Logging
	LogLevel  string
	LogFormat string // json or console
	LogOutput string // stdout, file, or both
	
	// External Services
	TrainingServiceURL   string
	ExperimentServiceURL string
	InferenceServiceURL  string
	
	// Kubernetes/Ray
	KubeConfig    string
	RayClusterURL string
}

func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnv("PORT", "8080"),
		
		DatabaseURL: getEnv("DATABASE_URL", "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		
		JWTSecretKey:       getEnv("JWT_SECRET_KEY", "your-secret-key-change-in-production"),
		JWTIssuer:          getEnv("JWT_ISSUER", "aitip-gateway"),
		JWTAccessTokenTTL:  parseDuration(getEnv("JWT_ACCESS_TOKEN_TTL", "2h")),
		JWTRefreshTokenTTL: parseDuration(getEnv("JWT_REFRESH_TOKEN_TTL", "168h")), // 7 days
		
		RateLimitRate:   parseInt(getEnv("RATE_LIMIT_RATE", "100")),
		RateLimitBurst:  parseInt(getEnv("RATE_LIMIT_BURST", "150")),
		RateLimitWindow: parseDuration(getEnv("RATE_LIMIT_WINDOW", "1m")),
		
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "console"),
		LogOutput: getEnv("LOG_OUTPUT", "stdout"),
		
		TrainingServiceURL:   getEnv("TRAINING_SERVICE_URL", "http://localhost:8081"),
		ExperimentServiceURL: getEnv("EXPERIMENT_SERVICE_URL", "http://localhost:8082"),
		InferenceServiceURL:  getEnv("INFERENCE_SERVICE_URL", "http://localhost:8083"),
		
		KubeConfig:    getEnv("KUBE_CONFIG", ""),
		RayClusterURL: getEnv("RAY_CLUSTER_URL", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(value string) int {
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	if err != nil {
		return 100
	}
	return result
}

func parseDuration(value string) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return time.Hour * 2
	}
	return d
}
