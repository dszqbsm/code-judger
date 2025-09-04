package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type AuthMiddleware struct {
	secret      string
	redisClient *redis.Redis
}

type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TokenID  string `json:"jti"`
	jwt.RegisteredClaims
}

type UserInfo struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TokenID  string `json:"token_id"`
}

func NewAuthMiddleware(secret string, redisClient *redis.Redis) AuthMiddleware {
	return AuthMiddleware{
		secret:      secret,
		redisClient: redisClient,
	}
}

func (m AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 提取Authorization头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":401,"message":"缺少认证令牌"}`))
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":401,"message":"无效的令牌格式"}`))
			return
		}

		// 提取令牌
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 解析JWT令牌
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.secret), nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":401,"message":"无效的令牌"}`))
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":401,"message":"无效的令牌声明"}`))
			return
		}

		// 检查令牌是否被撤销
		tokenKey := "revoked_token:" + claims.TokenID
		exists, err := m.redisClient.Exists(tokenKey)
		if err == nil && exists {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":401,"message":"令牌已被撤销"}`))
			return
		}

		// 将用户信息添加到上下文
		userInfo := &UserInfo{
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
			TokenID:  claims.TokenID,
		}

		ctx := context.WithValue(r.Context(), "user", userInfo)
		next(w, r.WithContext(ctx))
	}
}

// GetUserFromContext 从上下文中获取用户信息
func GetUserFromContext(ctx context.Context) (*UserInfo, bool) {
	user, ok := ctx.Value("user").(*UserInfo)
	return user, ok
}

// GetUserFromJWT 从HTTP请求中解析JWT获取用户信息
func GetUserFromJWT(r *http.Request, jwtManager interface{}) (*UserInfo, error) {
	// 提取Authorization头
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("缺少认证令牌")
	}

	// 检查Bearer前缀
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("无效的令牌格式")
	}

	// 提取令牌
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// 这里需要根据实际的JWTManager接口进行调用
	// 暂时使用简单的JWT解析
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 这里应该从jwtManager获取secret，暂时硬编码
		return []byte("oj-access-secret-key-2024"), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("无效的令牌: %v", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("无效的令牌声明")
	}

	return &UserInfo{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		TokenID:  claims.TokenID,
	}, nil
}
