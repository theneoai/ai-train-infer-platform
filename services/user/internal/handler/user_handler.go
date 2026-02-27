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
)

// AuthHandler 认证处理器
type AuthHandler struct {
	userService service.UserService
	jwtManager  *jwt.Manager
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(userService service.UserService, jwtManager *jwt.Manager) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtManager:  jwtManager,
	}
}

// Register 注册
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterRequest true "注册信息"
// @Success 200 {object} domain.AuthResponse
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid register request", logger.WithField("error", err.Error()).Fields...)
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.userService.Register(c.Request.Context(), &req)
	if err != nil {
		code := service.MapServiceError(err)
		msg := service.GetErrorMessage(err)
		response.ErrorWithMessage(c, code, msg)
		return
	}

	response.Success(c, resp)
}

// Login 登录
// @Summary 用户登录
// @Description 用户登录并获取 JWT Token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "登录信息"
// @Success 200 {object} domain.AuthResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid login request", logger.WithField("error", err.Error()).Fields...)
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		code := service.MapServiceError(err)
		msg := service.GetErrorMessage(err)
		response.ErrorWithMessage(c, code, msg)
		return
	}

	response.Success(c, resp)
}

// Me 获取当前用户
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {object} domain.UserProfile
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID.(string))
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInvalidParams, "invalid user id")
		return
	}

	profile, err := h.userService.GetCurrentUser(c.Request.Context(), id)
	if err != nil {
		code := service.MapServiceError(err)
		msg := service.GetErrorMessage(err)
		response.ErrorWithMessage(c, code, msg)
		return
	}

	response.Success(c, profile)
}

// Logout 登出
// @Summary 用户登出
// @Description 登出并使当前 Token 失效
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// 获取 Token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c, "authorization header required")
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		response.Unauthorized(c, "invalid authorization header format")
		return
	}

	token := parts[1]

	// 解析 Token 获取过期时间
	claims, err := h.jwtManager.ParseToken(token)
	if err != nil {
		// 即使 Token 解析失败，也返回成功，因为客户端想要登出
		logger.Warn("Failed to parse token during logout", logger.WithField("error", err.Error()).Fields...)
		response.Success(c, gin.H{"message": "logged out"})
		return
	}

	expiresAt := time.Unix(claims.ExpiresAt.Unix(), 0)
	
	if err := h.userService.Logout(c.Request.Context(), token, expiresAt); err != nil {
		logger.Error("Failed to logout", logger.WithField("error", err.Error()).Fields...)
		response.InternalServerError(c, "failed to logout")
		return
	}

	response.Success(c, gin.H{"message": "logged out successfully"})
}

// RefreshToken 刷新 Token
// @Summary 刷新 Access Token
// @Description 使用 Refresh Token 获取新的 Access Token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RefreshTokenRequest true "刷新 Token 请求"
// @Success 200 {object} domain.AuthResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req domain.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.userService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		code := service.MapServiceError(err)
		msg := service.GetErrorMessage(err)
		response.ErrorWithMessage(c, code, msg)
		return
	}

	response.Success(c, resp)
}

// UserHandler 用户处理器
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile 获取用户资料
// @Summary 获取用户资料
// @Description 获取当前登录用户的详细资料
// @Tags users
// @Produce json
// @Security Bearer
// @Success 200 {object} domain.UserProfile
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID.(string))
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInvalidParams, "invalid user id")
		return
	}

	profile, err := h.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		code := service.MapServiceError(err)
		msg := service.GetErrorMessage(err)
		response.ErrorWithMessage(c, code, msg)
		return
	}

	response.Success(c, profile)
}

// APIKeyHandler API Key 处理器
type APIKeyHandler struct {
	userService service.UserService
}

// NewAPIKeyHandler 创建 API Key 处理器
func NewAPIKeyHandler(userService service.UserService) *APIKeyHandler {
	return &APIKeyHandler{userService: userService}
}

// Create 创建 API Key
// @Summary 创建 API Key
// @Description 为当前用户创建新的 API Key
// @Tags api-keys
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body domain.CreateAPIKeyRequest true "创建 API Key 请求"
// @Success 200 {object} domain.APIKeyWithPlainText
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/api-keys [post]
func (h *APIKeyHandler) Create(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID.(string))
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInvalidParams, "invalid user id")
		return
	}

	var req domain.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	apiKey, err := h.userService.CreateAPIKey(c.Request.Context(), id, &req)
	if err != nil {
		code := service.MapServiceError(err)
		msg := service.GetErrorMessage(err)
		response.ErrorWithMessage(c, code, msg)
		return
	}

	response.Success(c, apiKey)
}

// List 列出 API Keys
// @Summary 列出 API Keys
// @Description 获取当前用户的所有 API Keys
// @Tags api-keys
// @Produce json
// @Security Bearer
// @Success 200 {array} domain.APIKeyResponse
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/api-keys [get]
func (h *APIKeyHandler) List(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID.(string))
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInvalidParams, "invalid user id")
		return
	}

	apiKeys, err := h.userService.ListAPIKeys(c.Request.Context(), id)
	if err != nil {
		code := service.MapServiceError(err)
		msg := service.GetErrorMessage(err)
		response.ErrorWithMessage(c, code, msg)
		return
	}

	response.Success(c, apiKeys)
}

// Delete 删除 API Key
// @Summary 删除 API Key
// @Description 删除指定的 API Key
// @Tags api-keys
// @Produce json
// @Security Bearer
// @Param id path string true "API Key ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/api-keys/{id} [delete]
func (h *APIKeyHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInvalidParams, "invalid user id")
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.ErrorWithMessage(c, response.ErrorInvalidParams, "invalid api key id")
		return
	}

	if err := h.userService.DeleteAPIKey(c.Request.Context(), uid, keyID); err != nil {
		code := service.MapServiceError(err)
		msg := service.GetErrorMessage(err)
		response.ErrorWithMessage(c, code, msg)
		return
	}

	response.Success(c, gin.H{"message": "api key deleted successfully"})
}
