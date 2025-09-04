package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type (
	// TestCaseModel 测试用例模型接口
	TestCaseModel interface {
		Insert(ctx context.Context, data *TestCase) (sql.Result, error)
		FindByProblemId(ctx context.Context, problemId int64) ([]*TestCase, error)
		FindOne(ctx context.Context, id int64) (*TestCase, error)
		Update(ctx context.Context, data *TestCase) error
		Delete(ctx context.Context, id int64) error
		DeleteByProblemId(ctx context.Context, problemId int64) error
		BatchInsert(ctx context.Context, testCases []*TestCase) error
		FindSamplesByProblemId(ctx context.Context, problemId int64) ([]*TestCase, error)
		FindNonSamplesByProblemId(ctx context.Context, problemId int64) ([]*TestCase, error)
		CountByProblemId(ctx context.Context, problemId int64) (int64, error)
	}

	defaultTestCaseModel struct {
		conn  *sql.DB
		table string
	}

	// TestCase 测试用例结构体
	TestCase struct {
		Id             int64     `db:"id" json:"id"`
		ProblemId      int64     `db:"problem_id" json:"problem_id"`
		InputData      string    `db:"input_data" json:"input_data"`
		ExpectedOutput string    `db:"expected_output" json:"expected_output"`
		IsSample       bool      `db:"is_sample" json:"is_sample"`
		Score          int       `db:"score" json:"score"`
		SortOrder      int       `db:"sort_order" json:"sort_order"`
		CreatedAt      time.Time `db:"created_at" json:"created_at"`
	}

	// TestCaseUploadRequest 测试用例上传请求
	TestCaseUploadRequest struct {
		ProblemId  int64             `json:"problem_id"`
		TestCases  []TestCaseRequest `json:"test_cases"`
		ReplaceAll bool              `json:"replace_all"` // 是否替换所有现有测试用例
	}

	// TestCaseRequest 单个测试用例请求
	TestCaseRequest struct {
		InputData      string `json:"input_data"`
		ExpectedOutput string `json:"expected_output"`
		IsSample       bool   `json:"is_sample"`
		Score          int    `json:"score"`
		SortOrder      int    `json:"sort_order"`
	}

	// TestCaseResponse 测试用例响应
	TestCaseResponse struct {
		Id             int64  `json:"id"`
		ProblemId      int64  `json:"problem_id"`
		InputData      string `json:"input_data"`
		ExpectedOutput string `json:"expected_output"`
		IsSample       bool   `json:"is_sample"`
		Score          int    `json:"score"`
		SortOrder      int    `json:"sort_order"`
		CreatedAt      string `json:"created_at"`
	}
)

// NewTestCaseModel 创建测试用例模型实例
func NewTestCaseModel(conn *sql.DB) TestCaseModel {
	return &defaultTestCaseModel{
		conn:  conn,
		table: "`test_cases`",
	}
}

// Insert 插入测试用例
func (m *defaultTestCaseModel) Insert(ctx context.Context, data *TestCase) (sql.Result, error) {
	query := fmt.Sprintf("INSERT INTO %s (problem_id, input_data, expected_output, is_sample, score, sort_order) VALUES (?, ?, ?, ?, ?, ?)", m.table)
	return m.conn.ExecContext(ctx, query, data.ProblemId, data.InputData, data.ExpectedOutput, data.IsSample, data.Score, data.SortOrder)
}

// FindByProblemId 根据题目ID查找所有测试用例
func (m *defaultTestCaseModel) FindByProblemId(ctx context.Context, problemId int64) ([]*TestCase, error) {
	query := fmt.Sprintf("SELECT id, problem_id, input_data, expected_output, is_sample, score, sort_order, created_at FROM %s WHERE problem_id = ? ORDER BY sort_order ASC, id ASC", m.table)
	
	rows, err := m.conn.QueryContext(ctx, query, problemId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []*TestCase
	for rows.Next() {
		var testCase TestCase
		err := rows.Scan(&testCase.Id, &testCase.ProblemId, &testCase.InputData, &testCase.ExpectedOutput, &testCase.IsSample, &testCase.Score, &testCase.SortOrder, &testCase.CreatedAt)
		if err != nil {
			return nil, err
		}
		testCases = append(testCases, &testCase)
	}

	return testCases, nil
}

// FindOne 根据ID查找测试用例
func (m *defaultTestCaseModel) FindOne(ctx context.Context, id int64) (*TestCase, error) {
	query := fmt.Sprintf("SELECT id, problem_id, input_data, expected_output, is_sample, score, sort_order, created_at FROM %s WHERE id = ? LIMIT 1", m.table)
	
	var testCase TestCase
	err := m.conn.QueryRowContext(ctx, query, id).Scan(&testCase.Id, &testCase.ProblemId, &testCase.InputData, &testCase.ExpectedOutput, &testCase.IsSample, &testCase.Score, &testCase.SortOrder, &testCase.CreatedAt)
	if err != nil {
		return nil, err
	}
	
	return &testCase, nil
}

// Update 更新测试用例
func (m *defaultTestCaseModel) Update(ctx context.Context, data *TestCase) error {
	query := fmt.Sprintf("UPDATE %s SET input_data = ?, expected_output = ?, is_sample = ?, score = ?, sort_order = ? WHERE id = ?", m.table)
	_, err := m.conn.ExecContext(ctx, query, data.InputData, data.ExpectedOutput, data.IsSample, data.Score, data.SortOrder, data.Id)
	return err
}

// Delete 删除测试用例
func (m *defaultTestCaseModel) Delete(ctx context.Context, id int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", m.table)
	_, err := m.conn.ExecContext(ctx, query, id)
	return err
}

// DeleteByProblemId 删除题目的所有测试用例
func (m *defaultTestCaseModel) DeleteByProblemId(ctx context.Context, problemId int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE problem_id = ?", m.table)
	_, err := m.conn.ExecContext(ctx, query, problemId)
	return err
}

// BatchInsert 批量插入测试用例
func (m *defaultTestCaseModel) BatchInsert(ctx context.Context, testCases []*TestCase) error {
	if len(testCases) == 0 {
		return nil
	}

	// 构建批量插入SQL
	query := fmt.Sprintf("INSERT INTO %s (problem_id, input_data, expected_output, is_sample, score, sort_order) VALUES ", m.table)
	
	var values []string
	var args []interface{}
	
	for _, testCase := range testCases {
		values = append(values, "(?, ?, ?, ?, ?, ?)")
		args = append(args, testCase.ProblemId, testCase.InputData, testCase.ExpectedOutput, testCase.IsSample, testCase.Score, testCase.SortOrder)
	}
	
	query += values[0]
	for i := 1; i < len(values); i++ {
		query += ", " + values[i]
	}
	
	_, err := m.conn.ExecContext(ctx, query, args...)
	return err
}

// FindSamplesByProblemId 查找示例测试用例
func (m *defaultTestCaseModel) FindSamplesByProblemId(ctx context.Context, problemId int64) ([]*TestCase, error) {
	query := fmt.Sprintf("SELECT id, problem_id, input_data, expected_output, is_sample, score, sort_order, created_at FROM %s WHERE problem_id = ? AND is_sample = true ORDER BY sort_order ASC, id ASC", m.table)
	
	rows, err := m.conn.QueryContext(ctx, query, problemId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []*TestCase
	for rows.Next() {
		var testCase TestCase
		err := rows.Scan(&testCase.Id, &testCase.ProblemId, &testCase.InputData, &testCase.ExpectedOutput, &testCase.IsSample, &testCase.Score, &testCase.SortOrder, &testCase.CreatedAt)
		if err != nil {
			return nil, err
		}
		testCases = append(testCases, &testCase)
	}

	return testCases, nil
}

// FindNonSamplesByProblemId 查找非示例测试用例
func (m *defaultTestCaseModel) FindNonSamplesByProblemId(ctx context.Context, problemId int64) ([]*TestCase, error) {
	query := fmt.Sprintf("SELECT id, problem_id, input_data, expected_output, is_sample, score, sort_order, created_at FROM %s WHERE problem_id = ? AND is_sample = false ORDER BY sort_order ASC, id ASC", m.table)
	
	rows, err := m.conn.QueryContext(ctx, query, problemId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []*TestCase
	for rows.Next() {
		var testCase TestCase
		err := rows.Scan(&testCase.Id, &testCase.ProblemId, &testCase.InputData, &testCase.ExpectedOutput, &testCase.IsSample, &testCase.Score, &testCase.SortOrder, &testCase.CreatedAt)
		if err != nil {
			return nil, err
		}
		testCases = append(testCases, &testCase)
	}

	return testCases, nil
}

// CountByProblemId 统计题目的测试用例数量
func (m *defaultTestCaseModel) CountByProblemId(ctx context.Context, problemId int64) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE problem_id = ?", m.table)
	
	var count int64
	err := m.conn.QueryRowContext(ctx, query, problemId).Scan(&count)
	return count, err
}
