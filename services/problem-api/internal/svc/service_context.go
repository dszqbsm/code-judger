package svc

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"

	"code-judger/services/problem-api/internal/config"
	"code-judger/services/problem-api/models"
)

type ServiceContext struct {
	Config config.Config

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

	return &ServiceContext{
		Config:       c,
		ProblemModel: models.NewProblemModel(db),
	}
}