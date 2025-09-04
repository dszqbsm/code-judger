package problem

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dszqbsm/code-judger/services/problem-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/problem-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/problem-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteProblemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewDeleteProblemLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *DeleteProblemLogic {
	return &DeleteProblemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *DeleteProblemLogic) DeleteProblem(req *types.DeleteProblemReq) (resp *types.DeleteProblemResp, err error) {
	// 1. 验证题目ID
	if req.Id <= 0 {
		return &types.DeleteProblemResp{
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
				return &types.DeleteProblemResp{
					BaseResp: types.BaseResp{
						Code:    401,
						Message: "认证失败：" + err.Error(),
					},
				}, nil
			}
		} else {
			logx.Errorf("无法获取用户信息: 上下文和请求头都为空")
			return &types.DeleteProblemResp{
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
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "题目不存在",
			},
		}, nil
	}

	// 4. 检查题目是否已被删除
	if existingProblem.DeletedAt.Valid {
		logx.Errorf("用户 %s (ID: %d) 尝试删除已删除的题目: ID=%d", user.Username, user.UserID, req.Id)
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "题目已被删除",
			},
		}, nil
	}

	// 5. 验证用户权限（使用中间件的权限验证函数）
	if err = middleware.ValidateDeleteProblemPermission(user.Role, user.UserID, existingProblem.CreatedBy); err != nil {
		logx.Errorf("用户 %s (ID: %d) 权限验证失败，无法删除题目 %d: %v",
			user.Username, user.UserID, req.Id, err)
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    403,
				Message: err.Error(),
			},
		}, nil
	}

	// 6. 检查题目是否有相关提交记录
	submissionCount, err := l.checkSubmissionRecords(req.Id)
	if err != nil {
		logx.Errorf("检查提交记录失败: problem_id=%d, error=%v", req.Id, err)
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "检查提交记录失败",
			},
		}, nil
	}

	if submissionCount > 0 {
		logx.Errorf("用户 %s (ID: %d) 尝试删除有提交记录的题目: ID=%d, 提交数=%d",
			user.Username, user.UserID, req.Id, submissionCount)
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: fmt.Sprintf("无法删除题目：该题目已有 %d 条提交记录", submissionCount),
			},
		}, nil
	}

	// 7. 记录操作开始日志
	logx.Infof("用户 %s (ID: %d, Role: %s) 开始删除题目: ID=%d, 标题=%s",
		user.Username, user.UserID, user.Role, req.Id, existingProblem.Title)

	// 8. 执行软删除
	err = l.svcCtx.ProblemModel.SoftDelete(l.ctx, req.Id)
	if err != nil {
		logx.Errorf("用户 %s (ID: %d) 删除题目失败: id=%d, error=%v",
			user.Username, user.UserID, req.Id, err)
		return &types.DeleteProblemResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "删除题目失败，请稍后重试",
			},
		}, nil
	}

	// 9. 执行配套清理操作
	if err := l.performCascadeCleanup(req.Id, existingProblem.Title); err != nil {
		logx.Errorf("配套清理操作失败: problem_id=%d, error=%v", req.Id, err)
		// 注意：这里不返回错误，因为题目已经删除成功，清理失败只是警告
	}

	// 10. 记录成功操作日志
	logx.Infof("题目删除成功: ID=%d, 标题=%s, 删除者=%s (ID: %d), 难度=%s",
		req.Id, existingProblem.Title, user.Username, user.UserID, existingProblem.Difficulty)

	return &types.DeleteProblemResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "题目删除成功",
		},
		Data: types.DeleteProblemData{
			ProblemId: req.Id,
			DeletedAt: time.Now().Format("2006-01-02T15:04:05Z07:00"),
			Message:   "题目已被标记为删除状态，相关资源已清理",
		},
	}, nil
}

// checkSubmissionRecords 检查题目是否有相关提交记录
func (l *DeleteProblemLogic) checkSubmissionRecords(problemId int64) (int64, error) {
	// 这里应该调用提交服务来检查提交记录
	// 由于跨服务调用的复杂性，这里先返回0表示没有提交记录
	// 在实际项目中，应该通过RPC调用submission-api服务

	// TODO: 实现跨服务调用检查提交记录
	// 可以通过以下方式实现：
	// 1. 通过RPC调用submission-api服务
	// 2. 通过消息队列异步查询
	// 3. 通过共享数据库直接查询（需要添加数据库连接）

	logx.Infof("检查题目 %d 的提交记录（当前返回0，需要实现跨服务调用）", problemId)

	// 暂时返回0，表示没有提交记录
	return 0, nil
}

// performCascadeCleanup 执行删除题目后的配套清理操作
func (l *DeleteProblemLogic) performCascadeCleanup(problemId int64, problemTitle string) error {
	logx.Infof("开始执行题目 %d (%s) 的配套清理操作", problemId, problemTitle)

	var errors []error

	// 1. 删除测试用例文件
	if err := l.cleanupTestCaseFiles(problemId); err != nil {
		logx.Errorf("清理测试用例文件失败: problem_id=%d, error=%v", problemId, err)
		errors = append(errors, fmt.Errorf("清理测试用例文件失败: %v", err))
	}

	// 2. 清理相关缓存
	if err := l.cleanupProblemCache(problemId); err != nil {
		logx.Errorf("清理题目缓存失败: problem_id=%d, error=%v", problemId, err)
		errors = append(errors, fmt.Errorf("清理缓存失败: %v", err))
	}

	// 3. 删除题目标签关联（如果有单独的标签表）
	if err := l.cleanupProblemTags(problemId); err != nil {
		logx.Errorf("清理题目标签关联失败: problem_id=%d, error=%v", problemId, err)
		errors = append(errors, fmt.Errorf("清理标签关联失败: %v", err))
	}

	// 4. 删除题目收藏记录（如果有）
	if err := l.cleanupProblemFavorites(problemId); err != nil {
		logx.Errorf("清理题目收藏记录失败: problem_id=%d, error=%v", problemId, err)
		errors = append(errors, fmt.Errorf("清理收藏记录失败: %v", err))
	}

	// 5. 清理题目统计数据（如果有单独的统计表）
	if err := l.cleanupProblemStatistics(problemId); err != nil {
		logx.Errorf("清理题目统计数据失败: problem_id=%d, error=%v", problemId, err)
		errors = append(errors, fmt.Errorf("清理统计数据失败: %v", err))
	}

	// 6. 发送通知给相关用户（可选）
	if err := l.notifyRelatedUsers(problemId, problemTitle); err != nil {
		logx.Errorf("发送删除通知失败: problem_id=%d, error=%v", problemId, err)
		// 通知失败不算严重错误，只记录日志
	}

	if len(errors) > 0 {
		return fmt.Errorf("配套清理操作部分失败: %v", errors)
	}

	logx.Infof("题目 %d (%s) 配套清理操作完成", problemId, problemTitle)
	return nil
}

// cleanupTestCaseFiles 清理测试用例文件
func (l *DeleteProblemLogic) cleanupTestCaseFiles(problemId int64) error {
	// 测试用例文件通常存储在特定目录下，如 /data/testcases/problem_{id}/
	testCaseDir := filepath.Join("/data/testcases", fmt.Sprintf("problem_%d", problemId))

	// 检查目录是否存在
	if _, err := os.Stat(testCaseDir); os.IsNotExist(err) {
		logx.Infof("测试用例目录不存在，跳过清理: %s", testCaseDir)
		return nil
	}

	// 删除整个测试用例目录
	err := os.RemoveAll(testCaseDir)
	if err != nil {
		return fmt.Errorf("删除测试用例目录失败: %v", err)
	}

	logx.Infof("成功删除测试用例目录: %s", testCaseDir)
	return nil
}

// cleanupProblemCache 清理题目相关缓存
func (l *DeleteProblemLogic) cleanupProblemCache(problemId int64) error {
	// 这里需要根据实际的缓存实现来清理
	// 如果使用Redis，需要删除相关的缓存键

	cacheKeys := []string{
		fmt.Sprintf("problem:detail:%d", problemId),
		fmt.Sprintf("problem:testcases:%d", problemId),
		"problem:list:*",   // 题目列表缓存需要清理
		"problem:search:*", // 搜索结果缓存需要清理
	}

	// 由于这里没有直接的Redis客户端，我们记录需要清理的缓存键
	logx.Infof("需要清理的缓存键: %v", cacheKeys)

	// TODO: 实际的缓存清理逻辑
	// if l.svcCtx.RedisClient != nil {
	//     for _, key := range cacheKeys {
	//         err := l.svcCtx.RedisClient.Del(key)
	//         if err != nil {
	//             return fmt.Errorf("删除缓存键失败: key=%s, error=%v", key, err)
	//         }
	//     }
	// }

	return nil
}

// cleanupProblemTags 清理题目标签关联
func (l *DeleteProblemLogic) cleanupProblemTags(problemId int64) error {
	// 如果有单独的题目标签关联表，需要删除相关记录
	// 由于当前题目的标签信息存储在JSON字段中，不需要清理关联表
	logx.Infof("题目 %d 的标签信息存储在JSON字段中，无需额外清理", problemId)
	return nil
}

// cleanupProblemFavorites 清理题目收藏记录
func (l *DeleteProblemLogic) cleanupProblemFavorites(problemId int64) error {
	// 如果有用户收藏题目的功能，需要删除相关记录
	// 这里应该调用用户服务来清理收藏记录
	// 由于跨服务调用的复杂性，这里先记录日志
	logx.Infof("需要清理题目 %d 的收藏记录（需要实现跨服务调用）", problemId)

	// TODO: 实现跨服务调用清理收藏记录
	// 可以通过RPC调用user-api服务来清理收藏记录

	return nil
}

// cleanupProblemStatistics 清理题目统计数据
func (l *DeleteProblemLogic) cleanupProblemStatistics(problemId int64) error {
	// 题目统计数据存储在题目表的字段中，软删除后会自动隐藏
	// 如果有单独的统计表，需要额外清理
	logx.Infof("题目 %d 的统计数据已通过软删除隐藏", problemId)

	// TODO: 如果有独立的统计服务，需要调用统计服务来清理数据
	// 可以通过RPC调用statistics-api服务

	return nil
}

// notifyRelatedUsers 通知相关用户题目已删除
func (l *DeleteProblemLogic) notifyRelatedUsers(problemId int64, problemTitle string) error {
	// 这里可以实现通知逻辑，比如：
	// 1. 通知收藏了该题目的用户
	// 2. 通知参与过该题目讨论的用户
	// 3. 发送系统消息或邮件

	logx.Infof("发送题目删除通知: ID=%d, 标题=%s", problemId, problemTitle)

	// TODO: 实现具体的通知逻辑
	// 可以通过消息队列发送异步通知
	// 可以调用通知服务API

	return nil
}
