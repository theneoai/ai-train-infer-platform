package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ai-train-infer-platform/services/user/internal/domain"
	"github.com/ai-train-infer-platform/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailExists 邮箱已存在
	ErrEmailExists = errors.New("email already exists")
	// ErrUsernameExists 用户名已存在
	ErrUsernameExists = errors.New("username already exists")
	// ErrAPIKeyNotFound API Key 不存在
	ErrAPIKeyNotFound = errors.New("api key not found")
)

// UserRepository 用户仓库接口
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

// userRepository 用户仓库实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	logger.Debug("Creating user", logger.WithField("email", user.Email).Fields...)

	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		// 检查唯一性冲突
		if isDuplicateKeyError(err) {
			if checkDuplicateField(err, "email") {
				logger.Warn("Email already exists", logger.WithField("email", user.Email).Fields...)
				return ErrEmailExists
			}
			if checkDuplicateField(err, "username") {
				logger.Warn("Username already exists", logger.WithField("username", user.Username).Fields...)
				return ErrUsernameExists
			}
		}
		logger.Error("Failed to create user", logger.WithField("error", err.Error()).Fields...)
		return err
	}

	logger.Info("User created successfully", logger.WithField("user_id", user.ID.String()).Fields...)
	return nil
}

// GetByID 根据 ID 获取用户
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to get user by ID", 
			logger.WithField("user_id", id.String()).Fields...,
			logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to get user by email", 
			logger.WithField("email", email).Fields...,
			logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "username = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to get user by username", 
			logger.WithField("username", username).Fields...,
			logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	logger.Debug("Updating user", logger.WithField("user_id", user.ID.String()).Fields...)

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
		logger.Error("Failed to update user", 
			logger.WithField("user_id", user.ID.String()).Fields...,
			logger.WithField("error", result.Error.Error()).Fields...)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	logger.Info("User updated successfully", logger.WithField("user_id", user.ID.String()).Fields...)
	return nil
}

// Delete 删除用户
func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	logger.Debug("Deleting user", logger.WithField("user_id", id.String()).Fields...)

	result := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id)
	if result.Error != nil {
		logger.Error("Failed to delete user", 
			logger.WithField("user_id", id.String()).Fields...,
			logger.WithField("error", result.Error.Error()).Fields...)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	logger.Info("User deleted successfully", logger.WithField("user_id", id.String()).Fields...)
	return nil
}

// ExistsByEmail 检查邮箱是否存在
func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByUsername 检查用户名是否存在
func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// APIKeyRepository API Key 仓库接口
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *domain.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.APIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// apiKeyRepository API Key 仓库实现
type apiKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository 创建 API Key 仓库
func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

// Create 创建 API Key
func (r *apiKeyRepository) Create(ctx context.Context, apiKey *domain.APIKey) error {
	logger.Debug("Creating API key", 
		logger.WithField("user_id", apiKey.UserID.String()).Fields...,
		logger.WithField("name", apiKey.Name).Fields...)

	if err := r.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		logger.Error("Failed to create API key", 
			logger.WithField("user_id", apiKey.UserID.String()).Fields...,
			logger.WithField("error", err.Error()).Fields...)
		return err
	}

	logger.Info("API key created successfully", 
		logger.WithField("api_key_id", apiKey.ID.String()).Fields...,
		logger.WithField("user_id", apiKey.UserID.String()).Fields...)
	return nil
}

// GetByID 根据 ID 获取 API Key
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

// GetByUserID 获取用户的所有 API Key
func (r *apiKeyRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.APIKey, error) {
	var apiKeys []domain.APIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&apiKeys).Error; err != nil {
		logger.Error("Failed to get API keys by user ID", 
			logger.WithField("user_id", userID.String()).Fields...,
			logger.WithField("error", err.Error()).Fields...)
		return nil, err
	}
	return apiKeys, nil
}

// GetByKeyHash 根据 Key Hash 获取 API Key
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

// Delete 删除 API Key
func (r *apiKeyRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	logger.Debug("Deleting API key", 
		logger.WithField("api_key_id", id.String()).Fields...,
		logger.WithField("user_id", userID.String()).Fields...)

	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&domain.APIKey{})
	if result.Error != nil {
		logger.Error("Failed to delete API key", 
			logger.WithField("api_key_id", id.String()).Fields...,
			logger.WithField("error", result.Error.Error()).Fields...)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrAPIKeyNotFound
	}

	logger.Info("API key deleted successfully", logger.WithField("api_key_id", id.String()).Fields...)
	return nil
}

// DeleteByUserID 删除用户的所有 API Key
func (r *apiKeyRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	logger.Debug("Deleting all API keys for user", logger.WithField("user_id", userID.String()).Fields...)

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&domain.APIKey{}).Error; err != nil {
		logger.Error("Failed to delete API keys by user ID", 
			logger.WithField("user_id", userID.String()).Fields...,
			logger.WithField("error", err.Error()).Fields...)
		return err
	}

	return nil
}

// CountByUserID 统计用户的 API Key 数量
func (r *apiKeyRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.APIKey{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// TokenBlacklistRepository Token 黑名单仓库接口
type TokenBlacklistRepository interface {
	Add(ctx context.Context, token string, expiresAt time.Time) error
	Exists(ctx context.Context, token string) (bool, error)
	CleanExpired(ctx context.Context) error
}

// tokenBlacklistRepository Token 黑名单仓库实现
type tokenBlacklistRepository struct {
	db *gorm.DB
}

// NewTokenBlacklistRepository 创建 Token 黑名单仓库
func NewTokenBlacklistRepository(db *gorm.DB) TokenBlacklistRepository {
	return &tokenBlacklistRepository{db: db}
}

// Add 添加 Token 到黑名单
func (r *tokenBlacklistRepository) Add(ctx context.Context, token string, expiresAt time.Time) error {
	blacklist := &domain.TokenBlacklist{
		Token:     token,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(blacklist).Error
}

// Exists 检查 Token 是否在黑名单中
func (r *tokenBlacklistRepository) Exists(ctx context.Context, token string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.TokenBlacklist{}).Where("token = ?", token).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CleanExpired 清理过期的 Token
func (r *tokenBlacklistRepository) CleanExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&domain.TokenBlacklist{}).Error
}

// 辅助函数

// isDuplicateKeyError 检查是否是唯一性冲突错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// 简单的字符串匹配，实际项目中可能需要根据数据库类型做更精确的判断
	errStr := err.Error()
	return contains(errStr, "duplicate") || contains(errStr, " Duplicate") || contains(errStr, "UNIQUE constraint")
}

// checkDuplicateField 检查是否是特定字段的重复
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
