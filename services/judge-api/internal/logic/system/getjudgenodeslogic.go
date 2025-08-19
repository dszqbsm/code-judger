package system

import (
	"context"
	"runtime"
	"time"

	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetJudgeNodesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetJudgeNodesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetJudgeNodesLogic {
	return &GetJudgeNodesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetJudgeNodesLogic) GetJudgeNodes(req *types.GetJudgeNodesReq) (resp *types.GetJudgeNodesResp, err error) {
	// 获取当前节点信息
	nodeId := l.svcCtx.Config.Cluster.NodeId
	nodeName := l.svcCtx.Config.Cluster.NodeName

	// 获取系统信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 计算CPU和内存使用率（简化实现）
	cpuUsage := float64(runtime.NumGoroutine()) / 1000.0 // 简化的CPU使用率计算
	if cpuUsage > 1.0 {
		cpuUsage = 1.0
	}

	memoryUsage := float64(memStats.Alloc) / float64(memStats.Sys)

	// 获取任务统计
	stats := l.svcCtx.TaskScheduler.GetStats()

	// 构建节点信息
	node := types.JudgeNode{
		NodeId:        nodeId,
		NodeName:      nodeName,
		Status:        "online",
		CpuUsage:      cpuUsage * 100,
		MemoryUsage:   memoryUsage * 100,
		ActiveTasks:   int(stats.RunningTasks),
		TotalTasks:    int(stats.TotalTasks),
		LastHeartbeat: time.Now().Format(time.RFC3339),
	}

	logx.Infof("Retrieved judge nodes info: node_id=%s, active_tasks=%d", nodeId, node.ActiveTasks)

	return &types.GetJudgeNodesResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: types.JudgeNodesData{
			Nodes: []types.JudgeNode{node},
			Total: 1,
		},
	}, nil
}
