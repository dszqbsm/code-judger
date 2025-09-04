package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/config"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/handler"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/common/consul"

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

	// 注册到Consul
	var consulRegistry *consul.ServiceRegistry
	if c.Consul.Enabled {
		var err error
		consulRegistry, err = registerToConsul(&c)
		if err != nil {
			logx.Errorf("Failed to register to Consul: %v", err)
		} else {
			logx.Info("Successfully registered to Consul")
		}
	}

	// 设置优雅关闭
	setupGracefulShutdown(ctx, consulRegistry)

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

// 注册到Consul
func registerToConsul(c *config.Config) (*consul.ServiceRegistry, error) {
	// 解析服务地址
	host := c.RestConf.Host
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	port := c.RestConf.Port

	// 创建服务注册器
	registry, err := consul.NewServiceRegistry(c.Consul.Address, fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, fmt.Errorf("创建Consul服务注册器失败: %w", err)
	}

	// 解析健康检查间隔
	healthInterval, err := time.ParseDuration(c.Consul.HealthInterval)
	if err != nil {
		healthInterval = 10 * time.Second // 默认10秒
	}

	// 解析健康检查超时
	healthTimeout, err := time.ParseDuration(c.Consul.HealthTimeout)
	if err != nil {
		healthTimeout = 3 * time.Second // 默认3秒
	}

	// 解析注销时间
	deregisterAfter, err := time.ParseDuration(c.Consul.DeregisterAfter)
	if err != nil {
		deregisterAfter = 30 * time.Second // 默认30秒
	}

	// 构建服务信息
	serviceInfo := consul.ServiceInfo{
		ServiceName: c.Consul.ServiceName,
		ServiceID:   c.Consul.ServiceID,
		Address:     host,
		Port:        port,
		Tags:        c.Consul.Tags,
		Health: consul.HealthCheck{
			HTTP:                           c.Consul.HealthCheckURL,
			Interval:                       healthInterval,
			Timeout:                        healthTimeout,
			DeregisterCriticalServiceAfter: deregisterAfter,
		},
		Meta: map[string]string{
			"version":     "1.0.0",
			"protocol":    "http",
			"service":     "judge-api",
		},
	}

	// 注册服务
	if err := registry.Register(serviceInfo); err != nil {
		return nil, fmt.Errorf("注册服务到Consul失败: %w", err)
	}

	logx.Infof("服务已注册到Consul: ServiceName=%s, ServiceID=%s, Address=%s:%d", 
		serviceInfo.ServiceName, serviceInfo.ServiceID, serviceInfo.Address, serviceInfo.Port)

	return registry, nil
}

// 设置优雅关闭
func setupGracefulShutdown(ctx *svc.ServiceContext, consulRegistry *consul.ServiceRegistry) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-c
		logx.Info("Shutting down judge server...")

		// 停止Kafka消费者
		if err := ctx.KafkaConsumer.Stop(); err != nil {
			logx.Errorf("Failed to stop Kafka consumer: %v", err)
		}

		// 停止Kafka生产者
		if err := ctx.KafkaProducer.Close(); err != nil {
			logx.Errorf("Failed to close Kafka producer: %v", err)
		}

		// 停止任务调度器
		if err := ctx.TaskScheduler.Stop(); err != nil {
			logx.Errorf("Failed to stop task scheduler: %v", err)
		}

		// 从Consul注销服务
		if consulRegistry != nil {
			if err := consulRegistry.Deregister(); err != nil {
				logx.Errorf("Failed to deregister from Consul: %v", err)
			} else {
				logx.Info("Successfully deregistered from Consul")
			}
		}

		// 给系统一些时间清理资源
		time.Sleep(2 * time.Second)

		logx.Info("Judge server stopped gracefully")
		os.Exit(0)
	}()
}
