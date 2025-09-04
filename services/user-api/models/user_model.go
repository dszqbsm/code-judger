package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/dszqbsm/code-judger/common/types"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserModel = (*customUserModel)(nil)

type (
	// UserModel 用户模型接口
	UserModel interface {
		userModel
		// 自定义方法
		FindByUsername(ctx context.Context, username string) (*types.User, error)
		FindByEmail(ctx context.Context, email string) (*types.User, error)
		FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*types.User, error)
		UpdateLastLogin(ctx context.Context, userID int64, ip string) error
		UpdateLoginCount(ctx context.Context, userID int64) error
		GetUserList(ctx context.Context, page, pageSize int64, role, status, keyword string) ([]*types.User, int64, error)
		UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
		UpdateProfile(ctx context.Context, userID int64, realName, avatarUrl, bio string) error
		UpdateRole(ctx context.Context, userID int64, role string) error
		CheckUsernameExists(ctx context.Context, username string) (bool, error)
		CheckEmailExists(ctx context.Context, email string) (bool, error)
	}

	customUserModel struct {
		*defaultUserModel
	}

	userModel interface {
		Insert(ctx context.Context, data *types.User) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*types.User, error)
		FindOneByUsername(ctx context.Context, username string) (*types.User, error)
		FindOneByEmail(ctx context.Context, email string) (*types.User, error)
		Update(ctx context.Context, data *types.User) error
		Delete(ctx context.Context, id int64) error
	}

	defaultUserModel struct {
		sqlc.CachedConn
		table string
	}
)

// NewUserModel 创建用户模型
func NewUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn, c, opts...),
	}
}

func newUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) *defaultUserModel {
	return &defaultUserModel{
		CachedConn: sqlc.NewConn(conn, c, opts...),
		table:      "`users`",
	}
}

// Insert 插入用户记录
func (m *defaultUserModel) Insert(ctx context.Context, data *types.User) (sql.Result, error) {
	query := fmt.Sprintf("INSERT INTO %s (`username`, `email`, `password_hash`, `real_name`, `avatar_url`, `bio`, `role`, `status`, `email_verified`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table)
	return m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, data.Username, data.Email, data.PasswordHash, data.RealName, data.AvatarUrl, data.Bio, data.Role, data.Status, data.EmailVerified)
	})
}

// FindOne 根据ID查找用户
func (m *defaultUserModel) FindOne(ctx context.Context, id int64) (*types.User, error) {
	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, id)
	var resp types.User
	err := m.CachedConn.QueryRowCtx(ctx, &resp, userIdKey, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE `id` = ? LIMIT 1", userRows, m.table)
		return conn.QueryRowCtx(ctx, v, query, id)
	})
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

// FindOneByUsername 根据用户名查找用户
func (m *defaultUserModel) FindOneByUsername(ctx context.Context, username string) (*types.User, error) {
	usernameKey := fmt.Sprintf("%s%v", cacheUserUsernamePrefix, username)
	var resp types.User
	err := m.CachedConn.QueryRowIndexCtx(ctx, &resp, usernameKey, m.formatPrimary, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) (i interface{}, e error) {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE `username` = ? LIMIT 1", userRows, m.table)
		if err := conn.QueryRowCtx(ctx, &resp, query, username); err != nil {
			return nil, err
		}
		return resp.ID, nil
	}, m.queryPrimary)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

// FindOneByEmail 根据邮箱查找用户
func (m *defaultUserModel) FindOneByEmail(ctx context.Context, email string) (*types.User, error) {
	emailKey := fmt.Sprintf("%s%v", cacheUserEmailPrefix, email)
	var resp types.User
	err := m.CachedConn.QueryRowIndexCtx(ctx, &resp, emailKey, m.formatPrimary, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) (i interface{}, e error) {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE `email` = ? LIMIT 1", userRows, m.table)
		if err := conn.QueryRowCtx(ctx, &resp, query, email); err != nil {
			return nil, err
		}
		return resp.ID, nil
	}, m.queryPrimary)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

// Update 更新用户信息
func (m *defaultUserModel) Update(ctx context.Context, newData *types.User) error {
	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, newData.ID)
	usernameKey := fmt.Sprintf("%s%v", cacheUserUsernamePrefix, newData.Username)
	emailKey := fmt.Sprintf("%s%v", cacheUserEmailPrefix, newData.Email)

	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("UPDATE %s SET `username` = ?, `email` = ?, `password_hash` = ?, `real_name` = ?, `avatar_url` = ?, `bio` = ?, `role` = ?, `status` = ?, `email_verified` = ?, `last_login_at` = ?, `last_login_ip` = ?, `login_count` = ?, `updated_at` = ? WHERE `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, newData.Username, newData.Email, newData.PasswordHash, newData.RealName, newData.AvatarUrl, newData.Bio, newData.Role, newData.Status, newData.EmailVerified, newData.LastLoginAt, newData.LastLoginIP, newData.LoginCount, time.Now(), newData.ID)
	}, userIdKey, usernameKey, emailKey)
	return err
}

// Delete 删除用户
func (m *defaultUserModel) Delete(ctx context.Context, id int64) error {
	data, err := m.FindOne(ctx, id)
	if err != nil {
		return err
	}

	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, id)
	usernameKey := fmt.Sprintf("%s%v", cacheUserUsernamePrefix, data.Username)
	emailKey := fmt.Sprintf("%s%v", cacheUserEmailPrefix, data.Email)

	_, err = m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("DELETE FROM %s WHERE `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, id)
	}, userIdKey, usernameKey, emailKey)
	return err
}

// 自定义方法实现

// FindByUsername 根据用户名查找用户
func (m *customUserModel) FindByUsername(ctx context.Context, username string) (*types.User, error) {
	return m.FindOneByUsername(ctx, username)
}

// FindByEmail 根据邮箱查找用户
func (m *customUserModel) FindByEmail(ctx context.Context, email string) (*types.User, error) {
	return m.FindOneByEmail(ctx, email)
}

// FindByUsernameOrEmail 根据用户名或邮箱查找用户
func (m *customUserModel) FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*types.User, error) {
	var resp types.User
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `username` = ? OR `email` = ? LIMIT 1", userRows, m.table)
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &resp, query, usernameOrEmail, usernameOrEmail)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

// UpdateLastLogin 更新最后登录信息
func (m *customUserModel) UpdateLastLogin(ctx context.Context, userID int64, ip string) error {
	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, userID)
	query := fmt.Sprintf("UPDATE %s SET `last_login_at` = ?, `last_login_ip` = ?, `login_count` = `login_count` + 1, `updated_at` = ? WHERE `id` = ?", m.table)
	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		return conn.ExecCtx(ctx, query, time.Now(), ip, time.Now(), userID)
	}, userIdKey)
	return err
}

// UpdateLoginCount 更新登录次数
func (m *customUserModel) UpdateLoginCount(ctx context.Context, userID int64) error {
	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, userID)
	query := fmt.Sprintf("UPDATE %s SET `login_count` = `login_count` + 1, `updated_at` = ? WHERE `id` = ?", m.table)
	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		return conn.ExecCtx(ctx, query, time.Now(), userID)
	}, userIdKey)
	return err
}

// GetUserList 获取用户列表
func (m *customUserModel) GetUserList(ctx context.Context, page, pageSize int64, role, status, keyword string) ([]*types.User, int64, error) {
	var whereClause []string
	var args []interface{}

	if role != "" {
		whereClause = append(whereClause, "`role` = ?")
		args = append(args, role)
	}

	if status != "" {
		whereClause = append(whereClause, "`status` = ?")
		args = append(args, status)
	}

	if keyword != "" {
		whereClause = append(whereClause, "(`username` LIKE ? OR `email` LIKE ? OR `real_name` LIKE ?)")
		searchKeyword := "%" + keyword + "%"
		args = append(args, searchKeyword, searchKeyword, searchKeyword)
	}

	whereSQL := ""
	if len(whereClause) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClause, " AND ")
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", m.table, whereSQL)
	var total int64
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// 查询数据
	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf("SELECT %s FROM %s %s ORDER BY `created_at` DESC LIMIT ? OFFSET ?", userRows, m.table, whereSQL)
	listArgs := append(args, pageSize, offset)

	var users []*types.User
	err = m.CachedConn.QueryRowsNoCacheCtx(ctx, &users, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdatePassword 更新密码
func (m *customUserModel) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, userID)
	query := fmt.Sprintf("UPDATE %s SET `password_hash` = ?, `updated_at` = ? WHERE `id` = ?", m.table)
	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		return conn.ExecCtx(ctx, query, passwordHash, time.Now(), userID)
	}, userIdKey)
	return err
}

// UpdateProfile 更新用户资料
func (m *customUserModel) UpdateProfile(ctx context.Context, userID int64, realName, avatarUrl, bio string) error {
	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, userID)
	query := fmt.Sprintf("UPDATE %s SET `real_name` = ?, `avatar_url` = ?, `bio` = ?, `updated_at` = ? WHERE `id` = ?", m.table)
	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		return conn.ExecCtx(ctx, query, realName, avatarUrl, bio, time.Now(), userID)
	}, userIdKey)
	return err
}

// UpdateRole 更新用户角色
func (m *customUserModel) UpdateRole(ctx context.Context, userID int64, role string) error {
	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, userID)
	query := fmt.Sprintf("UPDATE %s SET `role` = ?, `updated_at` = ? WHERE `id` = ?", m.table)
	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		return conn.ExecCtx(ctx, query, role, time.Now(), userID)
	}, userIdKey)
	return err
}

// CheckUsernameExists 检查用户名是否存在
func (m *customUserModel) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE `username` = ?", m.table)
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &count, query, username)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CheckEmailExists 检查邮箱是否存在
func (m *customUserModel) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE `email` = ?", m.table)
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &count, query, email)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 辅助方法
func (m *defaultUserModel) formatPrimary(primary interface{}) string {
	return fmt.Sprintf("%s%v", cacheUserIdPrefix, primary)
}

func (m *defaultUserModel) queryPrimary(ctx context.Context, conn sqlx.SqlConn, v, primary interface{}) error {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `id` = ? LIMIT 1", userRows, m.table)
	return conn.QueryRowCtx(ctx, v, query, primary)
}

// 常量定义
var (
	cacheUserIdPrefix       = "cache:user:id:"
	cacheUserUsernamePrefix = "cache:user:username:"
	cacheUserEmailPrefix    = "cache:user:email:"
	userRows                = "`id`, `username`, `email`, `password_hash`, `real_name`, `avatar_url`, `bio`, `role`, `status`, `email_verified`, `last_login_at`, `last_login_ip`, `login_count`, `created_at`, `updated_at`"
	ErrNotFound             = sqlc.ErrNotFound
)
