package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/jwt"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/logger"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/ratelimit"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
	"go.uber.org/zap"
)

// JWTAuth JWT 认证中间件
func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Unauthorized(c, "invalid authorization header format")
			c.Abort()
			return
		}

		claims, err := jwtManager.ParseToken(parts[1])
		if err != nil {
			switch err {
			case jwt.ErrExpiredToken:
				response.Unauthorized(c, "token has expired")
			case jwt.ErrInvalidToken:
				response.Unauthorized(c, "invalid token")
			default:
				response.Unauthorized(c, err.Error())
			}
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("orgID", claims.OrgID)
		c.Set("scopes", claims.Scopes)
		c.Set("claims", claims)

		c.Next()
	}
}

// OptionalAuth 可选认证中间件
func OptionalAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
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

		claims, err := jwtManager.ParseToken(parts[1])
		if err != nil {
			c.Next()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("orgID", claims.OrgID)
		c.Set("scopes", claims.Scopes)
		c.Set("claims", claims)

		c.Next()
	}
}

// RateLimit 限流中间件
func RateLimit(limiter ratelimit.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			logger.Error("rate limit check failed", zap.Error(err))
			c.Next()
			return
		}

		if !allowed {
			response.TooManyRequests(c, "too many requests, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByUser 按用户限流
func RateLimitByUser(limiter ratelimit.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if userID, exists := c.Get("userID"); exists {
			key = userID.(string)
		}

		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			logger.Error("rate limit check failed", zap.Error(err))
			c.Next()
			return
		}

		if !allowed {
			response.TooManyRequests(c, "too many requests, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RBAC 基于角色的访问控制
func RBAC(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "access denied: role not found")
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

		response.Forbidden(c, "access denied: insufficient permissions")
		c.Abort()
	}
}

// RequireScopes 要求特定权限范围
func RequireScopes(requiredScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scopes, exists := c.Get("scopes")
		if !exists {
			response.Forbidden(c, "access denied: scopes not found")
			c.Abort()
			return
		}

		userScopes := scopes.([]string)
		scopeSet := make(map[string]bool)
		for _, s := range userScopes {
			scopeSet[s] = true
		}

		for _, required := range requiredScopes {
			if !scopeSet[required] {
				response.Forbidden(c, "access denied: missing required scope: "+required)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequestID 请求 ID 中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// Logger 日志中间件（增强版）
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		requestID, _ := c.Get("requestID")
		userID, _ := c.Get("userID")

		if raw != "" {
			path = path + "?" + raw
		}

		fields := []zap.Field{
			zap.String("request_id", requestID.(string)),
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
		}

		if userID != nil {
			fields = append(fields, zap.String("user_id", userID.(string)))
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.Errors("errors", c.Errors.Errors()))
			logger.Error("request failed", fields...)
		} else if statusCode >= 500 {
			logger.Error("server error", fields...)
		} else if statusCode >= 400 {
			logger.Warn("client error", fields...)
		} else {
			logger.Info("request", fields...)
		}
	}
}

// Recovery 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("requestID")
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("request_id", requestID.(string)),
				)
				response.InternalServerError(c, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// CORS CORS 中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, HEAD, PATCH, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeaders 安全头部中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
