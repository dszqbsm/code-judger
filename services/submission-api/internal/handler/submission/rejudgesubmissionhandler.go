package submission

import (
	"net/http"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/logic/submission"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func RejudgeSubmissionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RejudgeSubmissionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := submission.NewRejudgeSubmissionLogic(r.Context(), svcCtx, r)
		resp, err := l.RejudgeSubmission(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
