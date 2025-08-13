package problem

import (
	"net/http"

	"code-judger/services/problem-api/internal/logic/problem"
	"code-judger/services/problem-api/internal/svc"
	"code-judger/services/problem-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateProblemHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateProblemReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := problem.NewCreateProblemLogic(r.Context(), svcCtx)
		resp, err := l.CreateProblem(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}