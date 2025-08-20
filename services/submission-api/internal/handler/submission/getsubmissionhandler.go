package submission

import (
	"net/http"

	"code-judger/services/submission-api/internal/logic/submission"
	"code-judger/services/submission-api/internal/svc"
	"code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetSubmissionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetSubmissionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := submission.NewGetSubmissionLogic(r.Context(), svcCtx)
		resp, err := l.GetSubmission(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
