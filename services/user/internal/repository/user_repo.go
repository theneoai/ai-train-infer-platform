package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ai-train-infer-platform/services/user/internal/domain"
	"github.com/ai-train-infer-platform/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrEmailExists    = errors.New("email already exists")
	ErrUsernameExists = errors.New("username already exists")
	ErrAPIKeyNotFound = errors.New("api key not found")
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	logger.Log.Debug("Creating user", zap.String("email", user.Email))

	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if isDuplicateKeyError(err) {
			if checkDuplicateField(err, "email") {
				logger.Log.Warn("Email already exists", zap.String("email", user.Email))
				return ErrEmailExists
			}
			if checkDuplicateField(err, "username") {
				logger.Log.Warn("Username already exists", zap.String("username", user.Username))
				return ErrUsernameExists
			}
		}
		logger.Log.Error("Failed to create user", zap.Error(err))
		return err
	}

	logger.Log.Info("User created successfully", zap.String("user_id", user.ID.String()))
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		logger.Log.Error("Failed to get user by ID", zap.String("user_id", id.String()), zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		logger.Log.Error("Failed to get user by email", zap.String("email", email), zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "username = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		logger.Log.Error("Failed to get user by username", zap.String("username", username), zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	logger.Log.Debug("Updating user", zap.String("user_id", user.ID.String()))

	result := r.db.WithContext(ctx).Model(user).Updates(map[string]interface{}{
		"email":      user.Email,
		"username":   user.Username,
		"role":       user.Role,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			if checkDuplicateField(result.Error, "email") {
				return ErrEmailExists
			}
			if checkDuplicateField(result.Error, "username") {
				return ErrUsernameExists
			}
		}
		logger.Log.Error("Failed to update user", zap.String("user_id", user.ID.String()), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	logger.Log.Info("User updated successfully", zap.String("user_id", user.ID.String()))
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	logger.Log.Debug("Deleting user", zap.String("user_id", id.String()))

	result := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id)
	if result.Error != nil {
		logger.Log.Error("Failed to delete user", zap.String("user_id", id.String()), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	logger.Log.Info("User deleted successfully", zap.String("user_id", id.String()))
	return nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *domain.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.APIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

type apiKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, apiKey *domain.APIKey) error {
	logger.Log.Debug("Creating API key", zap.String("user_id", apiKey.UserID.String()), zap.String("name", apiKey.Name))

	if err := r.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		logger.Log.Error("Failed to create API key", zap.String("user_id", apiKey.UserID.String()), zap.Error(err))
		return err
	}

	logger.Log.Info("API key created successfully", zap.String("api_key_id", apiKey.ID.String()), zap.String("user_id", apiKey.UserID.String()))
	return nil
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	var apiKey domain.APIKey
	if err := r.db.WithContext(ctx).First(&apiKey, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.APIKey, error) {
	var apiKeys []domain.APIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&apiKeys).Error; err != nil {
		logger.Log.Error("Failed to get API keys", zap.String("user_id", userID.String()), zap.Error(err))
		return nil, err
	}
	return apiKeys, nil
}

func (r *apiKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var apiKey domain.APIKey
	if err := r.db.WithContext(ctx).First(&apiKey, "key_hash = ?", keyHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	logger.Log.Debug("Deleting API key", zap.String("api_key_id", id.String()), zap.String("user_id", userID.String()))

	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&domain.APIKey{})
	if result.Error != nil {
		logger.Log.Error("Failed to delete API key", zap.String("api_key_id", id.String()), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrAPIKeyNotFound
	}

	logger.Log.Info("API key deleted successfully", zap.String("api_key_id", id.String()))
	return nil
}

func (r *apiKeyRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	logger.Log.Debug("Deleting all API keys for user", zap.String("user_id", userID.String()))

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&domain.APIKey{}).Error; err != nil {
		logger.Log.Error("Failed to delete API keys", zap.String("user_id", userID.String()), zap.Error(err))
		return err
	}
	return nil
}

func (r *apiKeyRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.APIKey{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

type TokenBlacklistRepository interface {
	Add(ctx context.Context, token string, expiresAt time.Time) error
	Exists(ctx context.Context, token string) (bool, error)
	CleanExpired(ctx context.Context) error
}

type tokenBlacklistRepository struct {
	db *gorm.DB
}

func NewTokenBlacklistRepository(db *gorm.DB) TokenBlacklistRepository {
	return &tokenBlacklistRepository{db: db}
}

func (r *tokenBlacklistRepository) Add(ctx context.Context, token string, expiresAt time.Time) error {
	blacklist := &domain.TokenBlacklist{
		Token:     token,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(blacklist).Error
}

func (r *tokenBlacklistRepository) Exists(ctx context.Context, token string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.TokenBlacklist{}).Where("token = ?", token).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *tokenBlacklistRepository) CleanExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&domain.TokenBlacklist{}).Error
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "duplicate") || contains(errStr, " Duplicate") || contains(errStr, "UNIQUE constraint")
}

func checkDuplicateField(err error, field string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, field)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
