package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dszqbsm/code-judger/common/types"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserTokenModel = (*customUserTokenModel)(nil)

type (
	// UserTokenModel 用户令牌模型接口
	UserTokenModel interface {
		userTokenModel
		// 自定义方法
		FindByTokenID(ctx context.Context, tokenID string) (*types.UserToken, error)
		FindByUserID(ctx context.Context, userID int64) ([]*types.UserToken, error)
		RevokeToken(ctx context.Context, tokenID string) error
		RevokeUserTokens(ctx context.Context, userID int64) error
		CleanExpiredTokens(ctx context.Context) error
		IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
	}

	customUserTokenModel struct {
		*defaultUserTokenModel
	}

	userTokenModel interface {
		Insert(ctx context.Context, data *types.UserToken) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*types.UserToken, error)
		Update(ctx context.Context, data *types.UserToken) error
		Delete(ctx context.Context, id int64) error
	}

	defaultUserTokenModel struct {
		sqlc.CachedConn
		table string
	}
)

// NewUserTokenModel 创建用户令牌模型
func NewUserTokenModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserTokenModel {
	return &customUserTokenModel{
		defaultUserTokenModel: newUserTokenModel(conn, c, opts...),
	}
}

func newUserTokenModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) *defaultUserTokenModel {
	return &defaultUserTokenModel{
		CachedConn: sqlc.NewConn(conn, c, opts...),
		table:      "`user_tokens`",
	}
}

// Insert 插入令牌记录
func (m *defaultUserTokenModel) Insert(ctx context.Context, data *types.UserToken) (sql.Result, error) {
	query := fmt.Sprintf("INSERT INTO %s (`user_id`, `token_id`, `refresh_token`, `access_token_expire`, `refresh_token_expire`, `client_info`) VALUES (?, ?, ?, ?, ?, ?)", m.table)
	return m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, data.UserID, data.TokenID, data.RefreshToken, data.AccessTokenExpire, data.RefreshTokenExpire, data.ClientInfo)
	})
}

// FindOne 根据ID查找令牌
func (m *defaultUserTokenModel) FindOne(ctx context.Context, id int64) (*types.UserToken, error) {
	tokenIdKey := fmt.Sprintf("%s%v", cacheUserTokenIdPrefix, id)
	var resp types.UserToken
	err := m.CachedConn.QueryRowCtx(ctx, &resp, tokenIdKey, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE `id` = ? LIMIT 1", userTokenRows, m.table)
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

// Update 更新令牌
func (m *defaultUserTokenModel) Update(ctx context.Context, newData *types.UserToken) error {
	tokenIdKey := fmt.Sprintf("%s%v", cacheUserTokenIdPrefix, newData.ID)
	tokenKey := fmt.Sprintf("%s%v", cacheUserTokenTokenIdPrefix, newData.TokenID)

	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("UPDATE %s SET `user_id` = ?, `token_id` = ?, `refresh_token` = ?, `access_token_expire` = ?, `refresh_token_expire` = ?, `client_info` = ?, `is_revoked` = ?, `updated_at` = ? WHERE `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, newData.UserID, newData.TokenID, newData.RefreshToken, newData.AccessTokenExpire, newData.RefreshTokenExpire, newData.ClientInfo, newData.IsRevoked, time.Now(), newData.ID)
	}, tokenIdKey, tokenKey)
	return err
}

// Delete 删除令牌
func (m *defaultUserTokenModel) Delete(ctx context.Context, id int64) error {
	data, err := m.FindOne(ctx, id)
	if err != nil {
		return err
	}

	tokenIdKey := fmt.Sprintf("%s%v", cacheUserTokenIdPrefix, id)
	tokenKey := fmt.Sprintf("%s%v", cacheUserTokenTokenIdPrefix, data.TokenID)

	_, err = m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("DELETE FROM %s WHERE `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, id)
	}, tokenIdKey, tokenKey)
	return err
}

// 自定义方法实现

// FindByTokenID 根据令牌ID查找
func (m *customUserTokenModel) FindByTokenID(ctx context.Context, tokenID string) (*types.UserToken, error) {
	tokenKey := fmt.Sprintf("%s%v", cacheUserTokenTokenIdPrefix, tokenID)
	var resp types.UserToken
	err := m.CachedConn.QueryRowIndexCtx(ctx, &resp, tokenKey, m.formatPrimary, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) (i interface{}, e error) {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE `token_id` = ? AND `is_revoked` = false AND `refresh_token_expire` > ? LIMIT 1", userTokenRows, m.table)
		if err := conn.QueryRowCtx(ctx, &resp, query, tokenID, time.Now()); err != nil {
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

// FindByUserID 根据用户ID查找所有令牌
func (m *customUserTokenModel) FindByUserID(ctx context.Context, userID int64) ([]*types.UserToken, error) {
	var tokens []*types.UserToken
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `user_id` = ? AND `is_revoked` = false ORDER BY `created_at` DESC", userTokenRows, m.table)
	err := m.CachedConn.QueryRowsNoCacheCtx(ctx, &tokens, query, userID)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// RevokeToken 撤销指定令牌
func (m *customUserTokenModel) RevokeToken(ctx context.Context, tokenID string) error {
	tokenKey := fmt.Sprintf("%s%v", cacheUserTokenTokenIdPrefix, tokenID)
	query := fmt.Sprintf("UPDATE %s SET `is_revoked` = true, `updated_at` = ? WHERE `token_id` = ?", m.table)
	_, err := m.CachedConn.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		return conn.ExecCtx(ctx, query, time.Now(), tokenID)
	}, tokenKey)
	return err
}

// RevokeUserTokens 撤销用户所有令牌
func (m *customUserTokenModel) RevokeUserTokens(ctx context.Context, userID int64) error {
	query := fmt.Sprintf("UPDATE %s SET `is_revoked` = true, `updated_at` = ? WHERE `user_id` = ? AND `is_revoked` = false", m.table)
	_, err := m.CachedConn.ExecNoCacheCtx(ctx, query, time.Now(), userID)
	return err
}

// CleanExpiredTokens 清理过期令牌
func (m *customUserTokenModel) CleanExpiredTokens(ctx context.Context) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE `refresh_token_expire` < ?", m.table)
	_, err := m.CachedConn.ExecNoCacheCtx(ctx, query, time.Now())
	return err
}

// IsTokenRevoked 检查令牌是否已撤销
func (m *customUserTokenModel) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	var isRevoked bool
	query := fmt.Sprintf("SELECT `is_revoked` FROM %s WHERE `token_id` = ? LIMIT 1", m.table)
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &isRevoked, query, tokenID)
	if err == sqlc.ErrNotFound {
		return true, nil // 令牌不存在视为已撤销
	}
	if err != nil {
		return false, err
	}
	return isRevoked, nil
}

// 辅助方法
func (m *defaultUserTokenModel) formatPrimary(primary interface{}) string {
	return fmt.Sprintf("%s%v", cacheUserTokenIdPrefix, primary)
}

func (m *defaultUserTokenModel) queryPrimary(ctx context.Context, conn sqlx.SqlConn, v, primary interface{}) error {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `id` = ? LIMIT 1", userTokenRows, m.table)
	return conn.QueryRowCtx(ctx, v, query, primary)
}

// 常量定义
var (
	cacheUserTokenIdPrefix      = "cache:user_token:id:"
	cacheUserTokenTokenIdPrefix = "cache:user_token:token_id:"
	userTokenRows               = "`id`, `user_id`, `token_id`, `refresh_token`, `access_token_expire`, `refresh_token_expire`, `client_info`, `is_revoked`, `created_at`, `updated_at`"
)
