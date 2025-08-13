package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type (
	// ProblemModel is an interface to be customized, add more methods here,
	// and implement the added methods in customProblemModel.
	ProblemModel interface {
		Insert(ctx context.Context, data *Problem) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*Problem, error)
		FindOneWithCache(ctx context.Context, id int64) (*Problem, error)
		Update(ctx context.Context, data *Problem) error
		UpdateWithCache(ctx context.Context, problem *Problem) error
		Delete(ctx context.Context, id int64) error
		SoftDelete(ctx context.Context, id int64) error
		FindByPage(ctx context.Context, page, limit int, filters *ProblemFilters) ([]*Problem, int64, error)
		Search(ctx context.Context, keyword string, page, limit int) ([]*Problem, int64, error)
		FindByTags(ctx context.Context, tags []string, page, limit int) ([]*Problem, int64, error)
		UpdateStatistics(ctx context.Context, problemId int64, totalSubmissions, acceptedSubmissions int) error
	}

	defaultProblemModel struct {
		conn  *sql.DB
		table string
	}

	// 题目结构体
	Problem struct {
		Id              int64          `db:"id" json:"id"`
		Title           string         `db:"title" json:"title"`
		Description     string         `db:"description" json:"description"`
		InputFormat     sql.NullString `db:"input_format" json:"input_format"`
		OutputFormat    sql.NullString `db:"output_format" json:"output_format"`
		SampleInput     sql.NullString `db:"sample_input" json:"sample_input"`
		SampleOutput    sql.NullString `db:"sample_output" json:"sample_output"`
		Difficulty      string         `db:"difficulty" json:"difficulty"`
		TimeLimit       int            `db:"time_limit" json:"time_limit"`
		MemoryLimit     int            `db:"memory_limit" json:"memory_limit"`
		Languages       sql.NullString `db:"languages" json:"languages"`       // JSON格式存储
		Tags            sql.NullString `db:"tags" json:"tags"`                  // JSON格式存储
		CreatedBy       int64          `db:"created_by" json:"created_by"`
		IsPublic        bool           `db:"is_public" json:"is_public"`
		SubmissionCount int            `db:"submission_count" json:"submission_count"`
		AcceptedCount   int            `db:"accepted_count" json:"accepted_count"`
		AcceptanceRate  float64        `db:"acceptance_rate" json:"acceptance_rate"`
		CreatedAt       time.Time      `db:"created_at" json:"created_at"`
		UpdatedAt       time.Time      `db:"updated_at" json:"updated_at"`
		DeletedAt       sql.NullTime   `db:"deleted_at" json:"deleted_at"`
	}

	// 查询过滤条件
	ProblemFilters struct {
		Difficulty string
		Tags       []string
		Keyword    string
		CreatedBy  int64
		IsPublic   *bool
		SortBy     string // created_at, title, difficulty, acceptance_rate
		Order      string // asc, desc
	}
)

// NewProblemModel returns a model for the database table.
func NewProblemModel(conn *sql.DB) ProblemModel {
	return &defaultProblemModel{
		conn:  conn,
		table: "`problems`",
	}
}

// 插入题目
func (m *defaultProblemModel) Insert(ctx context.Context, data *Problem) (sql.Result, error) {
	query := fmt.Sprintf("INSERT INTO %s (title, description, input_format, output_format, sample_input, sample_output, difficulty, time_limit, memory_limit, languages, tags, created_by, is_public) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table)

	var languagesJson, tagsJson sql.NullString
	if data.Languages.Valid {
		languagesJson = data.Languages
	}
	if data.Tags.Valid {
		tagsJson = data.Tags
	}

	return m.conn.ExecContext(ctx, query,
		data.Title, data.Description, data.InputFormat, data.OutputFormat,
		data.SampleInput, data.SampleOutput, data.Difficulty,
		data.TimeLimit, data.MemoryLimit, languagesJson, tagsJson,
		data.CreatedBy, data.IsPublic)
}

// 根据ID查找题目
func (m *defaultProblemModel) FindOne(ctx context.Context, id int64) (*Problem, error) {
	query := fmt.Sprintf("SELECT id, title, description, input_format, output_format, sample_input, sample_output, difficulty, time_limit, memory_limit, languages, tags, created_by, is_public, submission_count, accepted_count, acceptance_rate, created_at, updated_at, deleted_at FROM %s WHERE id = ? AND deleted_at IS NULL LIMIT 1", m.table)

	var resp Problem
	err := m.conn.QueryRowContext(ctx, query, id).Scan(
		&resp.Id, &resp.Title, &resp.Description, &resp.InputFormat, &resp.OutputFormat,
		&resp.SampleInput, &resp.SampleOutput, &resp.Difficulty, &resp.TimeLimit,
		&resp.MemoryLimit, &resp.Languages, &resp.Tags, &resp.CreatedBy, &resp.IsPublic,
		&resp.SubmissionCount, &resp.AcceptedCount, &resp.AcceptanceRate,
		&resp.CreatedAt, &resp.UpdatedAt, &resp.DeletedAt,
	)

	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// 带缓存的查找（暂时等同于普通查找）
func (m *defaultProblemModel) FindOneWithCache(ctx context.Context, id int64) (*Problem, error) {
	return m.FindOne(ctx, id)
}

// 更新题目
func (m *defaultProblemModel) Update(ctx context.Context, data *Problem) error {
	query := fmt.Sprintf("UPDATE %s SET title = ?, description = ?, input_format = ?, output_format = ?, sample_input = ?, sample_output = ?, difficulty = ?, time_limit = ?, memory_limit = ?, languages = ?, tags = ?, is_public = ?, updated_at = NOW() WHERE id = ?", m.table)

	_, err := m.conn.ExecContext(ctx, query,
		data.Title, data.Description, data.InputFormat, data.OutputFormat,
		data.SampleInput, data.SampleOutput, data.Difficulty,
		data.TimeLimit, data.MemoryLimit, data.Languages, data.Tags,
		data.IsPublic, data.Id)

	return err
}

// 带缓存清理的更新（暂时等同于普通更新）
func (m *defaultProblemModel) UpdateWithCache(ctx context.Context, problem *Problem) error {
	return m.Update(ctx, problem)
}

// 删除题目
func (m *defaultProblemModel) Delete(ctx context.Context, id int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", m.table)
	_, err := m.conn.ExecContext(ctx, query, id)
	return err
}

// 软删除
func (m *defaultProblemModel) SoftDelete(ctx context.Context, id int64) error {
	query := fmt.Sprintf("UPDATE %s SET deleted_at = NOW() WHERE id = ?", m.table)
	_, err := m.conn.ExecContext(ctx, query, id)
	return err
}

// 分页查询
func (m *defaultProblemModel) FindByPage(ctx context.Context, page, limit int, filters *ProblemFilters) ([]*Problem, int64, error) {
	offset := (page - 1) * limit

	// 构建WHERE条件
	var whereConditions []string
	var args []interface{}

	whereConditions = append(whereConditions, "deleted_at IS NULL")

	if filters != nil {
		if filters.Difficulty != "" {
			whereConditions = append(whereConditions, "difficulty = ?")
			args = append(args, filters.Difficulty)
		}

		if len(filters.Tags) > 0 {
			for _, tag := range filters.Tags {
				whereConditions = append(whereConditions, "JSON_CONTAINS(tags, ?)")
				args = append(args, fmt.Sprintf(`"%s"`, tag))
			}
		}

		if filters.Keyword != "" {
			whereConditions = append(whereConditions, "(title LIKE ? OR description LIKE ?)")
			keyword := "%" + filters.Keyword + "%"
			args = append(args, keyword, keyword)
		}

		if filters.CreatedBy > 0 {
			whereConditions = append(whereConditions, "created_by = ?")
			args = append(args, filters.CreatedBy)
		}

		if filters.IsPublic != nil {
			whereConditions = append(whereConditions, "is_public = ?")
			args = append(args, *filters.IsPublic)
		}
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 构建ORDER BY
	orderBy := "ORDER BY created_at DESC"
	if filters != nil && filters.SortBy != "" {
		orderBy = fmt.Sprintf("ORDER BY %s %s", filters.SortBy, filters.Order)
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", m.table, whereClause)
	var total int64
	err := m.conn.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 查询数据
	dataQuery := fmt.Sprintf("SELECT id, title, description, input_format, output_format, sample_input, sample_output, difficulty, time_limit, memory_limit, languages, tags, created_by, is_public, submission_count, accepted_count, acceptance_rate, created_at, updated_at, deleted_at FROM %s %s %s LIMIT ? OFFSET ?", m.table, whereClause, orderBy)
	args = append(args, limit, offset)

	rows, err := m.conn.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var problems []*Problem
	for rows.Next() {
		var problem Problem
		err := rows.Scan(
			&problem.Id, &problem.Title, &problem.Description, &problem.InputFormat, &problem.OutputFormat,
			&problem.SampleInput, &problem.SampleOutput, &problem.Difficulty, &problem.TimeLimit,
			&problem.MemoryLimit, &problem.Languages, &problem.Tags, &problem.CreatedBy, &problem.IsPublic,
			&problem.SubmissionCount, &problem.AcceptedCount, &problem.AcceptanceRate,
			&problem.CreatedAt, &problem.UpdatedAt, &problem.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		problems = append(problems, &problem)
	}

	return problems, total, nil
}

// 搜索题目
func (m *defaultProblemModel) Search(ctx context.Context, keyword string, page, limit int) ([]*Problem, int64, error) {
	filters := &ProblemFilters{
		Keyword: keyword,
		SortBy:  "created_at",
		Order:   "desc",
	}
	return m.FindByPage(ctx, page, limit, filters)
}

// 根据标签查找题目
func (m *defaultProblemModel) FindByTags(ctx context.Context, tags []string, page, limit int) ([]*Problem, int64, error) {
	filters := &ProblemFilters{
		Tags:   tags,
		SortBy: "created_at",
		Order:  "desc",
	}
	return m.FindByPage(ctx, page, limit, filters)
}

// 更新统计信息
func (m *defaultProblemModel) UpdateStatistics(ctx context.Context, problemId int64, totalSubmissions, acceptedSubmissions int) error {
	acceptanceRate := 0.0
	if totalSubmissions > 0 {
		acceptanceRate = float64(acceptedSubmissions) / float64(totalSubmissions) * 100
	}

	query := fmt.Sprintf("UPDATE %s SET submission_count = ?, accepted_count = ?, acceptance_rate = ? WHERE id = ?", m.table)
	_, err := m.conn.ExecContext(ctx, query, totalSubmissions, acceptedSubmissions, acceptanceRate, problemId)
	return err
}