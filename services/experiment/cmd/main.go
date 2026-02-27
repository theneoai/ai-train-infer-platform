package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/database"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
	"github.com/plucky-groove3/ai-train-infer-platform/services/experiment/internal/handler"
	"github.com/plucky-groove3/ai-train-infer-platform/services/experiment/internal/repository"
	"github.com/plucky-groove3/ai-train-infer-platform/services/experiment/internal/service"
	"go.uber.org/zap"
)

func main() {
	// Load config
	dbURL := getEnv("DATABASE_URL", "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable")
	port := getEnv("PORT", "8085")
	mlflowEnabled := getEnvBool("MLFLOW_ENABLED", false)
	mlflowURI := getEnv("MLFLOW_TRACKING_URI", "http://localhost:5000")

	// Initialize logger
	if err := logger.InitDevelopment(); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	// Connect to database
	db, err := database.NewFromURL(dbURL, 1)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}

	// Initialize repositories
	expRepo := repository.NewExperimentRepository(db)
	runRepo := repository.NewRunRepository(db)
	metricRepo := repository.NewMetricRepository(db)
	artifactRepo := repository.NewArtifactRepository(db)

	// Initialize services
	expService := service.NewExperimentService(expRepo, runRepo, metricRepo, artifactRepo)
	runService := service.NewRunService(runRepo, expRepo, metricRepo)
	metricService := service.NewMetricService(metricRepo, runRepo)
	vizService := service.NewVisualizationService(expRepo, runRepo, metricRepo)
	_ = service.NewMLflowService(mlflowEnabled, mlflowURI)

	// Initialize handlers
	expHandler := handler.NewExperimentHandler(expService, runService, metricService, vizService)
	runHandler := handler.NewRunHandler(runService, metricService)
	metricHandler := handler.NewMetricHandler(metricService)
	vizHandler := handler.NewVisualizationHandler(vizService)

	// Setup router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":  "healthy",
			"service": "experiment",
		})
	})

	// API v1 routes
	apiV1 := router.Group("/api/v1")
	{
		// Experiment routes
		experiments := apiV1.Group("/experiments")
		{
			experiments.POST("", expHandler.CreateExperiment)
			experiments.GET("", expHandler.ListExperiments)
			experiments.GET("/:id", expHandler.GetExperiment)
			experiments.PUT("/:id", expHandler.UpdateExperiment)
			experiments.DELETE("/:id", expHandler.DeleteExperiment)
			experiments.GET("/:id/runs", expHandler.GetExperimentRuns)
			experiments.GET("/:id/report", vizHandler.GetExperimentReport)
		}

		// Experiment comparison
		apiV1.POST("/experiments/compare", vizHandler.CompareExperiments)
		apiV1.POST("/experiments/hyperparameters/compare", vizHandler.GetHyperparameterComparison)

		// Run routes
		runs := apiV1.Group("/runs")
		{
			runs.POST("", runHandler.CreateRun)
			runs.GET("/:id", runHandler.GetRun)
			runs.PUT("/:id/status", runHandler.UpdateRunStatus)
			runs.POST("/:id/complete", runHandler.CompleteRun)
			runs.GET("/:id/metrics", metricHandler.GetRunMetrics)
			runs.GET("/:id/metrics/:key/series", metricHandler.GetMetricSeries)
			runs.GET("/:id/loss-curve", vizHandler.GetLossCurve)
			runs.GET("/:id/accuracy-trend", vizHandler.GetAccuracyTrend)
		}

		// Metric routes
		metrics := apiV1.Group("/metrics")
		{
			metrics.POST("", metricHandler.RecordMetric)
			metrics.POST("/batch", metricHandler.BatchRecordMetrics)
			metrics.GET("/query", metricHandler.QueryMetrics)
		}
	}

	logger.Log.Info("Experiment service starting", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Log.Fatal("Server failed", zap.Error(err))
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

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
