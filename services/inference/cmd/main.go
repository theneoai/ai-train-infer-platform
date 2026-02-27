package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/database"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
	"github.com/plucky-groove3/ai-train-infer-platform/services/inference/internal/config"
	"github.com/plucky-groove3/ai-train-infer-platform/services/inference/internal/docker"
	"github.com/plucky-groove3/ai-train-infer-platform/services/inference/internal/handler"
	"github.com/plucky-groove3/ai-train-infer-platform/services/inference/internal/repository"
	"github.com/plucky-groove3/ai-train-infer-platform/services/inference/internal/service"
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

	logger.Info("Starting Inference Service...")

	// 初始化数据库
	db, err := database.NewFromURL(cfg.DatabaseURL, 0)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// 测试数据库连接
	if err := database.HealthCheck(db); err != nil {
		logger.Fatal("Database health check failed", zap.Error(err))
	}
	logger.Info("Database connected")

	// 初始化 Docker 执行器
	dockerExec, err := docker.NewExecutor(cfg.DockerHost, cfg.DockerNetwork, cfg.ModelCachePath)
	if err != nil {
		logger.Fatal("Failed to create Docker executor", zap.Error(err))
	}
	defer dockerExec.Close()
	logger.Info("Docker executor initialized")

	// 初始化仓库
	serviceRepo := repository.NewServiceRepository(db)
	modelRepo := repository.NewModelRepository(db)

	// 初始化服务
	inferenceService := service.NewInferenceService(cfg, serviceRepo, modelRepo, dockerExec)

	// 初始化处理器
	serviceHandler := handler.NewServiceHandler(inferenceService)

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
			"service":   "inference",
			"timestamp": time.Now().Unix(),
		})
	})

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 注册推理服务路由
		serviceHandler.RegisterRoutes(v1)
	}

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("Inference Service started", zap.String("port", cfg.Port))

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Inference Service...")

	// 停止所有运行中的服务
	if err := inferenceService.StopAll(context.Background()); err != nil {
		logger.Error("Failed to stop all services", zap.Error(err))
	}

	// 优雅关闭 HTTP 服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Inference Service stopped")
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
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
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
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
			)
			response.Error(c, http.StatusInternalServerError, err.Error())
		}
	}
}
