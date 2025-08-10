package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/online-judge/code-judger/common/types"
	"github.com/online-judge/code-judger/common/utils"
	"github.com/online-judge/code-judger/services/user-api/internal/svc"
	usertypes "github.com/online-judge/code-judger/services/user-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *usertypes.LoginReq) (resp *usertypes.LoginResp, err error) {
	// 1. 查找用户
	user, err := l.svcCtx.UserModel.FindByUsernameOrEmail(l.ctx, req.Username)
	if err != nil {
		logx.Errorf("查找用户失败: %v", err)
		return &usertypes.LoginResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInvalidCredentials,
				Message: "用户名或密码错误",
			},
		}, nil
	}

	// 2. 验证密码
	if !utils.VerifyPassword(user.PasswordHash, req.Password) {
		return &usertypes.LoginResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInvalidCredentials,
				Message: "用户名或密码错误",
			},
		}, nil
	}

	// 3. 检查用户状态
	if user.Status == types.StatusBanned {
		return &usertypes.LoginResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeUserBanned,
				Message: "账户已被封禁",
			},
		}, nil
	}

	if user.Status == types.StatusInactive {
		return &usertypes.LoginResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeForbidden,
				Message: "账户未激活",
			},
		}, nil
	}

	// 4. 生成JWT令牌
	accessToken, refreshToken, tokenID, err := l.svcCtx.JWTManager.GenerateTokens(
		user.ID, user.Username, user.Role,
	)
	if err != nil {
		logx.Errorf("生成令牌失败: %v", err)
		return &usertypes.LoginResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInternalError,
				Message: "生成令牌失败",
			},
		}, nil
	}

	// 5. 保存令牌信息
	now := time.Now()
	userToken := &types.UserToken{
		UserID:             user.ID,
		TokenID:            tokenID,
		RefreshToken:       refreshToken,
		AccessTokenExpire:  now.Add(time.Duration(l.svcCtx.Config.Auth.AccessExpire) * time.Second),
		RefreshTokenExpire: now.Add(time.Duration(l.svcCtx.Config.Auth.RefreshExpire) * time.Second),
		ClientInfo:         "", // TODO: 从请求头提取客户端信息
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	_, err = l.svcCtx.UserTokenModel.Insert(l.ctx, userToken)
	if err != nil {
		logx.Errorf("保存令牌信息失败: %v", err)
		return &usertypes.LoginResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInternalError,
				Message: "系统错误",
			},
		}, nil
	}

	// 6. 更新用户登录信息
	err = l.svcCtx.UserModel.UpdateLastLogin(l.ctx, user.ID, "127.0.0.1") // TODO: 获取真实IP
	if err != nil {
		logx.Errorf("更新登录信息失败: %v", err)
		// 不影响登录流程
	}

	// 7. 记录登录日志
	loginLog := &types.UserLoginLog{
		UserID:      user.ID,
		LoginType:   types.LoginTypePassword,
		IPAddress:   "127.0.0.1", // TODO: 获取真实IP
		UserAgent:   "",          // TODO: 从请求头获取
		LoginStatus: types.LoginStatusSuccess,
		CreatedAt:   now,
	}
	_, err = l.svcCtx.UserLoginLogModel.Insert(l.ctx, loginLog)
	if err != nil {
		logx.Errorf("记录登录日志失败: %v", err)
		// 不影响登录流程
	}

	// 8. 构造响应
	userInfo := buildUserInfo(user)

	return &usertypes.LoginResp{
		BaseResp: usertypes.BaseResp{
			Code:    utils.CodeSuccess,
			Message: "登录成功",
		},
		Data: struct {
			usertypes.TokenInfo
			UserInfo usertypes.UserInfo `json:"user_info"`
		}{
			TokenInfo: usertypes.TokenInfo{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				TokenType:    "Bearer",
				ExpiresIn:    l.svcCtx.Config.Auth.AccessExpire,
			},
			UserInfo: userInfo,
		},
	}, nil
}

func buildUserInfo(user *types.User) usertypes.UserInfo {
	return usertypes.UserInfo{
		UserId:       user.ID,
		Username:     user.Username,
		Email:        user.Email,
		RealName:     user.RealName,
		AvatarUrl:    user.AvatarUrl,
		Bio:          user.Bio,
		Role:         user.Role,
		Status:       user.Status,
		EmailVerified: user.EmailVerified,
		LoginCount:   user.LoginCount,
		LastLoginAt:  formatTimePtr(user.LastLoginAt),
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
	}
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}