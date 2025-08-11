package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/online-judge/code-judger/common/types"
	"github.com/online-judge/code-judger/common/utils"
	"github.com/online-judge/code-judger/services/user-api/internal/svc"
)

type AuthMiddleware struct {
	svcCtx *svc.ServiceContext
}

func NewAuthMiddleware(svcCtx *svc.ServiceContext) *AuthMiddleware {
	return &AuthMiddleware{
		svcCtx: svcCtx,
	}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取Authorization头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.Error(w, utils.CodeUnauthorized, "缺少认证信息")
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.Error(w, utils.CodeUnauthorized, "无效的认证格式")
			return
		}

		// 提取令牌
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			utils.Error(w, utils.CodeUnauthorized, "令牌为空")
			return
		}

		// 解析和验证令牌
		claims, err := m.svcCtx.JWTManager.ParseAccessToken(token)
		if err != nil {
			utils.Error(w, utils.CodeInvalidToken, "无效的令牌")
			return
		}

		// 检查令牌是否被撤销
		isRevoked, err := m.svcCtx.UserTokenModel.IsTokenRevoked(r.Context(), claims.TokenID)
		if err != nil {
			utils.Error(w, utils.CodeInternalError, "验证令牌状态失败")
			return
		}
		if isRevoked {
			utils.Error(w, utils.CodeTokenExpired, "令牌已失效")
			return
		}

		// 获取用户信息
		user, err := m.svcCtx.UserModel.FindOne(r.Context(), claims.UserID)
		if err != nil {
			utils.Error(w, utils.CodeUserNotFound, "用户不存在")
			return
		}

		// 检查用户状态
		if user.Status != "active" {
			if user.Status == "banned" {
				utils.Error(w, utils.CodeUserBanned, "账户已被封禁")
			} else {
				utils.Error(w, utils.CodeForbidden, "账户未激活")
			}
			return
		}

		// 将用户信息添加到上下文
		ctx := context.WithValue(r.Context(), "user", user)
		ctx = context.WithValue(ctx, "claims", claims)

		next(w, r.WithContext(ctx))
	}
}

// GetCurrentUser 从上下文获取当前用户
func GetCurrentUser(ctx context.Context) (*types.User, bool) {
	user, ok := ctx.Value("user").(*types.User)
	return user, ok
}

// GetCurrentClaims 从上下文获取JWT声明
func GetCurrentClaims(ctx context.Context) (*utils.JWTClaims, bool) {
	claims, ok := ctx.Value("claims").(*utils.JWTClaims)
	return claims, ok
}
