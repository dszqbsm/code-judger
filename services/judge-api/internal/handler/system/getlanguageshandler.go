package system

import (
	"net/http"

	"github.com/online-judge/code-judger/services/judge-api/internal/logic/system"
	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetLanguagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetLanguagesReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := system.NewGetLanguagesLogic(r.Context(), svcCtx)
		resp, err := l.GetLanguages(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
