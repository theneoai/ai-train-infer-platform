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
)

const (
	// ContextKeyUserID 用户 ID 上下文键
	ContextKeyUserID = "user_id"
	// ContextKeyUserEmail 用户邮箱上下文键
	ContextKeyUserEmail = "user_email"
	// ContextKeyUserRole 用户角色上下文键
	ContextKeyUserRole = "user_role"
)

// JWTAuth JWT 认证中间件
func JWTAuth(jwtManager *jwt.Manager, userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Unauthorized(c, "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]

		// 验证 Token
		claims, err := userService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			logger.Warn("Token validation failed", logger.WithField("error", err.Error()).Fields...)
			response.ErrorWithMessage(c, response.ErrorUnauthorized, err.Error())
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserEmail, claims.Email)
		c.Set(ContextKeyUserRole, claims.Role)

		c.Next()
	}
}

// OptionalAuth 可选认证中间件（不强制要求登录）
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

		// 验证 Token（忽略错误）
		claims, err := userService.ValidateToken(c.Request.Context(), token)
		if err == nil {
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyUserEmail, claims.Email)
			c.Set(ContextKeyUserRole, claims.Role)
		}

		c.Next()
	}
}

// APIKeyAuth API Key 认证中间件
func APIKeyAuth(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// 尝试从查询参数获取
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			response.Unauthorized(c, "api key required")
			c.Abort()
			return
		}

		// 验证 API Key
		key, err := userService.ValidateAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			logger.Warn("API key validation failed", logger.WithField("error", err.Error()).Fields...)
			response.Unauthorized(c, "invalid api key")
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set(ContextKeyUserID, key.UserID.String())

		c.Next()
	}
}

// RoleAuth 角色权限中间件
func RoleAuth(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyUserRole)
		if !exists {
			response.Forbidden(c, "role not found")
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

		response.Forbidden(c, "insufficient permissions")
		c.Abort()
	}
}

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []interface{}{
			"status", status,
			"latency", latency,
			"client_ip", c.ClientIP(),
			"method", c.Request.Method,
			"path", path,
		}

		if query != "" {
			fields = append(fields, "query", query)
		}

		if len(c.Errors) > 0 {
			fields = append(fields, "errors", c.Errors.String())
			logger.Error("Request failed", fields...)
		} else if status >= 500 {
			logger.Error("Server error", fields...)
		} else if status >= 400 {
			logger.Warn("Client error", fields...)
		} else {
			logger.Info("Request processed", fields...)
		}
	}
}

// Recovery 自定义恢复中间件
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("Panic recovered", 
			"error", recovered,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
		)
		response.InternalServerError(c, "internal server error")
	})
}

// CORSMiddleware CORS 中间件
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

// RateLimiter 简单速率限制中间件
func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	// 简单的内存实现，生产环境建议使用 Redis
	requests := make(map[string][]time.Time)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// 清理过期的请求记录
		if timestamps, exists := requests[clientIP]; exists {
			var valid []time.Time
			for _, t := range timestamps {
				if now.Sub(t) < window {
					valid = append(valid, t)
				}
			}
			requests[clientIP] = valid
		}

		// 检查是否超过限制
		if len(requests[clientIP]) >= maxRequests {
			response.TooManyRequests(c, "rate limit exceeded, please try again later")
			c.Abort()
			return
		}

		// 记录当前请求
		requests[clientIP] = append(requests[clientIP], now)

		c.Next()
	}
}
