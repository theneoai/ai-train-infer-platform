package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/database"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
)

func main() {
	// Load config
	dbURL := getEnv("DATABASE_URL", "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable")
	port := getEnv("PORT", "8085")

	// Connect to database
	db, err := database.Connect(dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	logger := logger.New("info")

	router := gin.New()
	router.Use(gin.Recovery())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "healthy", "service": "experiment"})
	})

	// TODO: Add experiment routes

	logger.Info("Experiment service starting", "port", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
