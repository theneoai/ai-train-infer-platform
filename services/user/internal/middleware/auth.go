package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/ai-train-infer-platform/pkg/jwt"
	"github.com/ai-train-infer-platform/pkg/logger"
	"github.com/ai-train-infer-platform/pkg/response"
	"github.com/ai-train-infer-platform/services/user/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	ContextKeyUserID    = "user_id"
	ContextKeyUserEmail = "user_email"
	ContextKeyUserRole  = "user_role"
)

func JWTAuth(jwtManager *jwt.Manager, userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(c, http.StatusUnauthorized, "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]

		claims, err := userService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			logger.Log.Warn("Token validation failed", zap.Error(err))
			response.Error(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserEmail, claims.Email)
		c.Set(ContextKeyUserRole, claims.Role)

		c.Next()
	}
}

func OptionalAuth(jwtManager *jwt.Manager, userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.Next()
			return
		}

		token := parts[1]

		claims, err := userService.ValidateToken(c.Request.Context(), token)
		if err == nil {
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyUserEmail, claims.Email)
			c.Set(ContextKeyUserRole, claims.Role)
		}

		c.Next()
	}
}

func APIKeyAuth(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			response.Error(c, http.StatusUnauthorized, "api key required")
			c.Abort()
			return
		}

		key, err := userService.ValidateAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			logger.Log.Warn("API key validation failed", zap.Error(err))
			response.Error(c, http.StatusUnauthorized, "invalid api key")
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, key.UserID.String())

		c.Next()
	}
}

func RoleAuth(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyUserRole)
		if !exists {
			response.Error(c, http.StatusForbidden, "role not found")
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		response.Error(c, http.StatusForbidden, "insufficient permissions")
		c.Abort()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
			logger.Log.Error("Request failed", fields...)
		} else if status >= 500 {
			logger.Log.Error("Server error", fields...)
		} else if status >= 400 {
			logger.Log.Warn("Client error", fields...)
		} else {
			logger.Log.Info("Request processed", fields...)
		}
	}
}

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Log.Error("Panic recovered",
			zap.Any("error", recovered),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	})
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	requests := make(map[string][]time.Time)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		if timestamps, exists := requests[clientIP]; exists {
			var valid []time.Time
			for _, t := range timestamps {
				if now.Sub(t) < window {
					valid = append(valid, t)
				}
			}
			requests[clientIP] = valid
		}

		if len(requests[clientIP]) >= maxRequests {
			response.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}

		requests[clientIP] = append(requests[clientIP], now)

		c.Next()
	}
}
