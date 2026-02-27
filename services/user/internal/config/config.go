package config

import (
	"os"
	"strconv"
	"time"

	"github.com/ai-train-infer-platform/pkg/database"
	"github.com/ai-train-infer-platform/pkg/jwt"
	"github.com/ai-train-infer-platform/pkg/logger"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig
	Database database.Config
	JWT      jwt.Config
	Logger   logger.Config
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Server:   loadServerConfig(),
		Database: loadDatabaseConfig(),
		JWT:      loadJWTConfig(),
		Logger:   loadLoggerConfig(),
	}
}

func loadServerConfig() ServerConfig {
	port := getEnv("SERVER_PORT", "8080")
	if port[0] == ':' {
		port = port[1:]
	}

	readTimeout, _ := strconv.Atoi(getEnv("SERVER_READ_TIMEOUT", "30"))
	writeTimeout, _ := strconv.Atoi(getEnv("SERVER_WRITE_TIMEOUT", "30"))

	return ServerConfig{
		Port:         ":" + port,
		Environment:  getEnv("ENVIRONMENT", "development"),
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
	}
}

func loadDatabaseConfig() database.Config {
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10"))
	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "100"))
	connMaxLifetime, _ := strconv.Atoi(getEnv("DB_CONN_MAX_LIFETIME", "3600"))

	return database.Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5432"),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "password"),
		Database:        getEnv("DB_NAME", "aitip"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxIdleConns:    maxIdleConns,
		MaxOpenConns:    maxOpenConns,
		ConnMaxLifetime: time.Duration(connMaxLifetime) * time.Second,
		LogLevel:        getDatabaseLogLevel(getEnv("DB_LOG_LEVEL", "info")),
	}
}

func loadJWTConfig() jwt.Config {
	accessTokenTTL, _ := strconv.Atoi(getEnv("JWT_ACCESS_TOKEN_TTL", "7200"))      // 2 hours in seconds
	refreshTokenTTL, _ := strconv.Atoi(getEnv("JWT_REFRESH_TOKEN_TTL", "604800")) // 7 days in seconds

	return jwt.Config{
		SecretKey:       getEnv("JWT_SECRET_KEY", "your-secret-key-change-in-production"),
		Issuer:          getEnv("JWT_ISSUER", "aitip-user-service"),
		AccessTokenTTL:  time.Duration(accessTokenTTL) * time.Second,
		RefreshTokenTTL: time.Duration(refreshTokenTTL) * time.Second,
	}
}

func loadLoggerConfig() logger.Config {
	maxSize, _ := strconv.Atoi(getEnv("LOG_MAX_SIZE", "100"))
	maxAge, _ := strconv.Atoi(getEnv("LOG_MAX_AGE", "7"))
	maxBackups, _ := strconv.Atoi(getEnv("LOG_MAX_BACKUPS", "3"))
	compress, _ := strconv.ParseBool(getEnv("LOG_COMPRESS", "true"))

	return logger.Config{
		Level:      getEnv("LOG_LEVEL", "info"),
		Format:     getEnv("LOG_FORMAT", "console"),
		Output:     getEnv("LOG_OUTPUT", "stdout"),
		FilePath:   getEnv("LOG_FILE_PATH", "logs/user-service.log"),
		MaxSize:    maxSize,
		MaxAge:     maxAge,
		MaxBackups: maxBackups,
		Compress:   compress,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDatabaseLogLevel(level string) int {
	switch level {
	case "silent":
		return 1
	case "error":
		return 2
	case "warn":
		return 3
	case "info":
		return 4
	default:
		return 4
	}
}
