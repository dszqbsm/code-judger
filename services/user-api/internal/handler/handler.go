package handler

import (
	"net/http"

	"github.com/online-judge/code-judger/services/user-api/internal/handler/admin"
	"github.com/online-judge/code-judger/services/user-api/internal/handler/auth"
	"github.com/online-judge/code-judger/services/user-api/internal/handler/users"
	"github.com/online-judge/code-judger/services/user-api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	// 认证相关路由
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/register",
				Handler: auth.RegisterHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/login",
				Handler: auth.LoginHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/refresh",
				Handler: auth.RefreshTokenHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/logout",
				Handler: auth.LogoutHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/verify-permission",
				Handler: auth.VerifyPermissionHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api/v1/auth"),
	)

	// 用户相关路由（需要认证）
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/profile",
				Handler: users.GetProfileHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/profile",
				Handler: users.UpdateProfileHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/password",
				Handler: users.ChangePasswordHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/:user_id/stats",
				Handler: users.GetUserStatsHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/:user_id/permissions",
				Handler: users.GetUserPermissionsHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api/v1/users"),
	)

	// 管理员相关路由（需要认证和管理员权限）
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: admin.GetUserListHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/:user_id/role",
				Handler: admin.UpdateUserRoleHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api/v1/admin/users"),
	)
}
