package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/ai-train-infer-platform/services/user/internal/domain"
	"github.com/ai-train-infer-platform/services/user/internal/repository"
	"github.com/ai-train-infer-platform/pkg/jwt"
	"github.com/ai-train-infer-platform/pkg/logger"
	"github.com/ai-train-infer-platform/pkg/response"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCredentials 无效的凭据
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserAlreadyExists 用户已存在
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidToken 无效的 Token
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired Token 已过期
	ErrTokenExpired = errors.New("token has expired")
	// ErrTokenBlacklisted Token 已被列入黑名单
	ErrTokenBlacklisted = errors.New("token has been revoked")
	// ErrAPIKeyNotFound API Key 不存在
	ErrAPIKeyNotFound = errors.New("api key not found")
	// ErrMaxAPIKeysReached 达到最大 API Key 数量限制
	ErrMaxAPIKeysReached = errors.New("maximum number of API keys reached")
)

const (
	// MaxAPIKeysPerUser 每个用户最大 API Key 数量
	MaxAPIKeysPerUser = 10
	// APIKeyLength API Key 长度
	APIKeyLength = 32
)

// UserService 用户服务接口
type UserService interface {
	// 认证相关
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error)
	Logout(ctx context.Context, token string, expiresAt time.Time) error
	RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error)
	ValidateToken(ctx context.Context, token string) (*jwt.Claims, error)
	
	// 用户相关
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.UserProfile, error)
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error)
	
	// API Key 相关
	CreateAPIKey(ctx context.Context, userID uuid.UUID, req *domain.CreateAPIKeyRequest) (*domain.APIKeyWithPlainText, error)
	ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]domain.APIKeyResponse, error)
	DeleteAPIKey(ctx context.Context, userID uuid.UUID, keyID uuid.UUID) error
	ValidateAPIKey(ctx context.Context, key string) (*domain.APIKey, error)
}

// userService 用户服务实现
type userService struct {
	userRepo    repository.UserRepository
	apiKeyRepo  repository.APIKeyRepository
	blacklistRepo repository.TokenBlacklistRepository
	jwtManager  *jwt.Manager
}

// NewUserService 创建用户服务
func NewUserService(
	userRepo repository.UserRepository,
	apiKeyRepo repository.APIKeyRepository,
	blacklistRepo repository.TokenBlacklistRepository,
	jwtManager *jwt.Manager,
) UserService {
	return &userService{
		userRepo:      userRepo,
		apiKeyRepo:    apiKeyRepo,
		blacklistRepo: blacklistRepo,
		jwtManager:    jwtManager,
	}
}

// Register 用户注册
func (s *userService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	logger.Info("Processing user registration", logger.WithField("email", req.Email).Fields...)

	// 检查邮箱是否已存在
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		logger.Error("Failed to check email existence", logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}
	if exists {
		logger.Warn("Email already registered", logger.WithField("email", req.Email).Fields...)
		return nil, ErrUserAlreadyExists
	}

	// 检查用户名是否已存在
	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		logger.Error("Failed to check username existence", logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}
	if exists {
		logger.Warn("Username already taken", logger.WithField("username", req.Username).Fields...)
		return nil, ErrUserAlreadyExists
	}

	// 创建用户
	user := &domain.User{
		Email:    req.Email,
		Username: req.Username,
		Role:     "user",
	}

	// 设置密码
	if err := user.SetPassword(req.Password); err != nil {
		logger.Error("Failed to hash password", logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}

	// 保存用户
	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrEmailExists) || errors.Is(err, repository.ErrUsernameExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	logger.Info("User registered successfully", logger.WithField("user_id", user.ID.String()).Fields...)

	// 生成 Token
	return s.generateAuthResponse(user)
}

// Login 用户登录
func (s *userService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	logger.Info("Processing user login", logger.WithField("email", req.Email).Fields...)

	// 查找用户
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			logger.Warn("Login attempt with non-existent email", logger.WithField("email", req.Email).Fields...)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// 验证密码
	if !user.CheckPassword(req.Password) {
		logger.Warn("Login attempt with invalid password", logger.WithField("email", req.Email).Fields...)
		return nil, ErrInvalidCredentials
	}

	logger.Info("User logged in successfully", logger.WithField("user_id", user.ID.String()).Fields...)

	// 生成 Token
	return s.generateAuthResponse(user)
}

// Logout 用户登出
func (s *userService) Logout(ctx context.Context, token string, expiresAt time.Time) error {
	logger.Info("Processing user logout")

	// 将 Token 加入黑名单
	if err := s.blacklistRepo.Add(ctx, token, expiresAt); err != nil {
		logger.Error("Failed to add token to blacklist", logger.WithField("error", err.Error()).Fields...)
		return err
	}

	logger.Info("User logged out successfully")
	return nil
}

// RefreshToken 刷新 Token
func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error) {
	logger.Info("Processing token refresh")

	// 解析 Refresh Token
	userID, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrExpiredToken) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	// 查找用户
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	logger.Info("Token refreshed successfully", logger.WithField("user_id", user.ID.String()).Fields...)

	// 生成新的 Token
	return s.generateAuthResponse(user)
}

// ValidateToken 验证 Token
func (s *userService) ValidateToken(ctx context.Context, token string) (*jwt.Claims, error) {
	// 检查 Token 是否在黑名单中
	exists, err := s.blacklistRepo.Exists(ctx, token)
	if err != nil {
		logger.Error("Failed to check token blacklist", logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}
	if exists {
		return nil, ErrTokenBlacklisted
	}

	// 解析 Token
	claims, err := s.jwtManager.ParseToken(token)
	if err != nil {
		if errors.Is(err, jwt.ErrExpiredToken) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUserByID 根据 ID 获取用户
func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.UserProfile, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user.ToProfile(), nil
}

// GetCurrentUser 获取当前用户
func (s *userService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error) {
	return s.GetUserByID(ctx, userID)
}

// CreateAPIKey 创建 API Key
func (s *userService) CreateAPIKey(ctx context.Context, userID uuid.UUID, req *domain.CreateAPIKeyRequest) (*domain.APIKeyWithPlainText, error) {
	logger.Info("Creating API key", 
		logger.WithField("user_id", userID.String()).Fields...,
		logger.WithField("name", req.Name).Fields...)

	// 检查 API Key 数量限制
	count, err := s.apiKeyRepo.CountByUserID(ctx, userID)
	if err != nil {
		logger.Error("Failed to count API keys", logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}
	if count >= MaxAPIKeysPerUser {
		logger.Warn("Maximum API keys reached", logger.WithField("user_id", userID.String()).Fields...)
		return nil, ErrMaxAPIKeysReached
	}

	// 生成随机 API Key
	plainKey, err := generateAPIKey()
	if err != nil {
		logger.Error("Failed to generate API key", logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}

	// 创建 API Key
	apiKey := &domain.APIKey{
		UserID: userID,
		Name:   req.Name,
	}
	if err := apiKey.SetKey(plainKey); err != nil {
		logger.Error("Failed to hash API key", logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	logger.Info("API key created successfully", 
		logger.WithField("api_key_id", apiKey.ID.String()).Fields...,
		logger.WithField("user_id", userID.String()).Fields...)

	return &domain.APIKeyWithPlainText{
		ID:        apiKey.ID,
		UserID:    apiKey.UserID,
		Name:      apiKey.Name,
		Key:       plainKey, // 明文 key，仅返回这一次
		CreatedAt: apiKey.CreatedAt,
	}, nil
}

// ListAPIKeys 列出用户的 API Keys
func (s *userService) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]domain.APIKeyResponse, error) {
	logger.Debug("Listing API keys", logger.WithField("user_id", userID.String()).Fields...)

	apiKeys, err := s.apiKeyRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]domain.APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		responses[i] = domain.APIKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			CreatedAt: key.CreatedAt,
		}
	}

	return responses, nil
}

// DeleteAPIKey 删除 API Key
func (s *userService) DeleteAPIKey(ctx context.Context, userID uuid.UUID, keyID uuid.UUID) error {
	logger.Info("Deleting API key", 
		logger.WithField("api_key_id", keyID.String()).Fields...,
		logger.WithField("user_id", userID.String()).Fields...)

	if err := s.apiKeyRepo.Delete(ctx, keyID, userID); err != nil {
		if errors.Is(err, repository.ErrAPIKeyNotFound) {
			return ErrAPIKeyNotFound
		}
		return err
	}

	return nil
}

// ValidateAPIKey 验证 API Key
func (s *userService) ValidateAPIKey(ctx context.Context, key string) (*domain.APIKey, error) {
	// 这里需要遍历用户的所有 API Keys 来验证
	// 在实际生产环境中，可以考虑使用缓存优化
	return nil, errors.New("not implemented")
}

// 辅助方法

// generateAuthResponse 生成认证响应
func (s *userService) generateAuthResponse(user *domain.User) (*domain.AuthResponse, error) {
	tokenPair, err := s.jwtManager.GenerateTokenPair(
		user.ID.String(),
		user.Email,
		user.Username,
		user.Role,
		"", // org_id，暂不使用
		nil, // scopes，暂不使用
	)
	if err != nil {
		logger.Error("Failed to generate token pair", logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}

	return &domain.AuthResponse{
		User:         user.ToProfile(),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    tokenPair.TokenType,
	}, nil
}

// generateAPIKey 生成随机 API Key
func generateAPIKey() (string, error) {
	bytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// MapServiceError 将服务错误映射为响应错误码
func MapServiceError(err error) response.ResponseCode {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return response.ErrorUnauthorized
	case errors.Is(err, ErrUserAlreadyExists):
		return response.ErrorInvalidParams
	case errors.Is(err, ErrUserNotFound):
		return response.ErrorNotFound
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrTokenExpired), errors.Is(err, ErrTokenBlacklisted):
		return response.ErrorUnauthorized
	case errors.Is(err, ErrAPIKeyNotFound):
		return response.ErrorNotFound
	case errors.Is(err, ErrMaxAPIKeysReached):
		return response.ErrorInvalidParams
	default:
		return response.ErrorInternalServer
	}
}

// GetErrorMessage 获取错误信息
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
