package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/database"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/minio"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/redis"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/config"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/handler"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/repository"
	"github.com/plucky-groove3/ai-train-infer-platform/services/data/internal/service"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := initLogger(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting data service...",
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
	)

	// 初始化数据库
	db, err := database.New(cfg.Database.ToDatabaseConfig())
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(db)

	// 初始化 Redis
	redisClient, err := redis.New(cfg.Redis.ToRedisConfig())
	if err != nil {
		logger.Fatal("Failed to connect to redis", zap.Error(err))
	}
	defer redisClient.Close()

	// 初始化 MinIO
	minioClient, err := minio.New(cfg.MinIO.ToMinIOConfig())
	if err != nil {
		logger.Fatal("Failed to connect to minio", zap.Error(err))
	}

	// 测试 MinIO 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := minioClient.HealthCheck(ctx); err != nil {
		logger.Warn("MinIO health check failed", zap.Error(err))
	}

	// 初始化仓库
	datasetRepo := repository.NewDatasetRepository(db)

	// 初始化服务
	datasetService := service.NewDatasetService(datasetRepo, minioClient, redisClient.GetClient(), cfg)

	// 初始化处理器
	uploadConfig := handler.UploadConfig{
		MaxSize:        cfg.Upload.MaxSize,
		ChunkSize:      cfg.Upload.ChunkSize,
		TempDir:        cfg.Upload.TempDir,
		AllowedFormats: cfg.Upload.AllowedFormats,
	}
	datasetHandler := handler.NewDatasetHandler(datasetService, uploadConfig)

	// 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(loggerMiddleware())
	router.Use(corsMiddleware())

	// 健康检查
	router.GET("/health", handler.HealthCheck)

	// API 路由
	apiV1 := router.Group("/api/v1")
	datasetHandler.RegisterRoutes(apiV1)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器
	go func() {
		logger.Info("Server is running", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// loadConfig 加载配置
func loadConfig() (*config.Config, error) {
	// 优先从环境变量读取配置文件路径
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		// 如果配置文件加载失败，使用默认配置
		logger.Warn("Failed to load config file, using defaults", zap.Error(err))
		cfg = config.DefaultConfig()
	}

	// 从环境变量覆盖配置
	overrideConfigFromEnv(cfg)

	return cfg, nil
}

// overrideConfigFromEnv 从环境变量覆盖配置
func overrideConfigFromEnv(cfg *config.Config) {
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := fmt.Sscanf(port, "%d", &cfg.Server.Port); p == 1 && err == nil {
		}
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		cfg.Database.Port = dbPort
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		cfg.Database.User = dbUser
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		cfg.Database.Password = dbPassword
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		cfg.Database.Database = dbName
	}
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		cfg.Redis.Port = redisPort
	}
	if minioEndpoint := os.Getenv("MINIO_ENDPOINT"); minioEndpoint != "" {
		cfg.MinIO.Endpoint = minioEndpoint
	}
	if minioAccessKey := os.Getenv("MINIO_ACCESS_KEY"); minioAccessKey != "" {
		cfg.MinIO.AccessKeyID = minioAccessKey
	}
	if minioSecretKey := os.Getenv("MINIO_SECRET_KEY"); minioSecretKey != "" {
		cfg.MinIO.SecretAccessKey = minioSecretKey
	}
}

// initLogger 初始化日志
func initLogger(cfg *config.Config) error {
	logCfg := &logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		FilePath:   cfg.Log.FilePath,
		MaxSize:    cfg.Log.MaxSize,
		MaxAge:     cfg.Log.MaxAge,
		MaxBackups: cfg.Log.MaxBackups,
		Compress:   cfg.Log.Compress,
	}
	return logger.Init(logCfg)
}

// loggerMiddleware 日志中间件
func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("HTTP Request",
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("error", c.Errors.String()),
		)
	}
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// errorHandler 错误处理中间件
func errorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			response.InternalServerError(c, err.Error())
		}
	}
}
