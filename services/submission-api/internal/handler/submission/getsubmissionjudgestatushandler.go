package submission

import (
	"net/http"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/logic/submission"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetSubmissionJudgeStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetSubmissionJudgeStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := submission.NewGetSubmissionJudgeStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetSubmissionJudgeStatus(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}




