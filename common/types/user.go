package types

import (
	"database/sql"
	"time"
)

// User 用户基础信息
type User struct {
	ID            int64     `json:"id" db:"id"`
	Username      string    `json:"username" db:"username"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"` // 不对外暴露密码哈希
	RealName      sql.NullString `json:"real_name" db:"real_name"`
	AvatarUrl     sql.NullString `json:"avatar_url" db:"avatar_url"`
	Bio           sql.NullString `json:"bio" db:"bio"`
	Role          string    `json:"role" db:"role"`
	Status        string    `json:"status" db:"status"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	LastLoginAt   *time.Time `json:"last_login_at" db:"last_login_at"`
	LastLoginIP   string    `json:"last_login_ip" db:"last_login_ip"`
	LoginCount    int64     `json:"login_count" db:"login_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// UserToken 用户令牌信息
type UserToken struct {
	ID                 int64     `json:"id" db:"id"`
	UserID             int64     `json:"user_id" db:"user_id"`
	TokenID            string    `json:"token_id" db:"token_id"`
	RefreshToken       string    `json:"refresh_token" db:"refresh_token"`
	AccessTokenExpire  time.Time `json:"access_token_expire" db:"access_token_expire"`
	RefreshTokenExpire time.Time `json:"refresh_token_expire" db:"refresh_token_expire"`
	ClientInfo         string    `json:"client_info" db:"client_info"`
	IsRevoked          bool      `json:"is_revoked" db:"is_revoked"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// UserLoginLog 用户登录日志
type UserLoginLog struct {
	ID            int64     `json:"id" db:"id"`
	UserID        int64     `json:"user_id" db:"user_id"`
	LoginType     string    `json:"login_type" db:"login_type"`
	IPAddress     string    `json:"ip_address" db:"ip_address"`
	UserAgent     string    `json:"user_agent" db:"user_agent"`
	LoginStatus   string    `json:"login_status" db:"login_status"`
	FailureReason string    `json:"failure_reason" db:"failure_reason"`
	LocationInfo  string    `json:"location_info" db:"location_info"`
	DeviceInfo    string    `json:"device_info" db:"device_info"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// UserStatistics 用户统计信息
type UserStatistics struct {
	ID                  int64     `json:"id" db:"id"`
	UserID              int64     `json:"user_id" db:"user_id"`
	TotalSubmissions    int64     `json:"total_submissions" db:"total_submissions"`
	AcceptedSubmissions int64     `json:"accepted_submissions" db:"accepted_submissions"`
	SolvedProblems      int64     `json:"solved_problems" db:"solved_problems"`
	EasySolved          int64     `json:"easy_solved" db:"easy_solved"`
	MediumSolved        int64     `json:"medium_solved" db:"medium_solved"`
	HardSolved          int64     `json:"hard_solved" db:"hard_solved"`
	CurrentRating       int64     `json:"current_rating" db:"current_rating"`
	MaxRating           int64     `json:"max_rating" db:"max_rating"`
	RankLevel           string    `json:"rank_level" db:"rank_level"`
	TotalCodeTime       int64     `json:"total_code_time" db:"total_code_time"`
	AverageSolveTime    int64     `json:"average_solve_time" db:"average_solve_time"`
	ContestParticipated int64     `json:"contest_participated" db:"contest_participated"`
	ContestWon          int64     `json:"contest_won" db:"contest_won"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// 常量定义
const (
	// 用户角色
	RoleStudent = "student"
	RoleTeacher = "teacher"
	RoleAdmin   = "admin"

	// 用户状态
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusBanned   = "banned"

	// 登录类型
	LoginTypePassword     = "password"
	LoginTypeRefreshToken = "refresh_token"
	LoginTypeOAuth        = "oauth"

	// 登录状态
	LoginStatusSuccess = "success"
	LoginStatusFailed  = "failed"
	LoginStatusBlocked = "blocked"

	// 段位等级
	RankBronze   = "bronze"
	RankSilver   = "silver"
	RankGold     = "gold"
	RankPlatinum = "platinum"
	RankDiamond  = "diamond"
)

// 权限定义
var RolePermissions = map[string][]string{
	RoleStudent: {
		"user:profile:read",
		"user:profile:update",
		"user:password:change",
		"problem:read",
		"submission:create",
		"submission:read:own",
		"contest:participate",
	},
	RoleTeacher: {
		"user:profile:read",
		"user:profile:update", 
		"user:password:change",
		"problem:read",
		"problem:create",
		"problem:update:own",
		"submission:create",
		"submission:read:own",
		"submission:read:all",
		"contest:create",
		"contest:manage:own",
		"contest:participate",
	},
	RoleAdmin: {
		"user:*",
		"problem:*",
		"submission:*",
		"contest:*",
		"system:*",
	},
}