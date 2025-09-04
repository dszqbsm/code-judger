package auth

import (
	"net/http"

	"github.com/dszqbsm/code-judger/common/utils"
	"github.com/dszqbsm/code-judger/services/user-api/internal/logic/auth"
	"github.com/dszqbsm/code-judger/services/user-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/user-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := httpx.Parse(r, &req); err != nil {
			utils.Error(w, utils.CodeInvalidParams, err.Error())
			return
		}

		l := auth.NewLoginLogicWithRequest(r.Context(), svcCtx, r)
		resp, err := l.Login(&req)
		if err != nil {
			utils.Error(w, utils.CodeInternalError, err.Error())
		} else {
			httpx.OkJson(w, resp)
		}
	}
}

func RefreshTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshTokenReq
		if err := httpx.Parse(r, &req); err != nil {
			utils.Error(w, utils.CodeInvalidParams, err.Error())
			return
		}

		// TODO: 实现刷新令牌逻辑
		utils.Success(w, map[string]string{"message": "刷新令牌功能待实现"})
	}
}

func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: 实现登出逻辑
		utils.Success(w, map[string]string{"message": "登出功能待实现"})
	}
}

func VerifyPermissionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.VerifyPermissionReq
		if err := httpx.Parse(r, &req); err != nil {
			utils.Error(w, utils.CodeInvalidParams, err.Error())
			return
		}

		// TODO: 实现权限验证逻辑
		utils.Success(w, map[string]string{"message": "权限验证功能待实现"})
	}
}