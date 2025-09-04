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

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProblemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewUpdateProblemLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UpdateProblemLogic {
	return &UpdateProblemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *UpdateProblemLogic) UpdateProblem(req *types.UpdateProblemReq) (resp *types.UpdateProblemResp, err error) {
	// 1. 验证题目ID
	if req.Id <= 0 {
		return &types.UpdateProblemResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "无效的题目ID",
			},
		}, nil
	}

	// 2. 获取用户信息
	var user *middleware.UserInfo

	// 首先尝试从go-zero的JWT上下文获取用户信息
	user, err = middleware.GetUserFromContext(l.ctx)
	if err != nil {
		// 如果上下文中没有，尝试从HTTP请求头获取
		if l.r != nil {
			user, err = middleware.GetUserFromJWT(l.r, l.svcCtx.JWTManager)
			if err != nil {
				logx.Errorf("获取用户信息失败: %v", err)
				return &types.UpdateProblemResp{
					BaseResp: types.BaseResp{
						Code:    401,
						Message: "认证失败：" + err.Error(),
					},
				}, nil
			}
		} else {
			logx.Errorf("无法获取用户信息: 上下文和请求头都为空")
			return &types.UpdateProblemResp{
				BaseResp: types.BaseResp{
					Code:    401,
					Message: "认证失败：缺少用户信息",
				},
			}, nil
		}
	}

	// 3. 查询现有题目
	existingProblem, err := l.svcCtx.ProblemModel.FindOne(l.ctx, req.Id)
	if err != nil {
		logx.Errorf("查找题目失败: id=%d, error=%v", req.Id, err)
		return &types.UpdateProblemResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "题目不存在",
			},
		}, nil
	}

	// 4. 检查题目是否被删除
	if existingProblem.DeletedAt.Valid {
		logx.Errorf("用户 %s (ID: %d) 尝试修改已删除的题目: ID=%d", user.Username, user.UserID, req.Id)
		return &types.UpdateProblemResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "题目已被删除",
			},
		}, nil
	}

	// 5. 验证用户权限（使用中间件的权限验证函数）
	if err = middleware.ValidateUpdateProblemPermission(user.Role, user.UserID, existingProblem.CreatedBy); err != nil {
		logx.Errorf("用户 %s (ID: %d) 权限验证失败，无法修改题目 %d: %v",
			user.Username, user.UserID, req.Id, err)
		return &types.UpdateProblemResp{
			BaseResp: types.BaseResp{
				Code:    403,
				Message: err.Error(),
			},
		}, nil
	}

	// 6. 记录操作开始日志
	logx.Infof("用户 %s (ID: %d, Role: %s) 开始修改题目: ID=%d, 标题=%s",
		user.Username, user.UserID, user.Role, req.Id, existingProblem.Title)

	// 7. 验证更新参数
	if err := l.validateUpdateRequest(req); err != nil {
		return &types.UpdateProblemResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 8. 构建更新数据
	updatedProblem := *existingProblem
	hasChanges := false

	// 更新字段（只更新非空字段）
	if req.Title != "" {
		updatedProblem.Title = req.Title
		hasChanges = true
	}
	if req.Description != "" {
		updatedProblem.Description = req.Description
		hasChanges = true
	}
	if req.InputFormat != "" {
		updatedProblem.InputFormat = sql.NullString{String: req.InputFormat, Valid: true}
		hasChanges = true
	}
	if req.OutputFormat != "" {
		updatedProblem.OutputFormat = sql.NullString{String: req.OutputFormat, Valid: true}
		hasChanges = true
	}
	if req.SampleInput != "" {
		updatedProblem.SampleInput = sql.NullString{String: req.SampleInput, Valid: true}
		hasChanges = true
	}
	if req.SampleOutput != "" {
		updatedProblem.SampleOutput = sql.NullString{String: req.SampleOutput, Valid: true}
		hasChanges = true
	}
	if req.Difficulty != "" {
		updatedProblem.Difficulty = req.Difficulty
		hasChanges = true
	}
	if req.TimeLimit > 0 {
		updatedProblem.TimeLimit = req.TimeLimit
		hasChanges = true
	}
	if req.MemoryLimit > 0 {
		updatedProblem.MemoryLimit = req.MemoryLimit
		hasChanges = true
	}

	// 更新Languages
	if len(req.Languages) > 0 {
		languagesJson, err := json.Marshal(req.Languages)
		if err != nil {
			logx.Errorf("序列化编程语言列表失败: %v", err)
			return &types.UpdateProblemResp{
				BaseResp: types.BaseResp{
					Code:    400,
					Message: "编程语言格式错误",
				},
			}, nil
		}
		updatedProblem.Languages = sql.NullString{String: string(languagesJson), Valid: true}
		hasChanges = true
	}

	// 更新Tags
	if len(req.Tags) > 0 {
		tagsJson, err := json.Marshal(req.Tags)
		if err != nil {
			logx.Errorf("序列化标签列表失败: %v", err)
			return &types.UpdateProblemResp{
				BaseResp: types.BaseResp{
					Code:    400,
					Message: "标签格式错误",
				},
			}, nil
		}
		updatedProblem.Tags = sql.NullString{String: string(tagsJson), Valid: true}
		hasChanges = true
	}

	// 更新公开状态
	updatedProblem.IsPublic = req.IsPublic
	hasChanges = true

	// 9. 如果没有变更，直接返回
	if !hasChanges {
		logx.Infof("用户 %s (ID: %d) 尝试更新题目 %d，但没有实际变更",
			user.Username, user.UserID, req.Id)
		return &types.UpdateProblemResp{
			BaseResp: types.BaseResp{
				Code:    200,
				Message: "没有需要更新的内容",
			},
			Data: types.UpdateProblemData{
				ProblemId: req.Id,
				UpdatedAt: existingProblem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Message:   "题目信息无变化",
			},
		}, nil
	}

	// 10. 执行更新（带缓存清理）
	updatedProblem.UpdatedAt = time.Now()
	err = l.svcCtx.ProblemModel.UpdateWithCache(l.ctx, &updatedProblem)
	if err != nil {
		logx.Errorf("用户 %s (ID: %d) 更新题目失败: id=%d, error=%v",
			user.Username, user.UserID, req.Id, err)
		return &types.UpdateProblemResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "更新题目失败，请稍后重试",
			},
		}, nil
	}

	// 11. 记录成功操作日志
	logx.Infof("题目更新成功: ID=%d, 标题=%s, 更新者=%s (ID: %d), 难度=%s, 公开=%t",
		req.Id, updatedProblem.Title, user.Username, user.UserID,
		updatedProblem.Difficulty, updatedProblem.IsPublic)

	return &types.UpdateProblemResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "题目更新成功",
		},
		Data: types.UpdateProblemData{
			ProblemId: req.Id,
			UpdatedAt: updatedProblem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Message:   "题目信息已更新",
		},
	}, nil
}

// 验证更新请求参数
func (l *UpdateProblemLogic) validateUpdateRequest(req *types.UpdateProblemReq) error {
	// 验证标题长度
	if req.Title != "" && (len(req.Title) == 0 || len(req.Title) > 200) {
		return fmt.Errorf("题目标题长度必须在1-200字符之间")
	}

	// 验证描述长度
	if req.Description != "" && len(req.Description) < 10 {
		return fmt.Errorf("题目描述至少需要10个字符")
	}

	// 验证难度级别
	if req.Difficulty != "" {
		validDifficulties := map[string]bool{"easy": true, "medium": true, "hard": true}
		if !validDifficulties[req.Difficulty] {
			return fmt.Errorf("无效的难度级别: %s", req.Difficulty)
		}
	}

	// 验证时间限制
	if req.TimeLimit > 0 && (req.TimeLimit < 100 || req.TimeLimit > 10000) {
		return fmt.Errorf("时间限制必须在100-10000毫秒之间")
	}

	// 验证内存限制
	if req.MemoryLimit > 0 && (req.MemoryLimit < 16 || req.MemoryLimit > 512) {
		return fmt.Errorf("内存限制必须在16-512MB之间")
	}

	// 验证编程语言
	if len(req.Languages) > 0 && len(req.Languages) == 0 {
		return fmt.Errorf("至少需要指定一种编程语言")
	}

	// 验证标签数量
	if len(req.Tags) > 10 {
		return fmt.Errorf("标签数量不能超过10个")
	}

	return nil
}
