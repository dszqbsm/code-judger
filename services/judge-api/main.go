package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/online-judge/code-judger/services/judge-api/internal/config"
	"github.com/online-judge/code-judger/services/judge-api/internal/handler"
	"github.com/online-judge/code-judger/services/judge-api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/judge-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 创建工作目录
	if err := createWorkDirectories(&c); err != nil {
		logx.Errorf("Failed to create work directories: %v", err)
		os.Exit(1)
	}

	// 设置优雅关闭
	setupGracefulShutdown(ctx)

	fmt.Printf("Starting judge server at %s:%d...\n", c.RestConf.Host, c.RestConf.Port)
	logx.Infof("Judge API server starting at %s:%d", c.RestConf.Host, c.RestConf.Port)

	// 启动服务器
	server.Start()
}

// 创建工作目录
func createWorkDirectories(c *config.Config) error {
	dirs := []string{
		c.JudgeEngine.WorkDir,
		c.JudgeEngine.TempDir,
		c.JudgeEngine.DataDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	logx.Infof("Work directories created: %v", dirs)
	return nil
}

// 设置优雅关闭
func setupGracefulShutdown(ctx *svc.ServiceContext) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-c
		logx.Info("Shutting down judge server...")

		// 停止任务调度器
		if err := ctx.TaskScheduler.Stop(); err != nil {
			logx.Errorf("Failed to stop task scheduler: %v", err)
		}

		// 给系统一些时间清理资源
		time.Sleep(2 * time.Second)

		logx.Info("Judge server stopped gracefully")
		os.Exit(0)
	}()
}
