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
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	if err := logger.Init(&cfg.Logger); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Log.Sync()

	logger.Log.Info("Starting User Service",
		zap.String("port", cfg.Server.Port),
		zap.String("environment", cfg.Server.Environment),
	)

	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}

	if err := autoMigrate(db); err != nil {
		logger.Log.Fatal("Failed to auto migrate", zap.Error(err))
	}

	jwtConfig := &cfg.JWT
	jwtManager := jwt.NewManager(jwtConfig)

	userRepo := repository.NewUserRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	blacklistRepo := repository.NewTokenBlacklistRepository(db)

	userService := service.NewUserService(userRepo, apiKeyRepo, blacklistRepo, jwtManager)

	authHandler := handler.NewAuthHandler(userService, jwtManager)
	userHandler := handler.NewUserHandler(userService)
	apiKeyHandler := handler.NewAPIKeyHandler(userService)

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORSMiddleware())

	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":    "healthy",
			"service":   "user-service",
			"timestamp": time.Now().Unix(),
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		if err := database.HealthCheck(db); err != nil {
			response.Error(c, http.StatusServiceUnavailable, "database not ready")
			return
		}
		response.Success(c, gin.H{
			"status":  "ready",
			"service": "user-service",
		})
	})

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		authenticated := v1.Group("")
		authenticated.Use(middleware.JWTAuth(jwtManager, userService))
		{
			auth := authenticated.Group("/auth")
			{
				auth.GET("/me", authHandler.Me)
				auth.POST("/logout", authHandler.Logout)
			}

			users := authenticated.Group("/users")
			{
				users.GET("/profile", userHandler.GetProfile)
			}

			apiKeys := authenticated.Group("/api-keys")
			{
				apiKeys.POST("", apiKeyHandler.Create)
				apiKeys.GET("", apiKeyHandler.List)
				apiKeys.DELETE("/:id", apiKeyHandler.Delete)
			}
		}
	}

	srv := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Log.Info("User Service started", zap.String("address", cfg.Server.Port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down User Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
	}

	if err := database.Close(db); err != nil {
		logger.Log.Error("Failed to close database", zap.Error(err))
	}

	logger.Log.Info("User Service stopped")
}

func autoMigrate(db *gorm.DB) error {
	logger.Log.Info("Running database migrations...")
	
	models := []interface{}{
		&domain.User{},
		&domain.APIKey{},
		&domain.TokenBlacklist{},
	}

	if err := database.AutoMigrate(db, models...); err != nil {
		return err
	}

	logger.Log.Info("Database migrations completed")
	return nil
}
