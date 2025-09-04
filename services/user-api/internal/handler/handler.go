package handler

import (
	"net/http"

	"github.com/dszqbsm/code-judger/services/user-api/internal/handler/admin"
	"github.com/dszqbsm/code-judger/services/user-api/internal/handler/auth"
	"github.com/dszqbsm/code-judger/services/user-api/internal/handler/users"
	"github.com/dszqbsm/code-judger/services/user-api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	// 认证相关路由（公开接口）
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
		},
		rest.WithPrefix("/api/v1/auth"),
	)

	// 认证相关路由（需要令牌）
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/refresh",
				Handler: serverCtx.Auth(auth.RefreshTokenHandler(serverCtx)),
			},
			{
				Method:  http.MethodPost,
				Path:    "/logout",
				Handler: serverCtx.Auth(auth.LogoutHandler(serverCtx)),
			},
			{
				Method:  http.MethodPost,
				Path:    "/verify-permission",
				Handler: serverCtx.Auth(auth.VerifyPermissionHandler(serverCtx)),
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
				Handler: serverCtx.Auth(users.GetProfileHandler(serverCtx)),
			},
			{
				Method:  http.MethodPut,
				Path:    "/profile",
				Handler: serverCtx.Auth(users.UpdateProfileHandler(serverCtx)),
			},
			{
				Method:  http.MethodPut,
				Path:    "/password",
				Handler: serverCtx.Auth(users.ChangePasswordHandler(serverCtx)),
			},
			{
				Method:  http.MethodGet,
				Path:    "/:user_id/stats",
				Handler: serverCtx.Auth(users.GetUserStatsHandler(serverCtx)),
			},
			{
				Method:  http.MethodGet,
				Path:    "/:user_id/permissions",
				Handler: serverCtx.Auth(users.GetUserPermissionsHandler(serverCtx)),
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
				Handler: serverCtx.Auth(serverCtx.AdminOnly(admin.GetUserListHandler(serverCtx))),
			},
			{
				Method:  http.MethodPut,
				Path:    "/:user_id/role",
				Handler: serverCtx.Auth(serverCtx.AdminOnly(admin.UpdateUserRoleHandler(serverCtx))),
			},
		},
		rest.WithPrefix("/api/v1/admin/users"),
	)
}
