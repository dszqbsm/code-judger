package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWTClaims JWT载荷结构
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TokenID  string `json:"jti"` // JWT ID，用于令牌撤销
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpire  int64
	RefreshExpire int64
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(accessSecret, refreshSecret string, accessExpire, refreshExpire int64) *JWTManager {
	return &JWTManager{
		AccessSecret:  accessSecret,
		RefreshSecret: refreshSecret,
		AccessExpire:  accessExpire,
		RefreshExpire: refreshExpire,
	}
}

// GenerateTokens 生成访问令牌和刷新令牌
func (j *JWTManager) GenerateTokens(userID int64, username, role string) (accessToken, refreshToken, tokenID string, err error) {
	// 生成唯一的令牌ID
	tokenID, err = GenerateTokenID()
	if err != nil {
		return "", "", "", err
	}

	now := time.Now()

	// 生成访问令牌
	accessClaims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.AccessExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "oj-system",
			Subject:   "access-token",
		},
	}

	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(j.AccessSecret))
	if err != nil {
		return "", "", "", err
	}

	// 生成刷新令牌
	refreshClaims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.RefreshExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "oj-system",
			Subject:   "refresh-token",
		},
	}

	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(j.RefreshSecret))
	if err != nil {
		return "", "", "", err
	}

	return accessToken, refreshToken, tokenID, nil
}

// ParseAccessToken 解析访问令牌
func (j *JWTManager) ParseAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.AccessSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ParseRefreshToken 解析刷新令牌
func (j *JWTManager) ParseRefreshToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.RefreshSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

// ValidateToken 验证令牌是否有效
func (j *JWTManager) ValidateToken(tokenString string, isRefreshToken bool) (*JWTClaims, error) {
	if isRefreshToken {
		return j.ParseRefreshToken(tokenString)
	}
	return j.ParseAccessToken(tokenString)
}