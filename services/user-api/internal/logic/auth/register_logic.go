package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dszqbsm/code-judger/common/types"
	"github.com/dszqbsm/code-judger/common/utils"
	"github.com/dszqbsm/code-judger/services/user-api/internal/svc"
	usertypes "github.com/dszqbsm/code-judger/services/user-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *usertypes.RegisterReq) (resp *usertypes.RegisterResp, err error) {
	// 1. 参数验证
	if err := l.validateRegisterRequest(req); err != nil {
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInvalidParams,
				Message: err.Error(),
			},
		}, nil
	}

	// 2. 检查注册是否开启
	if !l.svcCtx.Config.Business.Registration.EnableRegistration {
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeForbidden,
				Message: "系统暂时关闭注册功能",
			},
		}, nil
	}

	// 3. 检查用户名是否已存在
	usernameExists, err := l.svcCtx.UserModel.CheckUsernameExists(l.ctx, req.Username)
	if err != nil {
		logx.Errorf("检查用户名是否存在失败: %v", err)
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInternalError,
				Message: "系统错误",
			},
		}, nil
	}
	if usernameExists {
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeUserAlreadyExists,
				Message: "用户名已存在",
			},
		}, nil
	}

	// 4. 检查邮箱是否已存在
	emailExists, err := l.svcCtx.UserModel.CheckEmailExists(l.ctx, req.Email)
	if err != nil {
		logx.Errorf("检查邮箱是否存在失败: %v", err)
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInternalError,
				Message: "系统错误",
			},
		}, nil
	}
	if emailExists {
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeUserAlreadyExists,
				Message: "邮箱已被注册",
			},
		}, nil
	}

	// 5. 密码加密
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		logx.Errorf("密码加密失败: %v", err)
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInternalError,
				Message: "系统错误",
			},
		}, nil
	}

	// 6. 创建用户记录
	now := time.Now()
	user := &types.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  passwordHash,
		Role:          l.getValidRole(req.Role),
		Status:        types.StatusActive,
		EmailVerified: !l.svcCtx.Config.Business.Registration.RequireEmailVerification,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 7. 插入用户记录
	result, err := l.svcCtx.UserModel.Insert(l.ctx, user)
	if err != nil {
		logx.Errorf("创建用户失败: %v", err)
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInternalError,
				Message: "创建用户失败",
			},
		}, nil
	}

	// 8. 获取用户ID
	userID, err := result.LastInsertId()
	if err != nil {
		logx.Errorf("获取用户ID失败: %v", err)
		return &usertypes.RegisterResp{
			BaseResp: usertypes.BaseResp{
				Code:    utils.CodeInternalError,
				Message: "系统错误",
			},
		}, nil
	}
	user.ID = userID

	// 9. 初始化用户统计信息
	userStats := &types.UserStatistics{
		UserID:        userID,
		CurrentRating: 1200,
		MaxRating:     1200,
		RankLevel:     types.RankBronze,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, err = l.svcCtx.UserStatsModel.Insert(l.ctx, userStats)
	if err != nil {
		logx.Errorf("初始化用户统计信息失败: %v", err)
		// 这里不返回错误，因为用户已经创建成功
	}

	// 10. 构造响应
	userInfo := l.buildUserInfo(user)

	return &usertypes.RegisterResp{
		BaseResp: usertypes.BaseResp{
			Code:    utils.CodeSuccess,
			Message: "注册成功",
		},
		Data: userInfo,
	}, nil
}

// validateRegisterRequest 验证注册请求
func (l *RegisterLogic) validateRegisterRequest(req *usertypes.RegisterReq) error {
	// 用户名验证
	if len(req.Username) < 3 || len(req.Username) > 50 {
		return errors.New("用户名长度必须在3-50个字符之间")
	}
	if !isValidUsername(req.Username) {
		return errors.New("用户名只能包含字母、数字、下划线")
	}

	// 邮箱验证
	if !isValidEmail(req.Email) {
		return errors.New("邮箱格式不正确")
	}

	// 密码验证
	if err := l.validatePassword(req.Password); err != nil {
		return err
	}

	// 确认密码验证
	if req.Password != req.ConfirmPassword {
		return errors.New("两次输入的密码不一致")
	}

	return nil
}

// validatePassword 验证密码强度
func (l *RegisterLogic) validatePassword(password string) error {
	policy := l.svcCtx.Config.Business.PasswordPolicy

	if len(password) < policy.MinLength {
		return fmt.Errorf("密码长度至少%d个字符", policy.MinLength)
	}

	if policy.RequireUppercase && !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return errors.New("密码必须包含大写字母")
	}

	if policy.RequireLowercase && !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return errors.New("密码必须包含小写字母")
	}

	if policy.RequireNumbers && !strings.ContainsAny(password, "0123456789") {
		return errors.New("密码必须包含数字")
	}

	if policy.RequireSpecialChars && !strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		return errors.New("密码必须包含特殊字符")
	}

	return nil
}

// getValidRole 获取有效的角色
func (l *RegisterLogic) getValidRole(role string) string {
	// 只允许注册学生和教师角色
	if role == types.RoleStudent || role == types.RoleTeacher {
		return role
	}
	return l.svcCtx.Config.Business.Registration.DefaultRole
}

// buildUserInfo 构建用户信息响应
func (l *RegisterLogic) buildUserInfo(user *types.User) usertypes.UserInfo {
	return usertypes.UserInfo{
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
		LastLoginAt:   utils.FormatNullTime(user.LastLoginAt),
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
	}
}

// 辅助函数
func isValidUsername(username string) bool {
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
