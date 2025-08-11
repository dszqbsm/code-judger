package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/online-judge/code-judger/common/types"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserLoginLogModel = (*customUserLoginLogModel)(nil)

type (
	// UserLoginLogModel 用户登录日志模型接口
	UserLoginLogModel interface {
		userLoginLogModel
		// 自定义方法
		FindByUserID(ctx context.Context, userID int64, limit int64) ([]*types.UserLoginLog, error)
		GetRecentLoginsByIP(ctx context.Context, ip string, limit int64) ([]*types.UserLoginLog, error)
	}

	customUserLoginLogModel struct {
		*defaultUserLoginLogModel
	}

	userLoginLogModel interface {
		Insert(ctx context.Context, data *types.UserLoginLog) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*types.UserLoginLog, error)
		Delete(ctx context.Context, id int64) error
	}

	defaultUserLoginLogModel struct {
		sqlc.CachedConn
		table string
	}
)

// NewUserLoginLogModel 创建用户登录日志模型
func NewUserLoginLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserLoginLogModel {
	return &customUserLoginLogModel{
		defaultUserLoginLogModel: newUserLoginLogModel(conn, c, opts...),
	}
}

func newUserLoginLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) *defaultUserLoginLogModel {
	return &defaultUserLoginLogModel{
		CachedConn: sqlc.NewConn(conn, c, opts...),
		table:      "`user_login_logs`",
	}
}

// Insert 插入登录日志记录
func (m *defaultUserLoginLogModel) Insert(ctx context.Context, data *types.UserLoginLog) (sql.Result, error) {
	query := fmt.Sprintf("INSERT INTO %s (`user_id`, `login_type`, `ip_address`, `user_agent`, `login_status`, `failure_reason`, `location_info`, `device_info`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", m.table)
	return m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, data.UserID, data.LoginType, data.IPAddress, data.UserAgent, data.LoginStatus, data.FailureReason, data.LocationInfo, data.DeviceInfo)
	})
}

// FindOne 根据ID查找登录日志
func (m *defaultUserLoginLogModel) FindOne(ctx context.Context, id int64) (*types.UserLoginLog, error) {
	loginLogIdKey := fmt.Sprintf("%s%v", cacheLoginLogIdPrefix, id)
	var resp types.UserLoginLog
	err := m.CachedConn.QueryRowCtx(ctx, &resp, loginLogIdKey, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE `id` = ? LIMIT 1", loginLogRows, m.table)
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

// Delete 删除登录日志
func (m *defaultUserLoginLogModel) Delete(ctx context.Context, id int64) error {
	loginLogIdKey := fmt.Sprintf("%s%v", cacheLoginLogIdPrefix, id)
	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("DELETE FROM %s WHERE `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, id)
	}, loginLogIdKey)
	return err
}

// 自定义方法实现

// FindByUserID 根据用户ID查找登录日志
func (m *customUserLoginLogModel) FindByUserID(ctx context.Context, userID int64, limit int64) ([]*types.UserLoginLog, error) {
	var logs []*types.UserLoginLog
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `user_id` = ? ORDER BY `created_at` DESC LIMIT ?", loginLogRows, m.table)
	err := m.CachedConn.QueryRowsNoCacheCtx(ctx, &logs, query, userID, limit)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// GetRecentLoginsByIP 根据IP地址获取最近的登录记录
func (m *customUserLoginLogModel) GetRecentLoginsByIP(ctx context.Context, ip string, limit int64) ([]*types.UserLoginLog, error) {
	var logs []*types.UserLoginLog
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `ip_address` = ? ORDER BY `created_at` DESC LIMIT ?", loginLogRows, m.table)
	err := m.CachedConn.QueryRowsNoCacheCtx(ctx, &logs, query, ip, limit)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// 常量定义
var (
	cacheLoginLogIdPrefix = "cache:login_log:id:"
	loginLogRows          = "`id`, `user_id`, `login_type`, `ip_address`, `user_agent`, `login_status`, `failure_reason`, `location_info`, `device_info`, `created_at`"
)
