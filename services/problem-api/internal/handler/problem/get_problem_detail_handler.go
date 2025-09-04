package problem

import (
	"net/http"

	"github.com/dszqbsm/code-judger/services/problem-api/internal/logic/problem"
	"github.com/dszqbsm/code-judger/services/problem-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/problem-api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetProblemDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetProblemDetailReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := problem.NewGetProblemDetailLogic(r.Context(), svcCtx)
		resp, err := l.GetProblemDetail(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
