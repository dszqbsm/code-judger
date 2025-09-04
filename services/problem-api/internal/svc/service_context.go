package svc

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"

	"github.com/dszqbsm/code-judger/common/utils"
	"github.com/dszqbsm/code-judger/services/problem-api/internal/config"
	"github.com/dszqbsm/code-judger/services/problem-api/models"
)

type ServiceContext struct {
	Config config.Config

	// JWT管理器
	JWTManager *utils.JWTManager

	// 数据库模型
	ProblemModel models.ProblemModel

	// 可以添加其他依赖，如：
	// UserRpc      userclient.User
	// RedisClient  *redis.Redis
	// Logger       logx.Logger
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db, err := sql.Open("mysql", c.DataSource)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 初始化JWT管理器
	jwtManager := utils.NewJWTManager(
		c.Auth.AccessSecret,
		"", // 题目服务不需要刷新令牌功能
		c.Auth.AccessExpire,
		0,
	)

	return &ServiceContext{
		Config:       c,
		JWTManager:   jwtManager,
		ProblemModel: models.NewProblemModel(db),
	}
}