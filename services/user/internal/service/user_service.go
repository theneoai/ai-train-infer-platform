package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/ai-train-infer-platform/services/user/internal/domain"
	"github.com/ai-train-infer-platform/services/user/internal/repository"
	"github.com/ai-train-infer-platform/pkg/jwt"
	"github.com/ai-train-infer-platform/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenBlacklisted   = errors.New("token has been revoked")
	ErrAPIKeyNotFound     = errors.New("api key not found")
	ErrMaxAPIKeysReached  = errors.New("maximum number of API keys reached")
)

const (
	MaxAPIKeysPerUser = 10
	APIKeyLength      = 32
)

type UserService interface {
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error)
	Logout(ctx context.Context, token string, expiresAt time.Time) error
	RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error)
	ValidateToken(ctx context.Context, token string) (*jwt.Claims, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.UserProfile, error)
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error)
	CreateAPIKey(ctx context.Context, userID uuid.UUID, req *domain.CreateAPIKeyRequest) (*domain.APIKeyWithPlainText, error)
	ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]domain.APIKeyResponse, error)
	DeleteAPIKey(ctx context.Context, userID uuid.UUID, keyID uuid.UUID) error
	ValidateAPIKey(ctx context.Context, key string) (*domain.APIKey, error)
}

type userService struct {
	userRepo      repository.UserRepository
	apiKeyRepo    repository.APIKeyRepository
	blacklistRepo repository.TokenBlacklistRepository
	jwtManager    *jwt.Manager
}

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

func (s *userService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	logger.Log.Info("Processing user registration", zap.String("email", req.Email))

	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		logger.Log.Error("Failed to check email existence", zap.Error(err))
		return nil, err
	}
	if exists {
		logger.Log.Warn("Email already registered", zap.String("email", req.Email))
		return nil, ErrUserAlreadyExists
	}

	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		logger.Log.Error("Failed to check username existence", zap.Error(err))
		return nil, err
	}
	if exists {
		logger.Log.Warn("Username already taken", zap.String("username", req.Username))
		return nil, ErrUserAlreadyExists
	}

	user := &domain.User{
		Email:    req.Email,
		Username: req.Username,
		Role:     "user",
	}

	if err := user.SetPassword(req.Password); err != nil {
		logger.Log.Error("Failed to hash password", zap.Error(err))
		return nil, err
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrEmailExists) || errors.Is(err, repository.ErrUsernameExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	logger.Log.Info("User registered successfully", zap.String("user_id", user.ID.String()))
	return s.generateAuthResponse(user)
}

func (s *userService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	logger.Log.Info("Processing user login", zap.String("email", req.Email))

	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			logger.Log.Warn("Login attempt with non-existent email", zap.String("email", req.Email))
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.CheckPassword(req.Password) {
		logger.Log.Warn("Login attempt with invalid password", zap.String("email", req.Email))
		return nil, ErrInvalidCredentials
	}

	logger.Log.Info("User logged in successfully", zap.String("user_id", user.ID.String()))
	return s.generateAuthResponse(user)
}

func (s *userService) Logout(ctx context.Context, token string, expiresAt time.Time) error {
	logger.Log.Info("Processing user logout")

	if err := s.blacklistRepo.Add(ctx, token, expiresAt); err != nil {
		logger.Log.Error("Failed to add token to blacklist", zap.Error(err))
		return err
	}

	logger.Log.Info("User logged out successfully")
	return nil
}

func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error) {
	logger.Log.Info("Processing token refresh")

	userID, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrExpiredToken) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

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

	logger.Log.Info("Token refreshed successfully", zap.String("user_id", user.ID.String()))
	return s.generateAuthResponse(user)
}

func (s *userService) ValidateToken(ctx context.Context, token string) (*jwt.Claims, error) {
	exists, err := s.blacklistRepo.Exists(ctx, token)
	if err != nil {
		logger.Log.Error("Failed to check token blacklist", zap.Error(err))
		return nil, err
	}
	if exists {
		return nil, ErrTokenBlacklisted
	}

	claims, err := s.jwtManager.ParseToken(token)
	if err != nil {
		if errors.Is(err, jwt.ErrExpiredToken) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	return claims, nil
}

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

func (s *userService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error) {
	return s.GetUserByID(ctx, userID)
}

func (s *userService) CreateAPIKey(ctx context.Context, userID uuid.UUID, req *domain.CreateAPIKeyRequest) (*domain.APIKeyWithPlainText, error) {
	logger.Log.Info("Creating API key", zap.String("user_id", userID.String()), zap.String("name", req.Name))

	count, err := s.apiKeyRepo.CountByUserID(ctx, userID)
	if err != nil {
		logger.Log.Error("Failed to count API keys", zap.Error(err))
		return nil, err
	}
	if count >= MaxAPIKeysPerUser {
		logger.Log.Warn("Maximum API keys reached", zap.String("user_id", userID.String()))
		return nil, ErrMaxAPIKeysReached
	}

	plainKey, err := generateAPIKey()
	if err != nil {
		logger.Log.Error("Failed to generate API key", zap.Error(err))
		return nil, err
	}

	apiKey := &domain.APIKey{
		UserID: userID,
		Name:   req.Name,
	}
	if err := apiKey.SetKey(plainKey); err != nil {
		logger.Log.Error("Failed to hash API key", zap.Error(err))
		return nil, err
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	logger.Log.Info("API key created successfully", zap.String("api_key_id", apiKey.ID.String()), zap.String("user_id", userID.String()))

	return &domain.APIKeyWithPlainText{
		ID:        apiKey.ID,
		UserID:    apiKey.UserID,
		Name:      apiKey.Name,
		Key:       plainKey,
		CreatedAt: apiKey.CreatedAt,
	}, nil
}

func (s *userService) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]domain.APIKeyResponse, error) {
	logger.Log.Debug("Listing API keys", zap.String("user_id", userID.String()))

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

func (s *userService) DeleteAPIKey(ctx context.Context, userID uuid.UUID, keyID uuid.UUID) error {
	logger.Log.Info("Deleting API key", zap.String("api_key_id", keyID.String()), zap.String("user_id", userID.String()))

	if err := s.apiKeyRepo.Delete(ctx, keyID, userID); err != nil {
		if errors.Is(err, repository.ErrAPIKeyNotFound) {
			return ErrAPIKeyNotFound
		}
		return err
	}

	return nil
}

func (s *userService) ValidateAPIKey(ctx context.Context, key string) (*domain.APIKey, error) {
	return nil, errors.New("not implemented")
}

func (s *userService) generateAuthResponse(user *domain.User) (*domain.AuthResponse, error) {
	tokenPair, err := s.jwtManager.GenerateTokenPair(
		user.ID.String(),
		user.Email,
		user.Username,
		user.Role,
		"",
		nil,
	)
	if err != nil {
		logger.Log.Error("Failed to generate token pair", zap.Error(err))
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

func generateAPIKey() (string, error) {
	bytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func MapServiceError(err error) int {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, ErrUserAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrTokenExpired), errors.Is(err, ErrTokenBlacklisted):
		return http.StatusUnauthorized
	case errors.Is(err, ErrAPIKeyNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrMaxAPIKeysReached):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
