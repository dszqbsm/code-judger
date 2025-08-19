package judge

import (
	"net/http"

	"github.com/online-judge/code-judger/services/judge-api/internal/logic/judge"
	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetJudgeResultHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetJudgeResultReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := judge.NewGetJudgeResultLogic(r.Context(), svcCtx)
		resp, err := l.GetJudgeResult(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
