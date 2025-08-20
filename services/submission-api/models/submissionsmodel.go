package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/builder"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/stringx"
)

var (
	submissionFieldNames          = builder.RawFieldNames(&Submission{})
	submissionRows                = strings.Join(submissionFieldNames, ",")
	submissionRowsExpectAutoSet   = strings.Join(stringx.Remove(submissionFieldNames, "`id`", "`created_at`"), ",")
	submissionRowsWithPlaceHolder = strings.Join(stringx.Remove(submissionFieldNames, "`id`", "`created_at`"), "=?,") + "=?"

	cacheSubmissionIdPrefix = "cache:submission:id:"
)

type (
	submissionModel interface {
		Insert(ctx context.Context, data *Submission) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*Submission, error)
		Update(ctx context.Context, data *Submission) error
		Delete(ctx context.Context, id int64) error
		BatchUpdateStatus(ctx context.Context, updates []StatusUpdate) error
	}
	


	defaultSubmissionModel struct {
		sqlc.CachedConn
		table string
	}

	// Submission 根据实际数据库表结构定义
	Submission struct {
		Id              int64          `db:"id"`
		UserId          int64          `db:"user_id"`
		ProblemId       int64          `db:"problem_id"`
		ContestId       sql.NullInt64  `db:"contest_id"`
		Language        string         `db:"language"`
		Code            string         `db:"code"`
		CodeLength      int64          `db:"code_length"`
		Status          string         `db:"status"`
		TimeUsed        int64          `db:"time_used"`
		MemoryUsed      int64          `db:"memory_used"`
		Score           int64          `db:"score"`
		CompileInfo     sql.NullString `db:"compile_info"`
		RuntimeInfo     sql.NullString `db:"runtime_info"`
		TestCaseResults sql.NullString `db:"test_case_results"`
		JudgeServer     sql.NullString `db:"judge_server"`
		IpAddress       sql.NullString `db:"ip_address"`
		CreatedAt       time.Time      `db:"created_at"`
		JudgedAt        sql.NullTime   `db:"judged_at"`
	}
)

func newSubmissionModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) *defaultSubmissionModel {
	return &defaultSubmissionModel{
		CachedConn: sqlc.NewConn(conn, c, opts...),
		table:      "`submissions`",
	}
}



func (m *defaultSubmissionModel) Delete(ctx context.Context, id int64) error {
	submissionIdKey := fmt.Sprintf("%s%v", cacheSubmissionIdPrefix, id)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("delete from %s where `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, id)
	}, submissionIdKey)
	return err
}

func (m *defaultSubmissionModel) FindOne(ctx context.Context, id int64) (*Submission, error) {
	submissionIdKey := fmt.Sprintf("%s%v", cacheSubmissionIdPrefix, id)
	var resp Submission
	err := m.QueryRowCtx(ctx, &resp, submissionIdKey, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
		query := fmt.Sprintf("select %s from %s where `id` = ? limit 1", submissionRows, m.table)
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

func (m *defaultSubmissionModel) Insert(ctx context.Context, data *Submission) (sql.Result, error) {
	submissionIdKey := fmt.Sprintf("%s%v", cacheSubmissionIdPrefix, data.Id)
	ret, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table, submissionRowsExpectAutoSet)
		return conn.ExecCtx(ctx, query, data.UserId, data.ProblemId, data.ContestId, data.Language, data.Code, data.CodeLength, data.Status, data.TimeUsed, data.MemoryUsed, data.Score, data.CompileInfo, data.RuntimeInfo, data.TestCaseResults, data.JudgeServer, data.IpAddress)
	}, submissionIdKey)
	return ret, err
}

func (m *defaultSubmissionModel) Update(ctx context.Context, data *Submission) error {
	submissionIdKey := fmt.Sprintf("%s%v", cacheSubmissionIdPrefix, data.Id)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("update %s set %s where `id` = ?", m.table, submissionRowsWithPlaceHolder)
		return conn.ExecCtx(ctx, query, data.UserId, data.ProblemId, data.ContestId, data.Language, data.Code, data.CodeLength, data.Status, data.TimeUsed, data.MemoryUsed, data.Score, data.CompileInfo, data.RuntimeInfo, data.TestCaseResults, data.JudgeServer, data.IpAddress, data.Id)
	}, submissionIdKey)
	return err
}

func (m *defaultSubmissionModel) formatPrimary(primary interface{}) string {
	return fmt.Sprintf("%s%v", cacheSubmissionIdPrefix, primary)
}

func (m *defaultSubmissionModel) queryPrimary(ctx context.Context, conn sqlx.SqlConn, v, primary interface{}) error {
	query := fmt.Sprintf("select %s from %s where `id` = ? limit 1", submissionRows, m.table)
	return conn.QueryRowCtx(ctx, v, query, primary)
}

func (m *defaultSubmissionModel) tableName() string {
	return m.table
}

func (m *defaultSubmissionModel) BatchUpdateStatus(ctx context.Context, updates []StatusUpdate) error {
	// 简单实现，循环更新每个记录
	for _, update := range updates {
		submissionIdKey := fmt.Sprintf("%s%v", cacheSubmissionIdPrefix, update.SubmissionID)
		_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
			query := fmt.Sprintf("update %s set `status` = ? where `id` = ?", m.table)
			return conn.ExecCtx(ctx, query, update.Status, update.SubmissionID)
		}, submissionIdKey)
		if err != nil {
			return err
		}
	}
	return nil
}