package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/database"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	pkgRedis "github.com/plucky-groove3/ai-train-infer-platform/pkg/redis"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/config"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/executor"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/handler"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/repository"
	"github.com/plucky-groove3/ai-train-infer-platform/services/training/internal/service"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化日志
	logCfg := &logger.Config{
		Level:    cfg.LogLevel,
		Format:   cfg.LogFormat,
		Output:   cfg.LogOutput,
		FilePath: cfg.LogPath,
	}
	if err := logger.Init(logCfg); err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("Starting Training Service...")

	// 初始化数据库
	db, err := database.NewFromURL(cfg.DatabaseURL, 0)
	if err != nil {
		logger.Fatal("Failed to connect to database", logger.WithField("error", err))
	}

	// 测试数据库连接
	if err := database.HealthCheck(db); err != nil {
		logger.Fatal("Database health check failed", logger.WithField("error", err))
	}
	logger.Info("Database connected")

	// 初始化 Redis
	redisClient, err := pkgRedis.NewFromURL(cfg.RedisURL)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", logger.WithField("error", err))
	}
	defer redisClient.Close()
	logger.Info("Redis connected")

	// 初始化仓库
	jobRepo := repository.NewJobRepository(db)
	logRepo := repository.NewLogRepository(redisClient.GetClient(), cfg.LogStreamMaxLen)

	// 初始化执行器
	dockerExec := executor.NewDockerExecutor(
		logRepo,
		cfg.DockerNetwork,
		cfg.DockerVolumeBase,
	)

	// 初始化服务
	jobService := service.NewJobService(cfg, jobRepo, logRepo, dockerExec)

	// 初始化处理器
	jobHandler := handler.NewJobHandler(jobService)

	// 设置 Gin 模式
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	router := gin.New()

	// 使用中间件
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestLogger())
	router.Use(errorHandler())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "training",
			"timestamp": time.Now().Unix(),
		})
	})

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 注册任务路由
		jobHandler.RegisterRoutes(v1)
	}

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", logger.WithField("error", err))
		}
	}()

	logger.Info("Training Service started", logger.WithField("port", cfg.Port))

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Training Service...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", logger.WithField("error", err))
	}

	logger.Info("Training Service stopped")
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// requestLogger 请求日志中间件
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		// 日志记录
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("HTTP Request",
			logger.WithField("client_ip", clientIP),
			logger.WithField("method", method),
			logger.WithField("path", path),
			logger.WithField("status", statusCode),
			logger.WithField("latency", latency),
		)
	}
}

// errorHandler 错误处理中间件
func errorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			logger.Error("Request error",
				logger.WithField("error", err.Error()),
				logger.WithField("path", c.Request.URL.Path),
			)
			response.InternalServerError(c, err.Error())
		}
	}
}
