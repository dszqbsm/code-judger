package types

// RegisterReq 用户注册请求
type RegisterReq struct {
	Username        string `json:"username" validate:"required,min=3,max=50"`
	Email          string `json:"email" validate:"required,email"`
	Password       string `json:"password" validate:"required,min=8,max=50"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
	Role           string `json:"role" validate:"oneof=student teacher" default:"student"`
}

// LoginReq 用户登录请求
type LoginReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RefreshTokenReq 令牌刷新请求
type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordReq 修改密码请求
type ChangePasswordReq struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

// UpdateProfileReq 更新用户信息请求
type UpdateProfileReq struct {
	RealName  string `json:"real_name,optional"`
	AvatarUrl string `json:"avatar_url,optional"`
	Bio       string `json:"bio,optional"`
}

// VerifyPermissionReq 权限验证请求
type VerifyPermissionReq struct {
	Resource string `json:"resource" validate:"required"`
	Action   string `json:"action" validate:"required"`
}

// UpdateUserRoleReq 更新用户角色请求
type UpdateUserRoleReq struct {
	Role string `json:"role" validate:"required,oneof=student teacher admin"`
}

// UserListReq 用户列表查询请求
type UserListReq struct {
	Page     int64  `form:"page,optional,default=1"`
	PageSize int64  `form:"page_size,optional,default=20"`
	Role     string `form:"role,optional"`
	Status   string `form:"status,optional"`
	Keyword  string `form:"keyword,optional"`
}

// UserInfo 用户信息响应
type UserInfo struct {
	UserId       int64  `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	RealName     string `json:"real_name"`
	AvatarUrl    string `json:"avatar_url"`
	Bio          string `json:"bio"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	EmailVerified bool  `json:"email_verified"`
	LoginCount   int64  `json:"login_count"`
	LastLoginAt  string `json:"last_login_at"`
	CreatedAt    string `json:"created_at"`
}

// TokenInfo 认证令牌信息
type TokenInfo struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// UserStats 用户统计信息
type UserStats struct {
	TotalSubmissions    int64  `json:"total_submissions"`
	AcceptedSubmissions int64  `json:"accepted_submissions"`
	SolvedProblems      int64  `json:"solved_problems"`
	EasySolved          int64  `json:"easy_solved"`
	MediumSolved        int64  `json:"medium_solved"`
	HardSolved          int64  `json:"hard_solved"`
	CurrentRating       int64  `json:"current_rating"`
	MaxRating           int64  `json:"max_rating"`
	RankLevel           string `json:"rank_level"`
	ContestParticipated int64  `json:"contest_participated"`
	ContestWon          int64  `json:"contest_won"`
}

// UserPermissions 用户权限信息
type UserPermissions struct {
	Permissions []string `json:"permissions"`
}

// UserListResp 用户列表响应
type UserListResp struct {
	Users []UserInfo `json:"users"`
	Total int64      `json:"total"`
}

// 统一响应格式
type BaseResp struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type RegisterResp struct {
	BaseResp
	Data UserInfo `json:"data"`
}

type LoginResp struct {
	BaseResp
	Data struct {
		TokenInfo
		UserInfo UserInfo `json:"user_info"`
	} `json:"data"`
}

type RefreshTokenResp struct {
	BaseResp
	Data TokenInfo `json:"data"`
}

type UserProfileResp struct {
	BaseResp
	Data UserInfo `json:"data"`
}

type UserStatsResp struct {
	BaseResp
	Data UserStats `json:"data"`
}

type UserPermissionsResp struct {
	BaseResp
	Data UserPermissions `json:"data"`
}

type UserListResponse struct {
	BaseResp
	Data UserListResp `json:"data"`
}