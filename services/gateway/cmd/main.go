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
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/jwt"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/models"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/ratelimit"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/redis"
	"github.com/plucky-groove3/ai-train-infer-platform/services/gateway/internal/config"
	"github.com/plucky-groove3/ai-train-infer-platform/services/gateway/internal/handler"
	"github.com/plucky-groove3/ai-train-infer-platform/services/gateway/internal/middleware"
)

func main() {
	cfg := config.Load()

	// Initialize logger
	logCfg := &logger.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
		Output: cfg.LogOutput,
	}
	if err := logger.Init(logCfg); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	logger.Info("Starting gateway service",
		logger.WithField("environment", cfg.Environment),
		logger.WithField("port", cfg.Port),
	)

	// Initialize database
	db, err := database.NewFromURL(cfg.DatabaseURL, logger.LogLevel())
	if err != nil {
		logger.Fatal("Failed to connect to database", logger.WithField("error", err))
	}
	defer database.Close(db)

	// Auto migrate
	if cfg.Environment != "production" {
		if err := models.AutoMigrate(db); err != nil {
			logger.Error("Failed to auto migrate", logger.WithField("error", err))
		}
	}

	// Initialize Redis
	rdb, err := redis.NewFromURL(cfg.RedisURL)
	if err != nil {
		logger.Fatal("Failed to connect to redis", logger.WithField("error", err))
	}
	defer rdb.Close()

	// Initialize JWT manager
	jwtCfg := &jwt.Config{
		SecretKey:       cfg.JWTSecretKey,
		Issuer:          cfg.JWTIssuer,
		AccessTokenTTL:  cfg.JWTAccessTokenTTL,
		RefreshTokenTTL: cfg.JWTRefreshTokenTTL,
	}
	jwtManager := jwt.NewManager(jwtCfg)

	// Initialize rate limiter
	rateLimitCfg := &ratelimit.Config{
		Rate:       cfg.RateLimitRate,
		Burst:      cfg.RateLimitBurst,
		WindowSize: cfg.RateLimitWindow,
	}
	rateLimiter := ratelimit.NewTokenBucket(rdb, rateLimitCfg, "gateway")

	// Create handlers
	h := handler.New(db, rdb, jwtManager)

	// Setup router
	router := setupRouter(cfg, jwtManager, rateLimiter, h)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Server starting", logger.WithField("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", logger.WithField("error", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", logger.WithField("error", err))
	}

	logger.Info("Server exited")
}

func setupRouter(cfg *config.Config, jwtManager *jwt.Manager, rateLimiter ratelimit.Limiter, h *handler.Handler) *gin.Engine {
	router := gin.New()

	// Global middlewares
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())

	// Health check (no auth required)
	router.GET("/health", h.HealthCheck)
	router.GET("/ready", h.ReadyCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/register", h.Register)
			auth.POST("/refresh", h.RefreshToken)
			auth.POST("/logout", h.Logout)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(jwtManager))
		{
			// User routes
			protected.GET("/user/profile", h.GetUserProfile)
			protected.PATCH("/user/profile", h.UpdateUserProfile)
			protected.GET("/user/projects", h.GetUserProjects)

			// Organization routes
			protected.GET("/organizations", h.ListOrganizations)
			protected.POST("/organizations", middleware.RBAC("admin", "manager"), h.CreateOrganization)
			protected.GET("/organizations/:id", h.GetOrganization)
			protected.PATCH("/organizations/:id", middleware.RBAC("admin", "manager"), h.UpdateOrganization)
			protected.DELETE("/organizations/:id", middleware.RBAC("admin"), h.DeleteOrganization)

			// Project routes
			protected.GET("/projects", h.ListProjects)
			protected.POST("/projects", h.CreateProject)
			protected.GET("/projects/:id", h.GetProject)
			protected.PATCH("/projects/:id", h.UpdateProject)
			protected.DELETE("/projects/:id", h.DeleteProject)

			// Training routes with rate limiting
			training := protected.Group("/train")
			training.Use(middleware.RateLimitByUser(rateLimiter))
			{
				training.GET("/jobs", h.ListTrainingJobs)
				training.POST("/jobs", h.CreateTrainingJob)
				training.GET("/jobs/:id", h.GetTrainingJob)
				training.PATCH("/jobs/:id", h.UpdateTrainingJob)
				training.DELETE("/jobs/:id", h.DeleteTrainingJob)
				training.POST("/jobs/:id/stop", h.StopTrainingJob)
				training.GET("/jobs/:id/logs", h.StreamTrainingLogs)
				training.GET("/jobs/:id/metrics", h.GetTrainingMetrics)
				training.GET("/jobs/:id/checkpoints", h.ListCheckpoints)
			}

			// Inference routes
			inference := protected.Group("/inference")
			inference.Use(middleware.RateLimit(rateLimiter))
			{
				inference.GET("/services", h.ListInferenceServices)
				inference.POST("/services", h.CreateInferenceService)
				inference.GET("/services/:id", h.GetInferenceService)
				inference.PATCH("/services/:id", h.UpdateInferenceService)
				inference.DELETE("/services/:id", h.DeleteInferenceService)
				inference.POST("/services/:id/scale", h.ScaleInferenceService)
			}

			// Simulation routes
			simulation := protected.Group("/simulation")
			simulation.Use(middleware.RateLimit(rateLimiter))
			{
				simulation.GET("/environments", h.ListSimEnvironments)
				simulation.POST("/environments", h.CreateSimEnvironment)
				simulation.GET("/environments/:id", h.GetSimEnvironment)
				simulation.POST("/environments/:id/run", h.RunSimulation)
				simulation.DELETE("/environments/:id", h.DeleteSimEnvironment)
			}

			// Experiment routes
			experiments := protected.Group("/experiments")
			{
				experiments.GET("", h.ListExperiments)
				experiments.POST("", h.CreateExperiment)
				experiments.GET("/:id", h.GetExperiment)
				experiments.PATCH("/:id", h.UpdateExperiment)
				experiments.DELETE("/:id", h.DeleteExperiment)
				experiments.GET("/:id/runs", h.ListRuns)
				experiments.POST("/:id/runs", h.StartRun)
				experiments.GET("/:id/metrics", h.GetExperimentMetrics)
			}

			// Run routes
			runs := protected.Group("/runs")
			{
				runs.GET("/:id", h.GetRun)
				runs.GET("/:id/metrics", h.GetRunMetrics)
				runs.POST("/:id/metrics", h.RecordMetrics)
				runs.GET("/:id/artifacts", h.ListArtifacts)
				runs.POST("/:id/artifacts", h.CreateArtifact)
				runs.GET("/:id/logs", h.StreamRunLogs)
			}

			// Model registry
			models := protected.Group("/models")
			{
				models.GET("", h.ListModels)
				models.POST("", h.CreateModel)
				models.GET("/:id", h.GetModel)
				models.PATCH("/:id", h.UpdateModel)
				models.DELETE("/:id", h.DeleteModel)
				models.GET("/:id/versions", h.ListModelVersions)
				models.POST("/:id/versions", h.CreateModelVersion)
			}

			// Agent API
			agent := protected.Group("/agent")
			agent.Use(middleware.RateLimit(rateLimiter))
			{
				agent.GET("/tools", h.ListAgentTools)
				agent.POST("/execute", h.ExecuteAgentTool)
				agent.GET("/tools/:tool/schema", h.GetToolSchema)
			}
		}
	}

	return router
}
