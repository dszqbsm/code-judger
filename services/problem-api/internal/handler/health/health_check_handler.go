package health

import (
	"net/http"

	"github.com/online-judge/code-judger/services/problem-api/internal/logic/health"
	"github.com/online-judge/code-judger/services/problem-api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func HealthCheckHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := health.NewHealthCheckLogic(r.Context(), svcCtx)
		resp, err := l.HealthCheck()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
