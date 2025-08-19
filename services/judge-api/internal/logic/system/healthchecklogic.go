package system

import (
	"context"
	"runtime"
	"time"

	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var startTime = time.Now()

type HealthCheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHealthCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthCheckLogic {
	return &HealthCheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HealthCheckLogic) HealthCheck(req *types.HealthCheckReq) (resp *types.HealthCheckResp, err error) {
	// 检查判题引擎健康状态
	engineErr := l.svcCtx.JudgeEngine.HealthCheck()
	status := "healthy"
	if engineErr != nil {
		status = "unhealthy"
		logx.Errorf("Judge engine health check failed: %v", engineErr)
	}

	// 获取系统信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 计算系统资源使用情况
	systemInfo := types.SystemInfo{
		CpuUsage:       float64(runtime.NumGoroutine()) / 1000.0 * 100, // 简化的CPU使用率
		MemoryUsage:    float64(memStats.Alloc) / float64(memStats.Sys) * 100,
		DiskUsage:      0.0, // TODO: 实现磁盘使用率检查
		GoroutineCount: runtime.NumGoroutine(),
	}

	// 限制CPU使用率显示
	if systemInfo.CpuUsage > 100 {
		systemInfo.CpuUsage = 100
	}

	// 计算运行时间
	uptime := time.Since(startTime).Milliseconds()

	logx.Infof("Health check completed: status=%s, uptime=%dms, goroutines=%d",
		status, uptime, systemInfo.GoroutineCount)

	return &types.HealthCheckResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "健康检查完成",
		},
		Data: types.HealthData{
			Status:     status,
			Timestamp:  time.Now().Format(time.RFC3339),
			Version:    "1.0.0", // TODO: 从配置或构建信息获取
			Uptime:     uptime,
			SystemInfo: systemInfo,
		},
	}, nil
}
