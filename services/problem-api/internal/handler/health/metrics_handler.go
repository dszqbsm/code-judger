package health

import (
	"net/http"

	"code-judger/services/problem-api/internal/logic/health"
	"code-judger/services/problem-api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func MetricsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := health.NewMetricsLogic(r.Context(), svcCtx)
		resp, err := l.Metrics()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}