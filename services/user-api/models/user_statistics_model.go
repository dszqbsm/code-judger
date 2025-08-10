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

var _ UserStatisticsModel = (*customUserStatisticsModel)(nil)

type (
	// UserStatisticsModel 用户统计模型接口
	UserStatisticsModel interface {
		userStatisticsModel
		// 自定义方法
		FindByUserID(ctx context.Context, userID int64) (*types.UserStatistics, error)
		UpdateStats(ctx context.Context, userID int64, stats *types.UserStatistics) error
	}

	customUserStatisticsModel struct {
		*defaultUserStatisticsModel
	}

	userStatisticsModel interface {
		Insert(ctx context.Context, data *types.UserStatistics) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*types.UserStatistics, error)
		Update(ctx context.Context, data *types.UserStatistics) error
		Delete(ctx context.Context, id int64) error
	}

	defaultUserStatisticsModel struct {
		sqlc.CachedConn
		table string
	}
)

// NewUserStatisticsModel 创建用户统计模型
func NewUserStatisticsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserStatisticsModel {
	return &customUserStatisticsModel{
		defaultUserStatisticsModel: newUserStatisticsModel(conn, c, opts...),
	}
}

func newUserStatisticsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) *defaultUserStatisticsModel {
	return &defaultUserStatisticsModel{
		CachedConn: sqlc.NewConn(conn, c, opts...),
		table:      "`user_statistics`",
	}
}

// Insert 插入用户统计记录
func (m *defaultUserStatisticsModel) Insert(ctx context.Context, data *types.UserStatistics) (sql.Result, error) {
	query := fmt.Sprintf("INSERT INTO %s (`user_id`, `current_rating`, `max_rating`, `rank_level`) VALUES (?, ?, ?, ?)", m.table)
	return m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, data.UserID, data.CurrentRating, data.MaxRating, data.RankLevel)
	})
}

// FindOne 根据ID查找用户统计
func (m *defaultUserStatisticsModel) FindOne(ctx context.Context, id int64) (*types.UserStatistics, error) {
	userStatsIdKey := fmt.Sprintf("%s%v", cacheUserStatsIdPrefix, id)
	var resp types.UserStatistics
	err := m.QueryRowCtx(ctx, &resp, userStatsIdKey, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE `id` = ? LIMIT 1", userStatsRows, m.table)
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

// Update 更新用户统计
func (m *defaultUserStatisticsModel) Update(ctx context.Context, newData *types.UserStatistics) error {
	userStatsIdKey := fmt.Sprintf("%s%v", cacheUserStatsIdPrefix, newData.ID)
	userStatsUserIdKey := fmt.Sprintf("%s%v", cacheUserStatsUserIdPrefix, newData.UserID)
	
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("UPDATE %s SET `total_submissions` = ?, `accepted_submissions` = ?, `solved_problems` = ?, `current_rating` = ?, `max_rating` = ?, `rank_level` = ? WHERE `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, newData.TotalSubmissions, newData.AcceptedSubmissions, newData.SolvedProblems, newData.CurrentRating, newData.MaxRating, newData.RankLevel, newData.ID)
	}, userStatsIdKey, userStatsUserIdKey)
	return err
}

// Delete 删除用户统计
func (m *defaultUserStatisticsModel) Delete(ctx context.Context, id int64) error {
	data, err := m.FindOne(ctx, id)
	if err != nil {
		return err
	}

	userStatsIdKey := fmt.Sprintf("%s%v", cacheUserStatsIdPrefix, id)
	userStatsUserIdKey := fmt.Sprintf("%s%v", cacheUserStatsUserIdPrefix, data.UserID)
	
	_, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("DELETE FROM %s WHERE `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, id)
	}, userStatsIdKey, userStatsUserIdKey)
	return err
}

// 自定义方法实现

// FindByUserID 根据用户ID查找统计信息
func (m *customUserStatisticsModel) FindByUserID(ctx context.Context, userID int64) (*types.UserStatistics, error) {
	userStatsUserIdKey := fmt.Sprintf("%s%v", cacheUserStatsUserIdPrefix, userID)
	var resp types.UserStatistics
	err := m.QueryRowIndexCtx(ctx, &resp, userStatsUserIdKey, m.formatPrimary, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) (i interface{}, e error) {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE `user_id` = ? LIMIT 1", userStatsRows, m.table)
		if err := conn.QueryRowCtx(ctx, &resp, query, userID); err != nil {
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

// UpdateStats 更新用户统计信息
func (m *customUserStatisticsModel) UpdateStats(ctx context.Context, userID int64, stats *types.UserStatistics) error {
	userStatsUserIdKey := fmt.Sprintf("%s%v", cacheUserStatsUserIdPrefix, userID)
	query := fmt.Sprintf("UPDATE %s SET `total_submissions` = ?, `accepted_submissions` = ?, `solved_problems` = ? WHERE `user_id` = ?", m.table)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		return conn.ExecCtx(ctx, query, stats.TotalSubmissions, stats.AcceptedSubmissions, stats.SolvedProblems, userID)
	}, userStatsUserIdKey)
	return err
}

// 辅助方法
func (m *defaultUserStatisticsModel) formatPrimary(primary interface{}) string {
	return fmt.Sprintf("%s%v", cacheUserStatsIdPrefix, primary)
}

func (m *defaultUserStatisticsModel) queryPrimary(ctx context.Context, conn sqlx.SqlConn, v, primary interface{}) error {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `id` = ? LIMIT 1", userStatsRows, m.table)
	return conn.QueryRowCtx(ctx, v, query, primary)
}

// 常量定义
var (
	cacheUserStatsIdPrefix     = "cache:user_stats:id:"
	cacheUserStatsUserIdPrefix = "cache:user_stats:user_id:"
	userStatsRows              = "`id`, `user_id`, `total_submissions`, `accepted_submissions`, `solved_problems`, `easy_solved`, `medium_solved`, `hard_solved`, `current_rating`, `max_rating`, `rank_level`, `total_code_time`, `average_solve_time`, `contest_participated`, `contest_won`, `created_at`, `updated_at`"
)