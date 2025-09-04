package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SubmissionModel = (*customSubmissionModel)(nil)

type (
	// SubmissionModel 提交模型接口
	SubmissionModel interface {
		submissionModel
		FindByUserID(ctx context.Context, userID int64, page, limit int) ([]*Submission, error)
		FindByProblemID(ctx context.Context, problemID int64, page, limit int) ([]*Submission, error)
		FindByContestID(ctx context.Context, contestID int64, page, limit int) ([]*Submission, error)
		UpdateStatus(ctx context.Context, id int64, status string) error
		UpdateResult(ctx context.Context, id int64, result *JudgeResult) error
		CountByProblemID(ctx context.Context, problemID int64) (int64, error)
		CountByUserID(ctx context.Context, userID int64) (int64, error)
		GetUserSubmissionStats(ctx context.Context, userID int64) (*UserSubmissionStats, error)
		Search(ctx context.Context, condition *SearchCondition) ([]*Submission, int64, error)
	}

	submissionModel interface {
		Insert(ctx context.Context, data *Submission) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*Submission, error)
		Update(ctx context.Context, data *Submission) error
		Delete(ctx context.Context, id int64) error
	}

	customSubmissionModel struct {
		*defaultSubmissionModel
	}

	defaultSubmissionModel struct {
		sqlc.CachedConn
		table string
	}

	// Submission 提交记录结构 - 基于实际数据库表结构
	Submission struct {
		ID               int64          `db:"id"`
		UserID           int64          `db:"user_id"`
		ProblemID        int64          `db:"problem_id"`
		ContestID        sql.NullInt64  `db:"contest_id"`
		Language         string         `db:"language"`
		Code             string         `db:"code"`
		CodeLength       sql.NullInt32  `db:"code_length"`
		Status           string         `db:"status"`
		TimeUsed         sql.NullInt32  `db:"time_used"`
		MemoryUsed       sql.NullInt32  `db:"memory_used"`
		Score            sql.NullInt32  `db:"score"`
		CompileInfo      sql.NullString `db:"compile_info"`      // 编译信息
		RuntimeInfo      sql.NullString `db:"runtime_info"`      // 运行时信息
		TestCaseResults  sql.NullString `db:"test_case_results"` // JSON格式的测试用例结果
		JudgeServer      sql.NullString `db:"judge_server"`      // 判题服务器
		IPAddress        sql.NullString `db:"ip_address"`        // 提交IP地址
		CreatedAt        sql.NullTime   `db:"created_at"`        // 创建时间
		JudgedAt         sql.NullTime   `db:"judged_at"`         // 判题完成时间
	}

	// JudgeResult 判题结果
	JudgeResult struct {
		Status          string `json:"status"`
		Score           int    `json:"score"`
		TimeUsed        int    `json:"time_used"`
		MemoryUsed      int    `json:"memory_used"`
		CompileOutput   string `json:"compile_output"`
		RuntimeOutput   string `json:"runtime_output"`
		ErrorMessage    string `json:"error_message"`
		TestCasesPassed int    `json:"test_cases_passed"`
		TestCasesTotal  int    `json:"test_cases_total"`
		JudgeServer     string `json:"judge_server"`
	}

	// UserSubmissionStats 用户提交统计
	UserSubmissionStats struct {
		TotalSubmissions    int64   `db:"total_submissions"`
		AcceptedSubmissions int64   `db:"accepted_submissions"`
		WrongSubmissions    int64   `db:"wrong_submissions"`
		CompileErrorCount   int64   `db:"compile_error_count"`
		RuntimeErrorCount   int64   `db:"runtime_error_count"`
		TimeLimitExceeded   int64   `db:"time_limit_exceeded"`
		MemoryLimitExceeded int64   `db:"memory_limit_exceeded"`
		AcceptanceRate      float64 `db:"acceptance_rate"`
	}

	// SubmissionFilters 查询过滤器
	SubmissionFilters struct {
		UserID    *int64
		ProblemID *int64
		ContestID *int64
		Language  *string
		Status    *string
		StartTime *time.Time
		EndTime   *time.Time
	}

	// SearchCondition 搜索条件
	SearchCondition struct {
		Page      int
		PageSize  int
		UserID    *int64
		ProblemID *int64
		ContestID *int64
		Status    string
		Language  string
	}
)

var ErrNotFound = sqlc.ErrNotFound

// NewSubmissionModel 创建提交模型实例
func NewSubmissionModel(conn sqlx.SqlConn, c cache.CacheConf) SubmissionModel {
	return &customSubmissionModel{
		defaultSubmissionModel: newSubmissionModel(conn, c),
	}
}

func newSubmissionModel(conn sqlx.SqlConn, c cache.CacheConf) *defaultSubmissionModel {
	return &defaultSubmissionModel{
		CachedConn: sqlc.NewConn(conn, c),
		table:      "`submissions`",
	}
}

// 基础CRUD方法
func (m *defaultSubmissionModel) Insert(ctx context.Context, data *Submission) (sql.Result, error) {
	// 设置创建时间（如果数据库字段允许NULL）
	now := time.Now()
	if !data.CreatedAt.Valid {
		data.CreatedAt = sql.NullTime{Time: now, Valid: true}
	}

	query := fmt.Sprintf("insert into %s (user_id, problem_id, contest_id, language, code, code_length, status, score, time_used, memory_used, compile_info, runtime_info, test_case_results, judge_server, ip_address, judged_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table)

	return m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, data.UserID, data.ProblemID, data.ContestID, data.Language, data.Code, data.CodeLength, data.Status, data.Score, data.TimeUsed, data.MemoryUsed, data.CompileInfo, data.RuntimeInfo, data.TestCaseResults, data.JudgeServer, data.IPAddress, data.JudgedAt)
	})
}

func (m *defaultSubmissionModel) FindOne(ctx context.Context, id int64) (*Submission, error) {
	var resp Submission
	query := fmt.Sprintf("select id, user_id, problem_id, contest_id, language, code, code_length, status, time_used, memory_used, score, compile_info, runtime_info, test_case_results, judge_server, ip_address, created_at, judged_at from %s where id = ? limit 1", m.table)

	err := m.QueryRowCtx(ctx, &resp, fmt.Sprintf("submission:id:%d", id), func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
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

func (m *defaultSubmissionModel) Update(ctx context.Context, data *Submission) error {
	query := fmt.Sprintf("update %s set user_id=?, problem_id=?, contest_id=?, language=?, code=?, code_length=?, status=?, time_used=?, memory_used=?, score=?, compile_info=?, runtime_info=?, test_case_results=?, judge_server=?, ip_address=?, judged_at=? where id=?", m.table)

	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, data.UserID, data.ProblemID, data.ContestID, data.Language, data.Code, data.CodeLength, data.Status, data.TimeUsed, data.MemoryUsed, data.Score, data.CompileInfo, data.RuntimeInfo, data.TestCaseResults, data.JudgeServer, data.IPAddress, data.JudgedAt, data.ID)
	}, fmt.Sprintf("submission:id:%d", data.ID))

	return err
}

func (m *defaultSubmissionModel) Delete(ctx context.Context, id int64) error {
	query := fmt.Sprintf("delete from %s where id = ?", m.table)

	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, id)
	}, fmt.Sprintf("submission:id:%d", id))

	return err
}

// 扩展方法
func (m *customSubmissionModel) FindByUserID(ctx context.Context, userID int64, page, limit int) ([]*Submission, error) {
	offset := (page - 1) * limit
		query := `SELECT id, user_id, problem_id, contest_id, language, code, code_length, status, 
		  score, time_used, memory_used, compile_info, runtime_info, test_case_results,
		  judge_server, ip_address, created_at, judged_at 
		  FROM submissions 
		  WHERE user_id = ? 
		  ORDER BY created_at DESC 
		  LIMIT ? OFFSET ?`

	var submissions []*Submission
	err := m.CachedConn.QueryRowsNoCacheCtx(ctx, &submissions, query, userID, limit, offset)
	return submissions, err
}

func (m *customSubmissionModel) FindByProblemID(ctx context.Context, problemID int64, page, limit int) ([]*Submission, error) {
	offset := (page - 1) * limit
		query := `SELECT id, user_id, problem_id, contest_id, language, code, code_length, status, 
		  score, time_used, memory_used, compile_info, runtime_info, test_case_results,
		  judge_server, ip_address, created_at, judged_at 
		  FROM submissions 
		  WHERE problem_id = ? 
		  ORDER BY created_at DESC 
		  LIMIT ? OFFSET ?`

	var submissions []*Submission
	err := m.CachedConn.QueryRowsNoCacheCtx(ctx, &submissions, query, problemID, limit, offset)
	return submissions, err
}

func (m *customSubmissionModel) FindByContestID(ctx context.Context, contestID int64, page, limit int) ([]*Submission, error) {
	offset := (page - 1) * limit
		query := `SELECT id, user_id, problem_id, contest_id, language, code, code_length, status, 
		  score, time_used, memory_used, compile_info, runtime_info, test_case_results,
		  judge_server, ip_address, created_at, judged_at 
		  FROM submissions 
		  WHERE contest_id = ? 
		  ORDER BY created_at DESC 
		  LIMIT ? OFFSET ?`

	var submissions []*Submission
	err := m.CachedConn.QueryRowsNoCacheCtx(ctx, &submissions, query, contestID, limit, offset)
	return submissions, err
}

func (m *customSubmissionModel) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE submissions SET status = ?, updated_at = ? WHERE id = ?`
	_, err := m.CachedConn.ExecNoCacheCtx(ctx, query, status, time.Now(), id)
	return err
}

func (m *customSubmissionModel) UpdateResult(ctx context.Context, id int64, result *JudgeResult) error {
	query := `UPDATE submissions SET 
			  status = ?, score = ?, time_used = ?, memory_used = ?, 
			  compile_output = ?, runtime_output = ?, error_message = ?,
			  test_cases_passed = ?, test_cases_total = ?, judge_server = ?,
			  judged_at = ?, updated_at = ?
			  WHERE id = ?`

	_, err := m.CachedConn.ExecNoCacheCtx(ctx, query,
		result.Status,
		result.Score,
		result.TimeUsed,
		result.MemoryUsed,
		result.CompileOutput,
		result.RuntimeOutput,
		result.ErrorMessage,
		result.TestCasesPassed,
		result.TestCasesTotal,
		result.JudgeServer,
		time.Now(), // judged_at
		time.Now(), // updated_at
		id,
	)
	return err
}

func (m *customSubmissionModel) CountByProblemID(ctx context.Context, problemID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM submissions WHERE problem_id = ?`
	var count int64
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &count, query, problemID)
	return count, err
}

func (m *customSubmissionModel) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM submissions WHERE user_id = ?`
	var count int64
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &count, query, userID)
	return count, err
}

func (m *customSubmissionModel) GetUserSubmissionStats(ctx context.Context, userID int64) (*UserSubmissionStats, error) {
	query := `SELECT 
			  COUNT(*) as total_submissions,
			  SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END) as accepted_submissions,
			  SUM(CASE WHEN status = 'wrong_answer' THEN 1 ELSE 0 END) as wrong_submissions,
			  SUM(CASE WHEN status = 'compile_error' THEN 1 ELSE 0 END) as compile_error_count,
			  SUM(CASE WHEN status = 'runtime_error' THEN 1 ELSE 0 END) as runtime_error_count,
			  SUM(CASE WHEN status = 'time_limit_exceeded' THEN 1 ELSE 0 END) as time_limit_exceeded,
			  SUM(CASE WHEN status = 'memory_limit_exceeded' THEN 1 ELSE 0 END) as memory_limit_exceeded,
			  ROUND(SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as acceptance_rate
			  FROM submissions WHERE user_id = ?`

	var stats UserSubmissionStats
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &stats, query, userID)
	return &stats, err
}

// Search 根据条件搜索提交记录
func (m *customSubmissionModel) Search(ctx context.Context, condition *SearchCondition) ([]*Submission, int64, error) {
	// 构建基础查询
		baseQuery := `SELECT id, user_id, problem_id, contest_id, language, code, code_length, status, 
			  score, time_used, memory_used, compile_info, runtime_info, test_case_results,
			  judge_server, ip_address, created_at, judged_at 
			  FROM submissions WHERE 1=1`

	countQuery := `SELECT COUNT(*) FROM submissions WHERE 1=1`

	var conditions []string
	var args []interface{}

	// 构建查询条件
	if condition.UserID != nil {
		conditions = append(conditions, "user_id = ?")
		args = append(args, *condition.UserID)
	}

	if condition.ProblemID != nil {
		conditions = append(conditions, "problem_id = ?")
		args = append(args, *condition.ProblemID)
	}

	if condition.ContestID != nil {
		conditions = append(conditions, "contest_id = ?")
		args = append(args, *condition.ContestID)
	}

	if condition.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, condition.Status)
	}

	if condition.Language != "" {
		conditions = append(conditions, "language = ?")
		args = append(args, condition.Language)
	}

	// 添加条件到查询
	if len(conditions) > 0 {
		conditionStr := " AND " + strings.Join(conditions, " AND ")
		baseQuery += conditionStr
		countQuery += conditionStr
	}

	// 先查询总数
	var total int64
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// 添加排序和分页
	baseQuery += " ORDER BY created_at DESC"
	offset := (condition.Page - 1) * condition.PageSize
	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", condition.PageSize, offset)

	// 查询数据
	var submissions []*Submission
	err = m.CachedConn.QueryRowsNoCacheCtx(ctx, &submissions, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return submissions, total, nil
}
