package svc

import (
	"github.com/online-judge/code-judger/common/utils"
	"github.com/online-judge/code-judger/services/user-api/internal/config"
	"github.com/online-judge/code-judger/services/user-api/models"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config config.Config

	// 数据库连接
	DB sqlx.SqlConn

	// Redis连接
	RedisClient *redis.Redis

	// 数据模型
	UserModel         models.UserModel
	UserTokenModel    models.UserTokenModel
	UserStatsModel    models.UserStatisticsModel
	UserLoginLogModel models.UserLoginLogModel

	// JWT管理器
	JWTManager *utils.JWTManager

	// 中间件
	Auth      rest.Middleware
	AdminOnly rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := sqlx.NewMysql(c.DataSource)

	// 初始化Redis连接
	rds := redis.MustNewRedis(c.RedisConf)

	// 初始化JWT管理器
	jwtManager := utils.NewJWTManager(
		c.Auth.AccessSecret,
		c.Auth.RefreshSecret,
		c.Auth.AccessExpire,
		c.Auth.RefreshExpire,
	)

	// 初始化数据模型
	// 配置Redis缓存
	cacheConf := cache.CacheConf{
		{
			RedisConf: c.RedisConf,
			Weight:    100,
		},
	}
	userModel := models.NewUserModel(db, cacheConf)
	userTokenModel := models.NewUserTokenModel(db, cacheConf)
	userStatsModel := models.NewUserStatisticsModel(db, cacheConf)
	userLoginLogModel := models.NewUserLoginLogModel(db, cacheConf)

	svcCtx := &ServiceContext{
		Config:            c,
		DB:                db,
		RedisClient:       rds,
		UserModel:         userModel,
		UserTokenModel:    userTokenModel,
		UserStatsModel:    userStatsModel,
		UserLoginLogModel: userLoginLogModel,
		JWTManager:        jwtManager,
	}

	// 中间件将在main.go中单独初始化，避免循环引用
	return svcCtx
}
