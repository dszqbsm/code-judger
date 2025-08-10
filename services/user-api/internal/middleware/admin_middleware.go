package middleware

import (
	"net/http"

	"github.com/online-judge/code-judger/common/types"
	"github.com/online-judge/code-judger/common/utils"
)

type AdminOnlyMiddleware struct{}

func NewAdminOnlyMiddleware() *AdminOnlyMiddleware {
	return &AdminOnlyMiddleware{}
}

func (m *AdminOnlyMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取当前用户信息
		user, ok := GetCurrentUser(r.Context())
		if !ok {
			utils.Error(w, utils.CodeUnauthorized, "未找到用户信息")
			return
		}

		// 检查是否为管理员
		if user.Role != types.RoleAdmin {
			utils.Error(w, utils.CodePermissionDenied, "需要管理员权限")
			return
		}

		next(w, r)
	}
}