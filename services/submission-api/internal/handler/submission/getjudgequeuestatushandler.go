package submission

import (
	"net/http"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/logic/submission"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetJudgeQueueStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := submission.NewGetJudgeQueueStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetJudgeQueueStatus()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}




