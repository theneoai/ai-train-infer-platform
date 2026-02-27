package domain

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户领域模型
type User struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email        string         `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	Username     string         `json:"username" gorm:"type:varchar(100);uniqueIndex;not null"`
	PasswordHash string         `json:"-" gorm:"type:varchar(255);not null"`
	Role         string         `json:"role" gorm:"type:varchar(50);default:'user'"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	APIKeys      []APIKey       `json:"api_keys,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// SetPassword 设置密码（bcrypt 加密）
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// ToProfile 转换为用户资料
func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}

// UserProfile 用户资料（公开信息）
type UserProfile struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKey API Key 领域模型
type APIKey struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Name      string    `json:"name" gorm:"type:varchar(255)"`
	KeyHash   string    `json:"-" gorm:"type:varchar(255);uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}

// SetKey 设置 API Key（存储哈希）
func (k *APIKey) SetKey(key string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	k.KeyHash = string(hash)
	return nil
}

// CheckKey 验证 API Key
func (k *APIKey) CheckKey(key string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(key))
	return err == nil
}

// APIKeyWithPlainText 包含明文 API Key 的响应
type APIKeyWithPlainText struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"` // 明文 key，仅创建时返回
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name string `json:"name" binding:"required,max=255"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	User         *UserProfile `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	TokenType    string       `json:"token_type"`
}

// APIKeyResponse API Key 响应
type APIKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenBlacklist Token 黑名单（用于登出）
type TokenBlacklist struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Token     string    `json:"-" gorm:"type:text;not null;index"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (TokenBlacklist) TableName() string {
	return "token_blacklist"
}
