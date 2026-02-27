package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-train-infer-platform/services/user/internal/config"
	"github.com/ai-train-infer-platform/services/user/internal/domain"
	"github.com/ai-train-infer-platform/services/user/internal/handler"
	"github.com/ai-train-infer-platform/services/user/internal/middleware"
	"github.com/ai-train-infer-platform/services/user/internal/repository"
	"github.com/ai-train-infer-platform/services/user/internal/service"
	"github.com/ai-train-infer-platform/pkg/database"
	"github.com/ai-train-infer-platform/pkg/jwt"
	"github.com/ai-train-infer-platform/pkg/logger"
	"github.com/ai-train-infer-platform/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化日志
	if err := logger.Init(&cfg.Logger); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("Starting User Service",
		"port", cfg.Server.Port,
		"environment", cfg.Server.Environment,
	)

	// 连接数据库
	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err.Error())
	}

	// 自动迁移
	if err := autoMigrate(db); err != nil {
		logger.Fatal("Failed to auto migrate", "error", err.Error())
	}

	// 初始化 JWT 管理器
	jwtConfig := &cfg.JWT
	jwtManager := jwt.NewManager(jwtConfig)

	// 初始化仓库
	userRepo := repository.NewUserRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	blacklistRepo := repository.NewTokenBlacklistRepository(db)

	// 初始化服务
	userService := service.NewUserService(userRepo, apiKeyRepo, blacklistRepo, jwtManager)

	// 初始化处理器
	authHandler := handler.NewAuthHandler(userService, jwtManager)
	userHandler := handler.NewUserHandler(userService)
	apiKeyHandler := handler.NewAPIKeyHandler(userService)

	// 设置 Gin 模式
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORSMiddleware())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":    "healthy",
			"service":   "user-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// 准备检查
	router.GET("/ready", func(c *gin.Context) {
		if err := database.HealthCheck(db); err != nil {
			response.ErrorWithMessage(c, response.ErrorServiceUnavailable, "database not ready")
			return
		}
		response.Success(c, gin.H{
			"status":  "ready",
			"service": "user-service",
		})
	})

	// API 路由
	v1 := router.Group("/api/v1")
	{
		// 公开路由（不需要认证）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// 需要认证的路由
		authenticated := v1.Group("")
		authenticated.Use(middleware.JWTAuth(jwtManager, userService))
		{
			// 认证相关
			auth := authenticated.Group("/auth")
			{
				auth.GET("/me", authHandler.Me)
				auth.POST("/logout", authHandler.Logout)
			}

			// 用户相关
			users := authenticated.Group("/users")
			{
				users.GET("/profile", userHandler.GetProfile)
			}

			// API Key 管理
			apiKeys := authenticated.Group("/api-keys")
			{
				apiKeys.POST("", apiKeyHandler.Create)
				apiKeys.GET("", apiKeyHandler.List)
				apiKeys.DELETE("/:id", apiKeyHandler.Delete)
			}
		}
	}

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err.Error())
		}
	}()

	logger.Info("User Service started", "address", cfg.Server.Port)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down User Service...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err.Error())
	}

	// 关闭数据库连接
	if err := database.Close(db); err != nil {
		logger.Error("Failed to close database", "error", err.Error())
	}

	logger.Info("User Service stopped")
}

// autoMigrate 自动迁移数据库
func autoMigrate(db *gorm.DB) error {
	logger.Info("Running database migrations...")
	
	models := []interface{}{
		&domain.User{},
		&domain.APIKey{},
		&domain.TokenBlacklist{},
	}

	if err := database.AutoMigrate(db, models...); err != nil {
		return err
	}

	logger.Info("Database migrations completed")
	return nil
}
