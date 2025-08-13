package config

import (
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	
	// 数据库配置
	DataSource string `json:",default=root:password@tcp(localhost:3306)/oj_problems?charset=utf8mb4&parseTime=true&loc=Local"`
	
	// Redis缓存配置
	// CacheConf cache.CacheConf
	
	// JWT认证配置
	Auth struct {
		AccessSecret string `json:",default=your-access-secret"`
		AccessExpire int64  `json:",default=3600"`
	}
	
	// 服务注册配置
	Consul struct {
		Host string `json:",default=localhost:8500"`
		Key  string `json:",default=problem-api"`
	}
	
	// 业务配置
	Business struct {
		// 分页配置
		DefaultPageSize int `json:",default=20"`
		MaxPageSize     int `json:",default=100"`
		
		// 缓存配置
		ProblemListCacheTTL   int `json:",default=300"`  // 题目列表缓存5分钟
		ProblemDetailCacheTTL int `json:",default=1800"` // 题目详情缓存30分钟
		
		// 文件上传配置
		MaxFileSize int64 `json:",default=10485760"` // 10MB
	}
	
	// 日志配置
	Log struct {
		ServiceName string `json:",default=problem-api"`
		Mode        string `json:",default=file"`
		Path        string `json:",default=logs"`
		Level       string `json:",default=info"`
	}
}