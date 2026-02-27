package config

import (
	"fmt"
	"time"

	"github.com/plucky-groove3/ai-train-infer-platform/pkg/database"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/minio"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/redis"
	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	MinIO    MinIOConfig    `mapstructure:"minio"`
	Log      LogConfig      `mapstructure:"log"`
	Upload   UploadConfig   `mapstructure:"upload"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	Mode            string        `mapstructure:"mode"` // debug, release, test
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string        `mapstructure:"host"`
	Port     string        `mapstructure:"port"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	PoolSize int           `mapstructure:"pool_size"`
	KeyPrefix string       `mapstructure:"key_prefix"`
}

// MinIOConfig MinIO 配置
type MinIOConfig struct {
	Endpoint        string        `mapstructure:"endpoint"`
	AccessKeyID     string        `mapstructure:"access_key_id"`
	SecretAccessKey string        `mapstructure:"secret_access_key"`
	UseSSL          bool          `mapstructure:"use_ssl"`
	Region          string        `mapstructure:"region"`
	PresignedExpiry time.Duration `mapstructure:"presigned_expiry"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json, console
	Output     string `mapstructure:"output"` // stdout, file, both
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxAge     int    `mapstructure:"max_age"`     // days
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxSize        int64         `mapstructure:"max_size"`         // 最大文件大小
	ChunkSize      int64         `mapstructure:"chunk_size"`       // 分片大小
	MaxConcurrency int           `mapstructure:"max_concurrency"`  // 最大并发数
	AllowedFormats []string      `mapstructure:"allowed_formats"`  // 允许的文件格式
	TempDir        string        `mapstructure:"temp_dir"`         // 临时目录
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"` // 清理间隔
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8083,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			Mode:            "release",
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            "5432",
			User:            "postgres",
			Password:        "password",
			Database:        "aitip",
			SSLMode:         "disable",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
		},
		Redis: RedisConfig{
			Host:      "localhost",
			Port:      "6379",
			Password:  "",
			DB:        0,
			PoolSize:  10,
			KeyPrefix: "data-service:",
		},
		MinIO: MinIOConfig{
			Endpoint:        "localhost:9000",
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			UseSSL:          false,
			Region:          "us-east-1",
			PresignedExpiry: 15 * time.Minute,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			FilePath:   "logs/data-service.log",
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 3,
			Compress:   true,
		},
		Upload: UploadConfig{
			MaxSize:         10 * 1024 * 1024 * 1024, // 10GB
			ChunkSize:       64 * 1024 * 1024,        // 64MB
			MaxConcurrency:  4,
			AllowedFormats:  []string{"csv", "json", "parquet", "txt", "zip", "tar", "gz"},
			TempDir:         "/tmp/data-service",
			CleanupInterval: time.Hour,
		},
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// 配置文件不存在，使用默认值
	}

	// 从环境变量读取
	viper.SetEnvPrefix("DATA_SERVICE")
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults() {
	defaults := DefaultConfig()
	viper.SetDefault("server.host", defaults.Server.Host)
	viper.SetDefault("server.port", defaults.Server.Port)
	viper.SetDefault("server.read_timeout", defaults.Server.ReadTimeout)
	viper.SetDefault("server.write_timeout", defaults.Server.WriteTimeout)
	viper.SetDefault("server.shutdown_timeout", defaults.Server.ShutdownTimeout)
	viper.SetDefault("server.mode", defaults.Server.Mode)

	viper.SetDefault("database.host", defaults.Database.Host)
	viper.SetDefault("database.port", defaults.Database.Port)
	viper.SetDefault("database.user", defaults.Database.User)
	viper.SetDefault("database.password", defaults.Database.Password)
	viper.SetDefault("database.database", defaults.Database.Database)
	viper.SetDefault("database.ssl_mode", defaults.Database.SSLMode)

	viper.SetDefault("redis.host", defaults.Redis.Host)
	viper.SetDefault("redis.port", defaults.Redis.Port)

	viper.SetDefault("minio.endpoint", defaults.MinIO.Endpoint)
	viper.SetDefault("minio.access_key_id", defaults.MinIO.AccessKeyID)
	viper.SetDefault("minio.secret_access_key", defaults.MinIO.SecretAccessKey)

	viper.SetDefault("log.level", defaults.Log.Level)
	viper.SetDefault("log.format", defaults.Log.Format)

	viper.SetDefault("upload.max_size", defaults.Upload.MaxSize)
	viper.SetDefault("upload.chunk_size", defaults.Upload.ChunkSize)
}

// ToDatabaseConfig 转换为数据库配置
func (c *DatabaseConfig) ToDatabaseConfig() *database.Config {
	return &database.Config{
		Host:            c.Host,
		Port:            c.Port,
		User:            c.User,
		Password:        c.Password,
		Database:        c.Database,
		SSLMode:         c.SSLMode,
		MaxIdleConns:    c.MaxIdleConns,
		MaxOpenConns:    c.MaxOpenConns,
		ConnMaxLifetime: c.ConnMaxLifetime,
	}
}

// ToRedisConfig 转换为 Redis 配置
func (c *RedisConfig) ToRedisConfig() *redis.Config {
	return &redis.Config{
		Host:     c.Host,
		Port:     c.Port,
		Password: c.Password,
		DB:       c.DB,
		PoolSize: c.PoolSize,
	}
}

// ToMinIOConfig 转换为 MinIO 配置
func (c *MinIOConfig) ToMinIOConfig() *minio.Config {
	return &minio.Config{
		Endpoint:        c.Endpoint,
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		UseSSL:          c.UseSSL,
		Region:          c.Region,
		PresignedExpiry: c.PresignedExpiry,
	}
}

// GetUploadProgressKey 获取上传进度 Redis key
func (c *RedisConfig) GetUploadProgressKey(uploadID string) string {
	return fmt.Sprintf("%supload:%s:progress", c.KeyPrefix, uploadID)
}

// GetUploadLockKey 获取上传锁 Redis key
func (c *RedisConfig) GetUploadLockKey(uploadID string) string {
	return fmt.Sprintf("%supload:%s:lock", c.KeyPrefix, uploadID)
}
