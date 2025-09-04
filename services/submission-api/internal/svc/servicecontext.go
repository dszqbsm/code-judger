package svc

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/dszqbsm/code-judger/common/consul"
	"github.com/dszqbsm/code-judger/common/utils"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/anticheat"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/client"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/config"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/dao"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/messagequeue"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/queue"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/websocket"
	"github.com/dszqbsm/code-judger/services/submission-api/models"
)

type ServiceContext struct {
	Config config.Config

	// 数据库连接
	DB sqlx.SqlConn

	// Redis连接
	RedisClient *redis.Redis

	// JWT管理器
	JWTManager *utils.JWTManager

	// 数据模型
	SubmissionModel models.SubmissionModel

	// DAO层
	SubmissionDao dao.SubmissionDao

	// 中间件
	Auth      middleware.AuthMiddleware
	AdminOnly middleware.AdminOnlyMiddleware

	// WebSocket管理器
	WSManager *websocket.Manager

	// 消息队列
	MessageQueue messagequeue.Producer
	Consumer     messagequeue.Consumer

	// 查重检测器
	AntiCheatDetector *anticheat.Detector

	// 限流器
	RateLimiter *middleware.RateLimiter

	// 队列管理器
	QueueManager *queue.QueueManager

	// 判题服务客户端
	JudgeClient client.JudgeServiceClient

	// 题目服务客户端
	ProblemClient client.ProblemServiceClient

	// Consul服务注册器
	ServiceRegistry *consul.ServiceRegistry
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := sqlx.NewMysql(c.DataSource)

	// 初始化Redis连接
	redisClient := redis.MustNewRedis(c.RedisConf)

	// 初始化JWT管理器
	jwtManager := utils.NewJWTManager(
		c.Auth.AccessSecret,
		"", // submission服务不需要刷新令牌功能
		c.Auth.AccessExpire,
		0,
	)

	// 初始化数据模型
	submissionModel := models.NewSubmissionModel(db, c.CacheConf)

	// 初始化DAO层
	submissionDao := dao.NewSubmissionDao(db, submissionModel)

	// 初始化中间件
	authMiddleware := middleware.NewAuthMiddleware(c.Auth.AccessSecret, redisClient)
	adminOnlyMiddleware := middleware.NewAdminOnlyMiddleware()
	rateLimiter := middleware.NewRateLimiter(redisClient, c.Business.MaxSubmissionPerMinute)

	// 初始化WebSocket管理器
	wsManager := websocket.NewManager(c.WebSocket, redisClient)

	// 初始化消息队列生产者
	producer := messagequeue.NewKafkaProducer(c.KafkaConf)

	// 初始化消息队列消费者
	consumer := messagequeue.NewKafkaConsumer(c.KafkaConf, wsManager, submissionDao)

	// 初始化查重检测器
	antiCheatDetector := anticheat.NewDetector(c.AntiCheat, submissionModel)

	// 初始化队列管理器
	queueManager := queue.NewQueueManager(redisClient)

	// 初始化判题服务客户端
	var judgeClient client.JudgeServiceClient
	var problemClient client.ProblemServiceClient
	var serviceRegistry *consul.ServiceRegistry

	if c.Consul.Enabled && c.RPC.Enabled {
		// 使用Consul + RPC方式
		var err error
		judgeClient, err = client.NewJudgeRPCClient(c.Consul.Address, c.RPC.DefaultTimeout)
		if err != nil {
			logx.Errorf("创建判题服务RPC客户端失败: %v", err)
			// 回退到HTTP客户端
			judgeClient = client.NewHttpJudgeClient(
				c.JudgeService.Endpoint,
				time.Duration(c.JudgeService.Timeout)*time.Second,
			)
		}

		problemClient, err = client.NewProblemRPCClient(c.Consul.Address, c.RPC.DefaultTimeout)
		if err != nil {
			logx.Errorf("创建题目服务RPC客户端失败: %v", err)
		}

		// 初始化Consul服务注册器
		serviceRegistry, err = initServiceRegistry(c)
		if err != nil {
			logx.Errorf("初始化Consul服务注册器失败: %v", err)
		}
	} else {
		// 使用传统HTTP方式
		judgeClient = client.NewHttpJudgeClient(
			c.JudgeService.Endpoint,
			time.Duration(c.JudgeService.Timeout)*time.Second,
		)
	}

	return &ServiceContext{
		Config:            c,
		DB:                db,
		RedisClient:       redisClient,
		JWTManager:        jwtManager,
		SubmissionModel:   submissionModel,
		SubmissionDao:     submissionDao,
		Auth:              authMiddleware,
		AdminOnly:         adminOnlyMiddleware,
		WSManager:         wsManager,
		MessageQueue:      producer,
		Consumer:          consumer,
		AntiCheatDetector: antiCheatDetector,
		RateLimiter:       rateLimiter,
		QueueManager:      queueManager,
		JudgeClient:       judgeClient,
		ProblemClient:     problemClient,
		ServiceRegistry:   serviceRegistry,
	}
}

// initServiceRegistry 初始化Consul服务注册器
func initServiceRegistry(c config.Config) (*consul.ServiceRegistry, error) {
	// 解析服务地址
	host, port, err := parseServiceAddr(c.Host, c.Port)
	if err != nil {
		return nil, fmt.Errorf("解析服务地址失败: %w", err)
	}

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
			"service":     "submission-api",
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

// parseServiceAddr 解析服务地址
func parseServiceAddr(host string, port int) (string, int, error) {
	// 如果host为空或为0.0.0.0，使用本地IP
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1" // 在生产环境中应该获取真实的本地IP
	}

	if port <= 0 {
		return "", 0, fmt.Errorf("无效的端口号: %d", port)
	}

	return host, port, nil
}

// StartBackgroundServices 启动后台服务
func (sc *ServiceContext) StartBackgroundServices(ctx context.Context) error {
	// 启动消息队列消费者
	if err := sc.Consumer.Start(ctx); err != nil {
		return err
	}

	return nil
}

// StopBackgroundServices 停止后台服务
func (sc *ServiceContext) StopBackgroundServices() error {
	// 停止消息队列消费者
	if err := sc.Consumer.Stop(); err != nil {
		logx.Errorf("停止消息队列消费者失败: %v", err)
	}

	// 从Consul注销服务
	if sc.ServiceRegistry != nil {
		if err := sc.ServiceRegistry.Deregister(); err != nil {
			logx.Errorf("从Consul注销服务失败: %v", err)
		}
	}

	return nil
}
