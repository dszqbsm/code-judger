package problem

import (
	"context"
	"time"

	"code-judger/services/problem-api/internal/svc"
	"code-judger/services/problem-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteProblemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteProblemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProblemLogic {
	return &DeleteProblemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteProblemLogic) DeleteProblem(req *types.DeleteProblemReq) (resp *types.DeleteProblemResp, err error) {
	// 验证题目ID
	if req.Id <= 0 {
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "无效的题目ID",
			},
		}, nil
	}

	// 获取当前用户ID（从JWT中获取，这里先模拟）
	currentUserId := int64(1) // 临时硬编码

	// 查询现有题目
	existingProblem, err := l.svcCtx.ProblemModel.FindOne(l.ctx, req.Id)
	if err != nil {
		logx.Errorf("Failed to find problem: id=%d, error=%v", req.Id, err)
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "题目不存在",
			},
		}, nil
	}

	// 检查题目是否已被删除
	if existingProblem.DeletedAt.Valid {
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "题目已被删除",
			},
		}, nil
	}

	// 检查权限（只有创建者和管理员可以删除）
	if existingProblem.CreatedBy != currentUserId {
		// TODO: 检查是否为管理员
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    403,
				Message: "没有权限删除此题目",
			},
		}, nil
	}

	// 检查题目是否有相关提交记录
	if existingProblem.SubmissionCount > 0 {
		// 如果有提交记录，可以选择禁止删除或给出警告
		logx.Infof("Warning: Attempting to delete problem with submissions: id=%d, submissions=%d", 
			req.Id, existingProblem.SubmissionCount)
		// 这里我们仍然允许删除，但在实际业务中可能需要更严格的控制
	}

	// 执行软删除
	err = l.svcCtx.ProblemModel.SoftDelete(l.ctx, req.Id)
	if err != nil {
		logx.Errorf("Failed to delete problem: id=%d, error=%v", req.Id, err)
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "删除题目失败",
			},
		}, nil
	}

	// 记录操作日志
	logx.Infof("Problem deleted successfully: id=%d, title=%s, deleted_by=%d", 
		req.Id, existingProblem.Title, currentUserId)

	// TODO: 可以在这里添加其他清理操作，如：
	// 1. 删除相关的测试用例文件
	// 2. 清理相关缓存
	// 3. 发送通知给相关用户
	// 4. 记录到审计日志

	return &types.DeleteProblemResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "题目删除成功",
		},
		Data: types.DeleteProblemData{
			ProblemId: req.Id,
			DeletedAt: time.Now().Format("2006-01-02T15:04:05Z07:00"),
			Message:   "题目已被标记为删除状态",
		},
	}, nil
}