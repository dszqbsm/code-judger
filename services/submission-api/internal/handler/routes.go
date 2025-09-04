package handler

import (
	"net/http"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/handler/submission"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			// 提交相关路由
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/submissions",
				Handler: submission.CreateSubmissionHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/submissions/:id",
				Handler: submission.GetSubmissionHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/submissions",
				Handler: submission.GetSubmissionListHandler(serverCtx),
			},
			// 判题相关代理接口
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/submissions/:submission_id/result",
				Handler: submission.GetSubmissionJudgeResultHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/submissions/:submission_id/status",
				Handler: submission.GetSubmissionJudgeStatusHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/submissions/:submission_id/rejudge",
				Handler: submission.RejudgeSubmissionProxyHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret), // JWT认证中间件
	)

	// 管理员专用路由
	server.AddRoutes(
		[]rest.Route{
			// 管理员判题队列监控
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/admin/submissions/judge/queue",
				Handler: submission.GetJudgeQueueStatusHandler(serverCtx),
			},
			// 管理员重新判题（保留原有的）
			{
				Method:  http.MethodPut,
				Path:    "/api/v1/admin/submissions/:id/rejudge",
				Handler: submission.RejudgeSubmissionHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		// 可以添加管理员权限中间件
		// rest.WithMiddleware(middleware.AdminAuthMiddleware),
	)
}
