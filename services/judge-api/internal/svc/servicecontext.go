package svc

import (
	"context"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/client"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/config"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/judge"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/messagequeue"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/scheduler"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config *config.Config

	// 数据库连接
	DB sqlx.SqlConn

	// 缓存
	Cache cache.Cache

	// 判题引擎
	JudgeEngine *judge.JudgeEngine

	// 任务调度器
	TaskScheduler *scheduler.TaskScheduler

	// 题目服务客户端
	ProblemClient client.ProblemServiceClient

	// 消息队列
	KafkaConsumer messagequeue.Consumer
	KafkaProducer messagequeue.Producer
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := sqlx.NewMysql(c.DataSource)

	// 初始化缓存
	cacheConf := cache.CacheConf{
		cache.NodeConf{
			RedisConf: c.RedisConf,
			Weight:    100,
		},
	}

	cacheClient := cache.New(cacheConf, nil, cache.NewStat("judge-api"), nil)

	// 初始化判题引擎
	judgeEngine := judge.NewJudgeEngine(&c.JudgeEngine)

	// 初始化任务调度器
	taskScheduler := scheduler.NewTaskScheduler(&c.TaskQueue, judgeEngine)

	// 启动任务调度器
	if err := taskScheduler.Start(); err != nil {
		logx.Errorf("Failed to start task scheduler: %v", err)
		panic(err)
	}

	// 初始化题目服务客户端
	var problemClient client.ProblemServiceClient
	if c.ProblemService.UseMock {
		problemClient = client.NewMockProblemClient()
		logx.Info("Using mock problem service client for development")
	} else if c.ProblemService.RPC.Enabled {
		// RPC暂时使用Mock，等RPC服务完善后再启用
		problemClient = client.NewMockProblemClient()
		logx.Infof("RPC client not implemented yet, using mock client for problem service")
	} else {
		// 兼容HTTP调用（向后兼容）
		baseClient := client.NewHttpProblemClientWithAuth(c.ProblemService.HTTP.Endpoint, c.ProblemService.HTTP.ApiKey)

		// 暂时不使用重试机制，直接使用基础客户端
		problemClient = baseClient
		logx.Infof("Using HTTP problem service client: %s", c.ProblemService.HTTP.Endpoint)
	}

	// 初始化Kafka生产者
	kafkaProducer := messagequeue.NewKafkaProducer(c.KafkaConf)

	// 初始化Kafka消费者（传入题目服务客户端）
	kafkaConsumer := messagequeue.NewKafkaConsumer(c.KafkaConf, taskScheduler, kafkaProducer, problemClient)

	// 启动Kafka消费者
	if err := kafkaConsumer.Start(context.Background()); err != nil {
		logx.Errorf("Failed to start Kafka consumer: %v", err)
		panic(err)
	}

	return &ServiceContext{
		Config:        &c,
		DB:            db,
		Cache:         cacheClient,
		JudgeEngine:   judgeEngine,
		TaskScheduler: taskScheduler,
		ProblemClient: problemClient,
		KafkaConsumer: kafkaConsumer,
		KafkaProducer: kafkaProducer,
	}
}
