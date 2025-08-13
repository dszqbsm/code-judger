package problem

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"code-judger/services/problem-api/internal/svc"
	"code-judger/services/problem-api/internal/types"
	"code-judger/services/problem-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateProblemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateProblemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateProblemLogic {
	return &CreateProblemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateProblemLogic) CreateProblem(req *types.CreateProblemReq) (resp *types.CreateProblemResp, err error) {
	// 验证用户权限 - 这里先模拟，实际应该从JWT中获取用户信息
	createdBy := int64(1) // 临时硬编码，实际应该从上下文获取

	// 验证请求参数
	if err = l.validateRequest(req); err != nil {
		return &types.CreateProblemResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 创建题目对象
	problem := &models.Problem{
		Title:        req.Title,
		Description:  req.Description,
		InputFormat:  l.nullString(req.InputFormat),
		OutputFormat: l.nullString(req.OutputFormat),
		SampleInput:  l.nullString(req.SampleInput),
		SampleOutput: l.nullString(req.SampleOutput),
		Difficulty:   req.Difficulty,
		TimeLimit:    req.TimeLimit,
		MemoryLimit:  req.MemoryLimit,
		CreatedBy:    createdBy,
		IsPublic:     req.IsPublic,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 处理Languages和Tags JSON格式
	if len(req.Languages) > 0 {
		languagesJson, _ := json.Marshal(req.Languages)
		problem.Languages.String = string(languagesJson)
		problem.Languages.Valid = true
	}

	if len(req.Tags) > 0 {
		tagsJson, _ := json.Marshal(req.Tags)
		problem.Tags.String = string(tagsJson)
		problem.Tags.Valid = true
	}

	// 插入数据库
	result, err := l.svcCtx.ProblemModel.Insert(l.ctx, problem)
	if err != nil {
		logx.Errorf("Failed to create problem: %v", err)
		return &types.CreateProblemResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "创建题目失败",
			},
		}, nil
	}

	// 获取创建的题目ID
	problemId, err := result.LastInsertId()
	if err != nil {
		logx.Errorf("Failed to get last insert id: %v", err)
		return &types.CreateProblemResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "获取题目ID失败",
			},
		}, nil
	}

	// 记录操作日志
	logx.Infof("Problem created successfully: id=%d, title=%s, created_by=%d", 
		problemId, req.Title, createdBy)

	return &types.CreateProblemResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "题目创建成功",
		},
		Data: types.CreateProblemData{
			ProblemId: problemId,
			Title:     req.Title,
			Status:    "draft",
			CreatedAt: time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}

// 验证请求参数
func (l *CreateProblemLogic) validateRequest(req *types.CreateProblemReq) error {
	// 验证标题长度
	if len(req.Title) == 0 || len(req.Title) > 200 {
		return fmt.Errorf("题目标题长度必须在1-200字符之间")
	}

	// 验证描述长度
	if len(req.Description) < 10 {
		return fmt.Errorf("题目描述至少需要10个字符")
	}

	// 验证难度级别
	validDifficulties := map[string]bool{"easy": true, "medium": true, "hard": true}
	if !validDifficulties[req.Difficulty] {
		return fmt.Errorf("无效的难度级别: %s", req.Difficulty)
	}

	// 验证时间限制
	if req.TimeLimit < 100 || req.TimeLimit > 10000 {
		return fmt.Errorf("时间限制必须在100-10000毫秒之间")
	}

	// 验证内存限制
	if req.MemoryLimit < 16 || req.MemoryLimit > 512 {
		return fmt.Errorf("内存限制必须在16-512MB之间")
	}

	// 验证编程语言
	if len(req.Languages) == 0 {
		return fmt.Errorf("至少需要指定一种编程语言")
	}

	// 验证标签数量
	if len(req.Tags) > 10 {
		return fmt.Errorf("标签数量不能超过10个")
	}

	return nil
}

// 辅助函数：字符串转NullString
func (l *CreateProblemLogic) nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}