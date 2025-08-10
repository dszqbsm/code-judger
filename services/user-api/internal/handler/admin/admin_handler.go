package admin

import (
	"net/http"

	"github.com/online-judge/code-judger/common/utils"
	"github.com/online-judge/code-judger/services/user-api/internal/svc"
	"github.com/online-judge/code-judger/services/user-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetUserListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UserListReq
		if err := httpx.Parse(r, &req); err != nil {
			utils.Error(w, utils.CodeInvalidParams, err.Error())
			return
		}

		// TODO: 实现获取用户列表逻辑
		utils.Success(w, map[string]string{"message": "获取用户列表功能待实现"})
	}
}

func UpdateUserRoleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateUserRoleReq
		if err := httpx.Parse(r, &req); err != nil {
			utils.Error(w, utils.CodeInvalidParams, err.Error())
			return
		}

		// TODO: 实现更新用户角色逻辑
		utils.Success(w, map[string]string{"message": "更新用户角色功能待实现"})
	}
}