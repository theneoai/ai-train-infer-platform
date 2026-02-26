package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plucky-groove3/ai-train-infer-platform/services/gateway/internal/config"
	"github.com/plucky-groove3/ai-train-infer-platform/services/gateway/internal/handler"
	"github.com/plucky-groove3/ai-train-infer-platform/services/gateway/internal/middleware"
)

func main() {
	cfg := config.Load()

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", handler.HealthCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Training
		training := v1.Group("/train")
		{
			training.GET("/jobs", handler.ListTrainingJobs)
			training.POST("/jobs", handler.CreateTrainingJob)
			training.GET("/jobs/:id", handler.GetTrainingJob)
			training.DELETE("/jobs/:id", handler.DeleteTrainingJob)
			training.GET("/jobs/:id/logs", handler.StreamTrainingLogs)
		}

		// Inference
		inference := v1.Group("/inference")
		{
			inference.GET("/services", handler.ListInferenceServices)
			inference.POST("/services", handler.CreateInferenceService)
			inference.GET("/services/:id", handler.GetInferenceService)
			inference.PATCH("/services/:id", handler.UpdateInferenceService)
			inference.DELETE("/services/:id", handler.DeleteInferenceService)
		}

		// Simulation
		simulation := v1.Group("/simulation")
		{
			simulation.GET("/environments", handler.ListSimEnvironments)
			simulation.POST("/environments", handler.CreateSimEnvironment)
			simulation.GET("/environments/:id", handler.GetSimEnvironment)
			simulation.POST("/environments/:id/run", handler.RunSimulation)
			simulation.DELETE("/environments/:id", handler.DeleteSimEnvironment)
		}

		// Experiments
		experiments := v1.Group("/experiments")
		{
			experiments.GET("", handler.ListExperiments)
			experiments.POST("", handler.CreateExperiment)
			experiments.GET("/:id", handler.GetExperiment)
			experiments.GET("/:id/runs", handler.ListRuns)
			experiments.POST("/:id/runs", handler.StartRun)
		}

		// Agent API
		agent := v1.Group("/agent")
		{
			agent.GET("/tools", handler.ListAgentTools)
			agent.POST("/execute", handler.ExecuteAgentTool)
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
