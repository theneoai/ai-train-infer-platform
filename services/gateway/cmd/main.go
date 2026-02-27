package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/services/gateway/internal/config"
	"github.com/plucky-groove3/ai-train-infer-platform/services/gateway/internal/middleware"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
)

// ServiceEndpoints maps service names to their URLs
type ServiceEndpoints struct {
	User      string
	Data      string
	Training  string
	Inference string
}

func main() {
	cfg := config.Load()
	logger := logger.New(cfg.LogLevel)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	// Service endpoints
	services := ServiceEndpoints{
		User:      getEnv("USER_SERVICE_URL", "http://user:8081"),
		Data:      getEnv("DATA_SERVICE_URL", "http://data:8082"),
		Training:  getEnv("TRAINING_SERVICE_URL", "http://training:8083"),
		Inference: getEnv("INFERENCE_SERVICE_URL", "http://inference:8084"),
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":  "healthy",
			"service": "gateway",
			"version": "0.1.0",
		})
	})

	// Public routes (no auth required)
	public := router.Group("/api/v1")
	{
		// Auth routes - forwarded to user service
		public.POST("/auth/register", forwardTo(services.User))
		public.POST("/auth/login", forwardTo(services.User))
		public.POST("/auth/refresh", forwardTo(services.User))
	}

	// Protected routes (auth required)
	protected := router.Group("/api/v1")
	protected.Use(middleware.Auth(cfg.JWTSecret))
	{
		// User routes
		protected.GET("/auth/me", forwardTo(services.User))
		protected.POST("/auth/logout", forwardTo(services.User))
		protected.GET("/users/profile", forwardTo(services.User))
		protected.POST("/api-keys", forwardTo(services.User))
		protected.GET("/api-keys", forwardTo(services.User))
		protected.DELETE("/api-keys/:id", forwardTo(services.User))

		// Dataset routes
		protected.GET("/datasets", forwardTo(services.Data))
		protected.POST("/datasets", forwardTo(services.Data))
		protected.GET("/datasets/:id", forwardTo(services.Data))
		protected.DELETE("/datasets/:id", forwardTo(services.Data))
		protected.GET("/datasets/:id/download", forwardTo(services.Data))

		// Training routes
		protected.GET("/training/jobs", forwardTo(services.Training))
		protected.POST("/training/jobs", forwardTo(services.Training))
		protected.GET("/training/jobs/:id", forwardTo(services.Training))
		protected.DELETE("/training/jobs/:id", forwardTo(services.Training))
		protected.GET("/training/jobs/:id/logs", forwardTo(services.Training))
		protected.POST("/training/jobs/:id/stop", forwardTo(services.Training))

		// Inference routes
		protected.GET("/inference/services", forwardTo(services.Inference))
		protected.POST("/inference/services", forwardTo(services.Inference))
		protected.GET("/inference/services/:id", forwardTo(services.Inference))
		protected.DELETE("/inference/services/:id", forwardTo(services.Inference))
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Gateway starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}

// forwardTo creates a reverse proxy handler for a service
func forwardTo(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, err := url.Parse(targetURL)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "invalid service URL")
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		
		// Customize the director to preserve the original path
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.URL.Path = c.Request.URL.Path
			req.URL.RawQuery = c.Request.URL.RawQuery
			
			// Forward user info from context to headers
			if userID := middleware.GetUserID(c); userID != "" {
				req.Header.Set("X-User-ID", userID)
			}
			if email := middleware.GetEmail(c); email != "" {
				req.Header.Set("X-User-Email", email)
			}
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
