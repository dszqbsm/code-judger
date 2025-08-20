package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"

	"github.com/online-judge/code-judger/services/submission-api/internal/config"
	"github.com/online-judge/code-judger/services/submission-api/internal/handler/submission"
	"github.com/online-judge/code-judger/services/submission-api/internal/svc"
)

var configFile = flag.String("f", "etc/submission-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf, rest.WithCustomCors(nil, func(w http.ResponseWriter) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
	}, "*"))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	registerHandlers(server, ctx)

	// 设置响应格式为JSON
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		}
	})

	fmt.Printf("Starting submission server at %s:%d...\n", c.RestConf.Host, c.RestConf.Port)
	server.Start()
}

func registerHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	// 提交相关路由
	server.AddRoutes([]rest.Route{
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/submissions",
			Handler: submission.CreateSubmissionHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/v1/submissions/:submission_id",
			Handler: submission.GetSubmissionHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/v1/submissions",
			Handler: submission.GetSubmissionListHandler(serverCtx),
		},
	})

	// WebSocket路由
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/ws/submissions/:submission_id/status",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			// 获取用户信息（从认证中间件）
			// 这里需要实现获取用户ID的逻辑
			userID := int64(1) // 简化处理

			serverCtx.WSManager.HandleWebSocket(w, r, userID)
		},
	})

	// 健康检查
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		},
	})
}
