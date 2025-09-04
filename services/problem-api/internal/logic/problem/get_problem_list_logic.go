package problem

import (
	"context"
	"encoding/json"
	"math"
	"strings"

	"github.com/dszqbsm/code-judger/services/problem-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/problem-api/internal/types"
	"github.com/dszqbsm/code-judger/services/problem-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProblemListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetProblemListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProblemListLogic {
	return &GetProblemListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetProblemListLogic) GetProblemList(req *types.GetProblemListReq) (resp *types.GetProblemListResp, err error) {
	// 验证和设置默认参数
	page := req.Page
	if page < 1 {
		page = 1
	}
	
	limit := req.Limit
	if limit < 1 || limit > l.svcCtx.Config.Business.MaxPageSize {
		limit = l.svcCtx.Config.Business.DefaultPageSize
	}

	// 构建过滤条件
	filters := &models.ProblemFilters{
		SortBy: req.SortBy,
		Order:  req.Order,
	}

	// 处理难度筛选
	if req.Difficulty != "" {
		filters.Difficulty = req.Difficulty
	}

	// 处理标签筛选
	if req.Tags != "" {
		tags := strings.Split(req.Tags, ",")
		var validTags []string
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				validTags = append(validTags, tag)
			}
		}
		if len(validTags) > 0 {
			filters.Tags = validTags
		}
	}

	// 处理关键词搜索
	if req.Keyword != "" {
		filters.Keyword = strings.TrimSpace(req.Keyword)
	}

	// 设置公开状态过滤（只显示公开的题目）
	isPublic := true
	filters.IsPublic = &isPublic

	// 查询题目列表
	problems, total, err := l.svcCtx.ProblemModel.FindByPage(l.ctx, page, limit, filters)
	if err != nil {
		logx.Errorf("Failed to get problem list: %v", err)
		return &types.GetProblemListResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "获取题目列表失败",
			},
		}, nil
	}

	// 转换为响应格式
	var problemList []types.ProblemListItem
	for _, problem := range problems {
		item := types.ProblemListItem{
			Id:             problem.Id,
			Title:          problem.Title,
			Difficulty:     problem.Difficulty,
			AcceptanceRate: problem.AcceptanceRate,
			CreatedAt:      problem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// 解析标签
		if problem.Tags.Valid && problem.Tags.String != "" {
			var tags []string
			if err := json.Unmarshal([]byte(problem.Tags.String), &tags); err == nil {
				item.Tags = tags
			}
		}

		problemList = append(problemList, item)
	}

	// 计算分页信息
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	pagination := types.Pagination{
		Page:  page,
		Limit: limit,
		Total: total,
		Pages: totalPages,
	}

	// 记录操作日志
	logx.Infof("Problem list retrieved: page=%d, limit=%d, total=%d, filters=%+v", 
		page, limit, total, filters)

	return &types.GetProblemListResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: types.GetProblemListData{
			Problems:   problemList,
			Pagination: pagination,
		},
	}, nil
}