package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/online-judge/code-judger/services/submission-api/internal/anticheat"
	"github.com/online-judge/code-judger/services/submission-api/internal/config"
	"github.com/online-judge/code-judger/services/submission-api/internal/messagequeue"
	"github.com/online-judge/code-judger/services/submission-api/internal/middleware"
	"github.com/online-judge/code-judger/services/submission-api/internal/websocket"
	"github.com/online-judge/code-judger/services/submission-api/models"
)

type ServiceContext struct {
	Config config.Config

	// 数据库连接
	DB sqlx.SqlConn

	// Redis连接
	RedisClient *redis.Redis

	// 数据模型
	SubmissionModel models.SubmissionModel

	// 中间件
	Auth      middleware.AuthMiddleware
	AdminOnly middleware.AdminOnlyMiddleware

	// WebSocket管理器
	WSManager *websocket.Manager

	// 消息队列
	MessageQueue messagequeue.Producer

	// 查重检测器
	AntiCheatDetector *anticheat.Detector

	// 限流器
	RateLimiter *middleware.RateLimiter
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := sqlx.NewMysql(c.DataSource)

	// 初始化Redis连接
	redisClient := redis.MustNewRedis(c.RedisConf)

	// 初始化数据模型
	submissionModel := models.NewSubmissionModel(db, c.CacheConf)

	// 初始化中间件
	authMiddleware := middleware.NewAuthMiddleware(c.Auth.AccessSecret, redisClient)
	adminOnlyMiddleware := middleware.NewAdminOnlyMiddleware()
	rateLimiter := middleware.NewRateLimiter(redisClient, c.Business.MaxSubmissionPerMinute)

	// 初始化WebSocket管理器
	wsManager := websocket.NewManager(c.WebSocket, redisClient)

	// 初始化消息队列
	producer := messagequeue.NewKafkaProducer(c.KafkaConf)

	// 初始化查重检测器
	antiCheatDetector := anticheat.NewDetector(c.AntiCheat, submissionModel)

	return &ServiceContext{
		Config:            c,
		DB:                db,
		RedisClient:       redisClient,
		SubmissionModel:   submissionModel,
		Auth:              authMiddleware,
		AdminOnly:         adminOnlyMiddleware,
		WSManager:         wsManager,
		MessageQueue:      producer,
		AntiCheatDetector: antiCheatDetector,
		RateLimiter:       rateLimiter,
	}
}

