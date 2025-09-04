package websocket

import (
	"net/http"
	"strconv"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

func WebSocketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从JWT中获取用户信息
		user, err := middleware.GetUserFromJWT(r, svcCtx.JWTManager)
		if err != nil {
			logx.Errorf("WebSocket连接认证失败: %v", err)
			http.Error(w, "认证失败", http.StatusUnauthorized)
			return
		}

		// 处理WebSocket连接
		svcCtx.WSManager.HandleWebSocket(w, r, user.UserID)
	}
}

func SubmissionWebSocketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从JWT中获取用户信息
		user, err := middleware.GetUserFromJWT(r, svcCtx.JWTManager)
		if err != nil {
			logx.Errorf("WebSocket连接认证失败: %v", err)
			http.Error(w, "认证失败", http.StatusUnauthorized)
			return
		}

		// 从路径参数获取提交ID
		submissionIDStr := r.URL.Query().Get("submission_id")
		if submissionIDStr == "" {
			http.Error(w, "缺少submission_id参数", http.StatusBadRequest)
			return
		}

		submissionID, err := strconv.ParseInt(submissionIDStr, 10, 64)
		if err != nil {
			http.Error(w, "无效的submission_id", http.StatusBadRequest)
			return
		}

		// 验证用户是否有权限订阅此提交的更新
		submission, err := svcCtx.SubmissionDao.GetSubmissionByID(r.Context(), submissionID)
		if err != nil {
			logx.Errorf("查询提交记录失败: %v", err)
			http.Error(w, "提交记录不存在", http.StatusNotFound)
			return
		}

		if submission.UserID != user.UserID && user.Role != "admin" && user.Role != "teacher" {
			http.Error(w, "无权限订阅此提交的更新", http.StatusForbidden)
			return
		}

		// 处理WebSocket连接
		svcCtx.WSManager.HandleWebSocket(w, r, user.UserID)
	}
}
