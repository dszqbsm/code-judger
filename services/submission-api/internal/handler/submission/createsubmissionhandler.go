package submission

import (
	"net/http"

	"github.com/online-judge/code-judger/services/submission-api/internal/logic/submission"
	"github.com/online-judge/code-judger/services/submission-api/internal/svc"
	"github.com/online-judge/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateSubmissionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateSubmissionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := submission.NewCreateSubmissionLogic(r.Context(), svcCtx)
		resp, err := l.CreateSubmission(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
