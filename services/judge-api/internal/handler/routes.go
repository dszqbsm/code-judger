package handler

import (
	"net/http"

	judge "github.com/dszqbsm/code-judger/services/judge-api/internal/handler/judge"
	system "github.com/dszqbsm/code-judger/services/judge-api/internal/handler/system"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/judge/submit",
				Handler: judge.SubmitJudgeHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/judge/result/:submission_id",
				Handler: judge.GetJudgeResultHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/judge/status/:submission_id",
				Handler: judge.GetJudgeStatusHandler(serverCtx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/api/v1/judge/cancel/:submission_id",
				Handler: judge.CancelJudgeHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/judge/rejudge/:submission_id",
				Handler: judge.RejudgeHandler(serverCtx),
			},
		},
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/judge/nodes",
				Handler: system.GetJudgeNodesHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/judge/queue",
				Handler: system.GetJudgeQueueHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/judge/health",
				Handler: system.HealthCheckHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/judge/languages",
				Handler: system.GetLanguagesHandler(serverCtx),
			},
		},
	)
}
