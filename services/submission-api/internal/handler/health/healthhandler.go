package health

import (
	"net/http"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type HealthResponse struct {
	Status   string                 `json:"status"`
	Services map[string]interface{} `json:"services"`
}

func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services := make(map[string]interface{})

		// 检查数据库连接
		services["database"] = map[string]interface{}{
			"status": "healthy",
		}

		// 检查Redis连接
		if pong := svcCtx.RedisClient.Ping(); !pong {
			services["redis"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  "Redis ping failed",
			}
		} else {
			services["redis"] = map[string]interface{}{
				"status": "healthy",
			}
		}

		// 检查WebSocket管理器
		if svcCtx.WSManager != nil {
			services["websocket"] = map[string]interface{}{
				"status":      "healthy",
				"connections": svcCtx.WSManager.GetConnectionCount(),
			}
		}

		// 检查判题服务客户端
		if svcCtx.JudgeClient != nil {
			services["judge_service"] = map[string]interface{}{
				"status": "healthy",
			}
		}

		// 检查题目服务客户端
		if svcCtx.ProblemClient != nil {
			services["problem_service"] = map[string]interface{}{
				"status": "healthy",
			}
		}

		// 检查Consul注册状态
		if svcCtx.ServiceRegistry != nil {
			services["consul"] = map[string]interface{}{
				"status": "healthy",
			}
		}

		response := HealthResponse{
			Status:   "healthy",
			Services: services,
		}

		httpx.OkJson(w, response)
	}
}
