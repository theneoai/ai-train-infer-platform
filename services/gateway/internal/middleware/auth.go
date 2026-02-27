package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/jwt"
	"github.com/plucky-groove3/ai-train-infer-platform/pkg/response"
)

const (
	ContextUserID = "user_id"
	ContextEmail  = "email"
	ContextRole   = "role"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "missing authorization header")
			c.Abort()
			return
		}

		// Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(c, http.StatusUnauthorized, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwt.ValidateToken(tokenString, jwtSecret)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRole, claims.Role)

		c.Next()
	}
}

func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			response.Error(c, http.StatusUnauthorized, "missing api key")
			c.Abort()
			return
		}

		// TODO: Validate API key against database
		// For MVP, we'll implement this in user service

		c.Next()
	}
}

// GetUserID gets user ID from context
func GetUserID(c *gin.Context) string {
	userID, _ := c.Get(ContextUserID)
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

// GetEmail gets email from context
func GetEmail(c *gin.Context) string {
	email, _ := c.Get(ContextEmail)
	if e, ok := email.(string); ok {
		return e
	}
	return ""
}
