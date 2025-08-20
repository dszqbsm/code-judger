package middleware

import (
	"net/http"
)

type AdminOnlyMiddleware struct{}

func NewAdminOnlyMiddleware() AdminOnlyMiddleware {
	return AdminOnlyMiddleware{}
}

func (m AdminOnlyMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取用户信息
		user, ok := GetUserFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":401,"message":"用户信息不存在"}`))
			return
		}

		// 检查是否为管理员
		if user.Role != "admin" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"code":403,"message":"权限不足，需要管理员权限"}`))
			return
		}

		next(w, r)
	}
}
