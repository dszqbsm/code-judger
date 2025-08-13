package handler

import (
	"net/http"

	"code-judger/services/problem-api/internal/handler/health"
	"code-judger/services/problem-api/internal/handler/problem"
	"code-judger/services/problem-api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			// 健康检查和监控接口（不需要认证）
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/health",
				Handler: health.HealthCheckHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/metrics",
				Handler: health.MetricsHandler(serverCtx),
			},
		},
	)

	server.AddRoutes(
		[]rest.Route{
			// 题目管理接口（需要JWT认证）
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/problems",
				Handler: problem.CreateProblemHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/problems",
				Handler: problem.GetProblemListHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/problems/:id",
				Handler: problem.GetProblemDetailHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/api/v1/problems/:id",
				Handler: problem.UpdateProblemHandler(serverCtx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/api/v1/problems/:id",
				Handler: problem.DeleteProblemHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret), // JWT认证中间件
	)
}