package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/dszqbsm/code-judger/common/types"
	"github.com/dszqbsm/code-judger/common/utils"
)

// UserInfo 用户信息结构
type UserInfo struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TokenID  string `json:"token_id"`
}

// GetUserFromJWT 从HTTP请求的JWT令牌中获取用户信息
func GetUserFromJWT(r *http.Request, jwtManager *utils.JWTManager) (*UserInfo, error) {
	// 获取Authorization头
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("缺少认证信息")
	}

	// 检查Bearer前缀
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("无效的认证格式")
	}

	// 提取令牌
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, fmt.Errorf("令牌为空")
	}

	// 解析和验证令牌
	claims, err := jwtManager.ParseAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("无效的令牌: %v", err)
	}

	return &UserInfo{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		TokenID:  claims.TokenID,
	}, nil
}

// GetUserFromContext 从go-zero上下文中获取用户信息（如果使用了go-zero的JWT中间件）
func GetUserFromContext(ctx context.Context) (*UserInfo, error) {
	// go-zero JWT中间件会将用户ID存储在上下文中
	userIDValue := ctx.Value("userId")
	if userIDValue == nil {
		return nil, fmt.Errorf("用户信息不存在")
	}

	userID, ok := userIDValue.(int64)
	if !ok {
		return nil, fmt.Errorf("用户ID格式错误")
	}

	// 尝试获取其他用户信息（如果存在）
	username := ""
	role := ""
	tokenID := ""

	if usernameValue := ctx.Value("username"); usernameValue != nil {
		if u, ok := usernameValue.(string); ok {
			username = u
		}
	}

	if roleValue := ctx.Value("role"); roleValue != nil {
		if r, ok := roleValue.(string); ok {
			role = r
		}
	}

	if tokenIDValue := ctx.Value("tokenId"); tokenIDValue != nil {
		if t, ok := tokenIDValue.(string); ok {
			tokenID = t
		}
	}

	return &UserInfo{
		UserID:   userID,
		Username: username,
		Role:     role,
		TokenID:  tokenID,
	}, nil
}

// HasPermission 检查用户是否具有指定权限
func HasPermission(userRole string, requiredPermission string) bool {
	permissions, exists := types.RolePermissions[userRole]
	if !exists {
		return false
	}

	// 检查是否有全权限
	for _, perm := range permissions {
		if perm == "*" || strings.HasSuffix(perm, ":*") {
			// 检查是否匹配通配符权限
			if perm == "*" {
				return true
			}
			if strings.HasPrefix(requiredPermission, strings.TrimSuffix(perm, "*")) {
				return true
			}
		}
		if perm == requiredPermission {
			return true
		}
	}

	return false
}

// ValidateCreateProblemPermission 验证创建题目权限
func ValidateCreateProblemPermission(userRole string) error {
	// 只有教师和管理员可以创建题目
	if userRole != types.RoleTeacher && userRole != types.RoleAdmin {
		return fmt.Errorf("权限不足：只有教师和管理员可以创建题目")
	}

	// 使用通用权限检查
	if !HasPermission(userRole, "problem:create") {
		return fmt.Errorf("权限不足：缺少题目创建权限")
	}

	return nil
}

// ValidateUpdateProblemPermission 验证更新题目权限
func ValidateUpdateProblemPermission(userRole string, userID int64, problemCreatedBy int64) error {
	// 管理员可以修改任何题目
	if userRole == types.RoleAdmin && HasPermission(userRole, "problem:*") {
		return nil
	}

	// 教师只能修改自己创建的题目
	if userRole == types.RoleTeacher {
		if HasPermission(userRole, "problem:update:own") && userID == problemCreatedBy {
			return nil
		}
		return fmt.Errorf("权限不足：只能修改自己创建的题目")
	}

	return fmt.Errorf("权限不足：无题目修改权限")
}

// ValidateDeleteProblemPermission 验证删除题目权限
func ValidateDeleteProblemPermission(userRole string, userID int64, problemCreatedBy int64) error {
	// 管理员可以删除任何题目
	if userRole == types.RoleAdmin && HasPermission(userRole, "problem:*") {
		return nil
	}

	// 教师只能删除自己创建的题目
	if userRole == types.RoleTeacher {
		if HasPermission(userRole, "problem:delete:own") && userID == problemCreatedBy {
			return nil
		}
		return fmt.Errorf("权限不足：只能删除自己创建的题目")
	}

	return fmt.Errorf("权限不足：无题目删除权限")
}





