package problem

import (
	"context"
	"encoding/json"

	"code-judger/services/problem-api/internal/svc"
	"code-judger/services/problem-api/internal/types"
	"code-judger/services/problem-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProblemDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetProblemDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProblemDetailLogic {
	return &GetProblemDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetProblemDetailLogic) GetProblemDetail(req *types.GetProblemDetailReq) (resp *types.GetProblemDetailResp, err error) {
	// 验证题目ID
	if req.Id <= 0 {
		return &types.GetProblemDetailResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "无效的题目ID",
			},
		}, nil
	}

	// 从数据库查询题目详情（带缓存）
	problem, err := l.svcCtx.ProblemModel.FindOneWithCache(l.ctx, req.Id)
	if err != nil {
		logx.Errorf("Failed to get problem detail: id=%d, error=%v", req.Id, err)
		return &types.GetProblemDetailResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "题目不存在",
			},
		}, nil
	}

	// 检查题目是否被删除
	if problem.DeletedAt.Valid {
		return &types.GetProblemDetailResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "题目已被删除",
			},
		}, nil
	}

	// 检查题目是否公开（这里可以根据用户权限决定是否显示非公开题目）
	if !problem.IsPublic {
		// TODO: 检查用户权限，如果是题目创建者或管理员可以查看
		return &types.GetProblemDetailResp{
			BaseResp: types.BaseResp{
				Code:    403,
				Message: "题目未公开",
			},
		}, nil
	}

	// 转换为响应格式
	problemInfo := l.convertToProblemInfo(problem)

	// 记录查看日志
	logx.Infof("Problem detail viewed: id=%d, title=%s", req.Id, problem.Title)

	return &types.GetProblemDetailResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: problemInfo,
	}, nil
}

// 转换模型为响应结构
func (l *GetProblemDetailLogic) convertToProblemInfo(problem *models.Problem) types.ProblemInfo {
	info := types.ProblemInfo{
		Id:          problem.Id,
		Title:       problem.Title,
		Description: problem.Description,
		Difficulty:  problem.Difficulty,
		TimeLimit:   problem.TimeLimit,
		MemoryLimit: problem.MemoryLimit,
		CreatedAt:   problem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   problem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 处理可选字段
	if problem.InputFormat.Valid {
		info.InputFormat = problem.InputFormat.String
	}
	if problem.OutputFormat.Valid {
		info.OutputFormat = problem.OutputFormat.String
	}
	if problem.SampleInput.Valid {
		info.SampleInput = problem.SampleInput.String
	}
	if problem.SampleOutput.Valid {
		info.SampleOutput = problem.SampleOutput.String
	}

	// 解析Languages JSON
	if problem.Languages.Valid && problem.Languages.String != "" {
		var languages []string
		if err := json.Unmarshal([]byte(problem.Languages.String), &languages); err == nil {
			info.Languages = languages
		} else {
			logx.Errorf("Failed to parse languages JSON: %v", err)
			info.Languages = []string{}
		}
	} else {
		info.Languages = []string{}
	}

	// 解析Tags JSON
	if problem.Tags.Valid && problem.Tags.String != "" {
		var tags []string
		if err := json.Unmarshal([]byte(problem.Tags.String), &tags); err == nil {
			info.Tags = tags
		} else {
			logx.Errorf("Failed to parse tags JSON: %v", err)
			info.Tags = []string{}
		}
	} else {
		info.Tags = []string{}
	}

	// 设置作者信息（这里先模拟，实际应该查询用户服务）
	info.Author = types.UserInfo{
		UserId:   problem.CreatedBy,
		Username: "user" + string(rune(problem.CreatedBy)), // 临时模拟
		Name:     "用户" + string(rune(problem.CreatedBy)),   // 临时模拟
	}

	// 设置统计信息
	info.Statistics = types.ProblemStats{
		TotalSubmissions:    problem.SubmissionCount,
		AcceptedSubmissions: problem.AcceptedCount,
		AcceptanceRate:      problem.AcceptanceRate,
	}

	return info
}