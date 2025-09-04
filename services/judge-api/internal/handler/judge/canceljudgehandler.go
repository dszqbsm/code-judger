package judge

import (
	"net/http"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/logic/judge"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CancelJudgeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CancelJudgeReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := judge.NewCancelJudgeLogic(r.Context(), svcCtx)
		resp, err := l.CancelJudge(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
