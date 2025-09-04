package problem

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dszqbsm/code-judger/services/problem-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/problem-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/problem-api/internal/types"
	"github.com/dszqbsm/code-judger/services/problem-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateProblemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewCreateProblemLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *CreateProblemLogic {
	return &CreateProblemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *CreateProblemLogic) CreateProblem(req *types.CreateProblemReq) (resp *types.CreateProblemResp, err error) {
	// 1. 获取用户信息
	var user *middleware.UserInfo
	
	// 首先尝试从go-zero的JWT上下文获取用户信息
	user, err = middleware.GetUserFromContext(l.ctx)
	if err != nil {
		// 如果上下文中没有，尝试从HTTP请求头获取
		if l.r != nil {
			user, err = middleware.GetUserFromJWT(l.r, l.svcCtx.JWTManager)
			if err != nil {
				logx.Errorf("获取用户信息失败: %v", err)
				return &types.CreateProblemResp{
					BaseResp: types.BaseResp{
						Code:    401,
						Message: "认证失败：" + err.Error(),
					},
				}, nil
			}
		} else {
			logx.Errorf("无法获取用户信息: 上下文和请求头都为空")
			return &types.CreateProblemResp{
				BaseResp: types.BaseResp{
					Code:    401,
					Message: "认证失败：缺少用户信息",
				},
			}, nil
		}
	}

	// 2. 验证用户权限
	if err = middleware.ValidateCreateProblemPermission(user.Role); err != nil {
		logx.Errorf("用户 %s (ID: %d) 权限验证失败: %v", user.Username, user.UserID, err)
		return &types.CreateProblemResp{
			BaseResp: types.BaseResp{
				Code:    403,
				Message: err.Error(),
			},
		}, nil
	}

	// 3. 验证请求参数
	if err = l.validateRequest(req); err != nil {
		return &types.CreateProblemResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 4. 记录操作日志
	logx.Infof("用户 %s (ID: %d, Role: %s) 开始创建题目: %s", 
		user.Username, user.UserID, user.Role, req.Title)

	// 5. 创建题目对象
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
		CreatedBy:    user.UserID, // 使用实际的用户ID
		IsPublic:     req.IsPublic,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 6. 处理Languages和Tags JSON格式
	if len(req.Languages) > 0 {
		languagesJson, err := json.Marshal(req.Languages)
		if err != nil {
			logx.Errorf("序列化编程语言列表失败: %v", err)
			return &types.CreateProblemResp{
				BaseResp: types.BaseResp{
					Code:    400,
					Message: "编程语言格式错误",
				},
			}, nil
		}
		problem.Languages.String = string(languagesJson)
		problem.Languages.Valid = true
	}

	if len(req.Tags) > 0 {
		tagsJson, err := json.Marshal(req.Tags)
		if err != nil {
			logx.Errorf("序列化标签列表失败: %v", err)
			return &types.CreateProblemResp{
				BaseResp: types.BaseResp{
					Code:    400,
					Message: "标签格式错误",
				},
			}, nil
		}
		problem.Tags.String = string(tagsJson)
		problem.Tags.Valid = true
	}

	// 7. 插入数据库
	result, err := l.svcCtx.ProblemModel.Insert(l.ctx, problem)
	if err != nil {
		logx.Errorf("用户 %s (ID: %d) 创建题目失败: %v", user.Username, user.UserID, err)
		return &types.CreateProblemResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "创建题目失败，请稍后重试",
			},
		}, nil
	}

	// 8. 获取创建的题目ID
	problemId, err := result.LastInsertId()
	if err != nil {
		logx.Errorf("获取题目ID失败: %v", err)
		return &types.CreateProblemResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "获取题目ID失败",
			},
		}, nil
	}

	// 9. 记录成功操作日志
	logx.Infof("题目创建成功: ID=%d, 标题=%s, 创建者=%s (ID: %d), 难度=%s, 公开=%t", 
		problemId, req.Title, user.Username, user.UserID, req.Difficulty, req.IsPublic)

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