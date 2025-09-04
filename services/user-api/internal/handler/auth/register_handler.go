package auth

import (
	"net/http"

	"github.com/dszqbsm/code-judger/common/utils"
	"github.com/dszqbsm/code-judger/services/user-api/internal/logic/auth"
	"github.com/dszqbsm/code-judger/services/user-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/user-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterReq
		if err := httpx.Parse(r, &req); err != nil {
			utils.Error(w, utils.CodeInvalidParams, err.Error())
			return
		}

		l := auth.NewRegisterLogic(r.Context(), svcCtx)
		resp, err := l.Register(&req)
		if err != nil {
			utils.Error(w, utils.CodeInternalError, err.Error())
		} else {
			httpx.OkJson(w, resp)
		}
	}
}