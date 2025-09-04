package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/dszqbsm/code-judger/services/submission-api/models"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// SubmissionDao 提交数据访问接口
type SubmissionDao interface {
	CreateSubmission(ctx context.Context, submission *models.Submission) (int64, error)
	GetSubmissionUserID(ctx context.Context, submissionID int64) (int64, error)
	UpdateSubmissionStatus(ctx context.Context, submissionID int64, status string) error
	UpdateSubmissionResult(ctx context.Context, submissionID int64, resultData map[string]interface{}) error
	UpdateCompileInfo(ctx context.Context, submissionID int64, compileData map[string]interface{}) error
	GetSubmissionByID(ctx context.Context, submissionID int64) (*models.Submission, error)
	GetSubmissionsByUserID(ctx context.Context, userID int64, limit, offset int) ([]*models.Submission, error)
}

// SubmissionDaoImpl 提交数据访问实现
type SubmissionDaoImpl struct {
	conn         sqlx.SqlConn
	submissionModel models.SubmissionModel
}

func NewSubmissionDao(conn sqlx.SqlConn, submissionModel models.SubmissionModel) SubmissionDao {
	return &SubmissionDaoImpl{
		conn:            conn,
		submissionModel: submissionModel,
	}
}

// CreateSubmission 创建提交记录
func (d *SubmissionDaoImpl) CreateSubmission(ctx context.Context, submission *models.Submission) (int64, error) {
	result, err := d.submissionModel.Insert(ctx, submission)
	if err != nil {
		logx.WithContext(ctx).Errorf("创建提交记录失败: %v", err)
		return 0, fmt.Errorf("创建提交记录失败: %w", err)
	}

	submissionID, err := result.LastInsertId()
	if err != nil {
		logx.WithContext(ctx).Errorf("获取插入ID失败: %v", err)
		return 0, fmt.Errorf("获取插入ID失败: %w", err)
	}

	logx.WithContext(ctx).Infof("提交记录创建成功: ID=%d", submissionID)
	return submissionID, nil
}

// GetSubmissionUserID 获取提交的用户ID
func (d *SubmissionDaoImpl) GetSubmissionUserID(ctx context.Context, submissionID int64) (int64, error) {
	submission, err := d.submissionModel.FindOne(ctx, submissionID)
	if err != nil {
		if err == models.ErrNotFound {
			return 0, fmt.Errorf("提交记录不存在: SubmissionID=%d", submissionID)
		}
		logx.WithContext(ctx).Errorf("查询提交记录失败: %v", err)
		return 0, fmt.Errorf("查询提交记录失败: %w", err)
	}

	return submission.UserID, nil
}

// UpdateSubmissionStatus 更新提交状态
func (d *SubmissionDaoImpl) UpdateSubmissionStatus(ctx context.Context, submissionID int64, status string) error {
	query := `UPDATE submissions SET status = ?, updated_at = NOW() WHERE id = ?`
	
	_, err := d.conn.ExecCtx(ctx, query, status, submissionID)
	if err != nil {
		logx.WithContext(ctx).Errorf("更新提交状态失败: SubmissionID=%d, Status=%s, Error=%v", 
			submissionID, status, err)
		return fmt.Errorf("更新提交状态失败: %w", err)
	}

	logx.WithContext(ctx).Infof("提交状态更新成功: SubmissionID=%d, Status=%s", submissionID, status)
	return nil
}

// UpdateSubmissionResult 更新提交结果
func (d *SubmissionDaoImpl) UpdateSubmissionResult(ctx context.Context, submissionID int64, resultData map[string]interface{}) error {
	// 序列化结果数据为JSON
	resultJSON, err := json.Marshal(resultData)
	if err != nil {
		return fmt.Errorf("序列化结果数据失败: %w", err)
	}

	// 从结果数据中提取主要字段
	var score sql.NullInt32
	var timeUsed sql.NullInt32
	var memoryUsed sql.NullInt32

	if val, ok := resultData["score"].(int); ok {
		score = sql.NullInt32{Int32: int32(val), Valid: true}
	}
	if val, ok := resultData["time_used"].(int); ok {
		timeUsed = sql.NullInt32{Int32: int32(val), Valid: true}
	}
	if val, ok := resultData["memory_used"].(int); ok {
		memoryUsed = sql.NullInt32{Int32: int32(val), Valid: true}
	}

	query := `UPDATE submissions SET 
		score = ?, 
		time_used = ?, 
		memory_used = ?, 
		judge_result = ?, 
		updated_at = NOW() 
		WHERE id = ?`

	_, err = d.conn.ExecCtx(ctx, query, score, timeUsed, memoryUsed, string(resultJSON), submissionID)
	if err != nil {
		logx.WithContext(ctx).Errorf("更新提交结果失败: SubmissionID=%d, Error=%v", submissionID, err)
		return fmt.Errorf("更新提交结果失败: %w", err)
	}

	logx.WithContext(ctx).Infof("提交结果更新成功: SubmissionID=%d, Score=%v, TimeUsed=%v, MemoryUsed=%v", 
		submissionID, score, timeUsed, memoryUsed)
	return nil
}

// UpdateCompileInfo 更新编译信息
func (d *SubmissionDaoImpl) UpdateCompileInfo(ctx context.Context, submissionID int64, compileData map[string]interface{}) error {
	// 序列化编译信息为JSON
	compileJSON, err := json.Marshal(compileData)
	if err != nil {
		return fmt.Errorf("序列化编译信息失败: %w", err)
	}

	query := `UPDATE submissions SET compile_info = ?, updated_at = NOW() WHERE id = ?`
	
	_, err = d.conn.ExecCtx(ctx, query, string(compileJSON), submissionID)
	if err != nil {
		logx.WithContext(ctx).Errorf("更新编译信息失败: SubmissionID=%d, Error=%v", submissionID, err)
		return fmt.Errorf("更新编译信息失败: %w", err)
	}

	logx.WithContext(ctx).Infof("编译信息更新成功: SubmissionID=%d", submissionID)
	return nil
}

// GetSubmissionByID 根据ID获取提交记录
func (d *SubmissionDaoImpl) GetSubmissionByID(ctx context.Context, submissionID int64) (*models.Submission, error) {
	submission, err := d.submissionModel.FindOne(ctx, submissionID)
	if err != nil {
		if err == models.ErrNotFound {
			return nil, fmt.Errorf("提交记录不存在: SubmissionID=%d", submissionID)
		}
		logx.WithContext(ctx).Errorf("查询提交记录失败: SubmissionID=%d, Error=%v", submissionID, err)
		return nil, fmt.Errorf("查询提交记录失败: %w", err)
	}

	return submission, nil
}

// GetSubmissionsByUserID 根据用户ID获取提交记录列表
func (d *SubmissionDaoImpl) GetSubmissionsByUserID(ctx context.Context, userID int64, limit, offset int) ([]*models.Submission, error) {
	query := `SELECT id, user_id, problem_id, contest_id, language, code, code_length, 
		status, score, time_used, memory_used, compile_info, judge_result, ip_address, 
		created_at, updated_at 
		FROM submissions 
		WHERE user_id = ? 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?`

	var submissions []*models.Submission
	err := d.conn.QueryRowsCtx(ctx, &submissions, query, userID, limit, offset)
	if err != nil {
		logx.WithContext(ctx).Errorf("查询用户提交记录失败: UserID=%d, Error=%v", userID, err)
		return nil, fmt.Errorf("查询用户提交记录失败: %w", err)
	}

	logx.WithContext(ctx).Infof("查询用户提交记录成功: UserID=%d, Count=%d", userID, len(submissions))
	return submissions, nil
}