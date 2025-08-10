package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	
	// 数据库配置
	DataSource string
	
	// Redis配置
	RedisConf redis.RedisConf
	
	// JWT认证配置
	Auth struct {
		AccessSecret  string
		AccessExpire  int64
		RefreshSecret string
		RefreshExpire int64
	}
	
	// Kafka配置
	KafkaConf struct {
		Brokers []string
		Topic   string
	}
	
	// 业务配置
	Business struct {
		// 密码策略
		PasswordPolicy struct {
			MinLength            int
			RequireUppercase     bool
			RequireLowercase     bool
			RequireNumbers       bool
			RequireSpecialChars  bool
		}
		
		// 登录安全
		LoginSecurity struct {
			MaxFailedAttempts int
			LockoutDuration   int
			SessionTimeout    int
		}
		
		// 注册配置
		Registration struct {
			EnableRegistration        bool
			RequireEmailVerification  bool
			DefaultRole              string
		}
		
		// 分页配置
		Pagination struct {
			DefaultPageSize int
			MaxPageSize     int
		}
	}
}