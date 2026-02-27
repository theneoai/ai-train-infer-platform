package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken 无效的 Token
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken Token 已过期
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidIssuer 无效的签发者
	ErrInvalidIssuer = errors.New("invalid issuer")
)

// Config JWT 配置
type Config struct {
	SecretKey       string
	Issuer          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		SecretKey:       "your-secret-key-change-in-production",
		Issuer:          "aitip-gateway",
		AccessTokenTTL:  time.Hour * 2,
		RefreshTokenTTL: time.Hour * 24 * 7, // 7 days
	}
}

// Claims JWT 声明
type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Role   string   `json:"role"`
	OrgID  string   `json:"org_id"`
	Scopes []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair Token 对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// Manager JWT 管理器
type Manager struct {
	config *Config
}

// NewManager 创建 JWT 管理器
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	return &Manager{config: config}
}

// GenerateTokenPair 生成 Token 对
func (m *Manager) GenerateTokenPair(userID, email, name, role, orgID string, scopes []string) (*TokenPair, error) {
	accessToken, err := m.GenerateAccessToken(userID, email, name, role, orgID, scopes)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(m.config.AccessTokenTTL),
		TokenType:    "Bearer",
	}, nil
}

// GenerateAccessToken 生成访问令牌
func (m *Manager) GenerateAccessToken(userID, email, name, role, orgID string, scopes []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Name:   name,
		Role:   role,
		OrgID:  orgID,
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
			Subject:   userID,
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// GenerateRefreshToken 生成刷新令牌
func (m *Manager) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(m.config.RefreshTokenTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    m.config.Issuer,
		Subject:   userID,
		ID:        uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ParseToken 解析 Token
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// 验证 Issuer
		if claims.Issuer != m.config.Issuer {
			return nil, ErrInvalidIssuer
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// ParseRefreshToken 解析刷新令牌
func (m *Manager) ParseRefreshToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		subject, ok := claims["sub"].(string)
		if !ok {
			return "", ErrInvalidToken
		}
		return subject, nil
	}

	return "", ErrInvalidToken
}

// RefreshTokens 刷新令牌
func (m *Manager) RefreshTokens(refreshToken string, userInfo map[string]string, scopes []string) (*TokenPair, error) {
	userID, err := m.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return m.GenerateTokenPair(
		userID,
		userInfo["email"],
		userInfo["name"],
		userInfo["role"],
		userInfo["org_id"],
		scopes,
	)
}

// ValidateToken 验证 Token 是否有效
func (m *Manager) ValidateToken(tokenString string) error {
	_, err := m.ParseToken(tokenString)
	return err
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *Config {
	return m.config
}

// SetConfig 设置配置
func (m *Manager) SetConfig(config *Config) {
	m.config = config
}
