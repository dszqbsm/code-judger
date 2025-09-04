package users

import (
	"net/http"
	"time"

	"github.com/dszqbsm/code-judger/common/utils"
	"github.com/dszqbsm/code-judger/services/user-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/user-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/user-api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetProfileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从上下文获取当前用户
		user, ok := middleware.GetCurrentUser(r.Context())
		if !ok {
			utils.Error(w, utils.CodeUnauthorized, "未找到用户信息")
			return
		}

		userInfo := types.UserInfo{
			UserId:        user.ID,
			Username:      user.Username,
			Email:         user.Email,
			RealName:      user.RealName,
			AvatarUrl:     user.AvatarUrl,
			Bio:           user.Bio,
			Role:          user.Role,
			Status:        user.Status,
			EmailVerified: user.EmailVerified,
			LoginCount:    user.LoginCount,
			LastLoginAt:   utils.FormatNullTimeCustom(user.LastLoginAt, "2006-01-02T15:04:05Z07:00"),
			CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		resp := types.UserProfileResp{
			BaseResp: types.BaseResp{
				Code:    utils.CodeSuccess,
				Message: "获取成功",
			},
			Data: userInfo,
		}

		utils.Success(w, resp)
	}
}

func UpdateProfileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateProfileReq
		if err := httpx.Parse(r, &req); err != nil {
			utils.Error(w, utils.CodeInvalidParams, err.Error())
			return
		}

		// TODO: 实现更新个人信息逻辑
		utils.Success(w, map[string]string{"message": "更新个人信息功能待实现"})
	}
}

func ChangePasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChangePasswordReq
		if err := httpx.Parse(r, &req); err != nil {
			utils.Error(w, utils.CodeInvalidParams, err.Error())
			return
		}

		// TODO: 实现修改密码逻辑
		utils.Success(w, map[string]string{"message": "修改密码功能待实现"})
	}
}

func GetUserStatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: 实现获取用户统计逻辑
		utils.Success(w, map[string]string{"message": "获取用户统计功能待实现"})
	}
}

func GetUserPermissionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: 实现获取用户权限逻辑
		utils.Success(w, map[string]string{"message": "获取用户权限功能待实现"})
	}
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}
