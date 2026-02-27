package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/ai-train-infer-platform/services/user/internal/domain"
	"github.com/ai-train-infer-platform/services/user/internal/service"
	"github.com/ai-train-infer-platform/pkg/jwt"
	"github.com/ai-train-infer-platform/pkg/logger"
	"github.com/ai-train-infer-platform/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthHandler struct {
	userService service.UserService
	jwtManager  *jwt.Manager
}

func NewAuthHandler(userService service.UserService, jwtManager *jwt.Manager) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtManager:  jwtManager,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("Invalid register request", zap.Error(err))
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.userService.Register(c.Request.Context(), &req)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("Invalid login request", zap.Error(err))
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	profile, err := h.userService.GetCurrentUser(c.Request.Context(), id)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, profile)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Error(c, http.StatusUnauthorized, "authorization header required")
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		response.Error(c, http.StatusUnauthorized, "invalid authorization header format")
		return
	}

	token := parts[1]

	claims, err := h.jwtManager.ParseToken(token)
	if err != nil {
		logger.Log.Warn("Failed to parse token during logout", zap.Error(err))
		response.Success(c, gin.H{"message": "logged out"})
		return
	}

	expiresAt := time.Unix(claims.ExpiresAt.Unix(), 0)
	
	if err := h.userService.Logout(c.Request.Context(), token, expiresAt); err != nil {
		logger.Log.Error("Failed to logout", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "failed to logout")
		return
	}

	response.Success(c, gin.H{"message": "logged out successfully"})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req domain.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.userService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, resp)
}

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	profile, err := h.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, profile)
}

type APIKeyHandler struct {
	userService service.UserService
}

func NewAPIKeyHandler(userService service.UserService) *APIKeyHandler {
	return &APIKeyHandler{userService: userService}
}

func (h *APIKeyHandler) Create(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req domain.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	apiKey, err := h.userService.CreateAPIKey(c.Request.Context(), id, &req)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, apiKey)
}

func (h *APIKeyHandler) List(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	apiKeys, err := h.userService.ListAPIKeys(c.Request.Context(), id)
	if err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, apiKeys)
}

func (h *APIKeyHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid api key id")
		return
	}

	if err := h.userService.DeleteAPIKey(c.Request.Context(), uid, keyID); err != nil {
		status := service.MapServiceError(err)
		response.Error(c, status, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "api key deleted successfully"})
}
