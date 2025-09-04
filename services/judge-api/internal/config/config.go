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

	// Kafka配置
	KafkaConf KafkaConf `json:",omitempty"`

	// JWT配置
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	// 自定义监控配置
	CustomMetrics struct {
		Host string
		Port int
		Path string
	} `json:",omitempty"`

	// 判题引擎配置
	JudgeEngine JudgeEngineConf

	// 任务队列配置
	TaskQueue TaskQueueConf

	// 缓存配置
	Cache CacheConf

	// 监控配置
	Monitor MonitorConf `json:",omitempty"`

	// 集群配置
	Cluster ClusterConf `json:",omitempty"`

	// 题目服务配置
	ProblemService ProblemServiceConf

	// Consul配置
	Consul ConsulConf
}

// Kafka配置
type KafkaConf struct {
	Brokers         []string `json:"brokers"`
	Topic           string   `json:"topic"`             // 判题任务队列
	Group           string   `json:"group"`             // 消费者组
	ResultTopic     string   `json:"result_topic"`      // 判题结果队列
	StatusTopic     string   `json:"status_topic"`      // 状态更新队列
	DeadLetterTopic string   `json:"dead_letter_topic"` // 死信队列
}

// 判题引擎配置
type JudgeEngineConf struct {
	// 工作目录配置
	WorkDir string
	TempDir string
	DataDir string

	// 沙箱配置
	Sandbox SandboxConf

	// 资源限制配置
	ResourceLimits ResourceLimitsConf

	// 编译器配置
	Compilers map[string]CompilerConf

	// 安全配置
	Security SecurityConf
}

// 沙箱配置
type SandboxConf struct {
	EnableSeccomp bool
	EnableChroot  bool
	EnablePtrace  bool
	JailUser      string
	JailUID       int
	JailGID       int
	MaxProcesses  int
}

// 资源限制配置
type ResourceLimitsConf struct {
	DefaultTimeLimit   int // 默认时间限制(毫秒)
	DefaultMemoryLimit int // 默认内存限制(MB)
	MaxTimeLimit       int // 最大时间限制(毫秒)
	MaxMemoryLimit     int // 最大内存限制(MB)
	MaxOutputSize      int // 最大输出大小(10MB)
	MaxStackSize       int // 最大栈大小(8MB)
	MaxFileSize        int // 最大文件大小(10MB)
}

// 编译器配置
type CompilerConf struct {
	Name             string `json:",omitempty"`
	Version          string `json:",omitempty"`
	FileExtension    string
	CompileCommand   string `json:",omitempty"`
	ExecuteCommand   string `json:",omitempty"`
	CompileTimeout   int
	TimeMultiplier   float64
	MemoryMultiplier float64
	MaxProcesses     int
	AllowedSyscalls  []int `json:",omitempty"`
}

// 任务队列配置
type TaskQueueConf struct {
	MaxWorkers      int // 最大工作协程数
	QueueSize       int // 队列大小
	TaskTimeout     int // 任务超时时间(秒)
	RetryTimes      int // 重试次数
	RetryInterval   int // 重试间隔(秒)
	AverageTaskTime int // 平均任务执行时间(秒)，用于预估等待时间
}

// 缓存配置
type CacheConf struct {
	JudgeResultExpire    int // 判题结果缓存过期时间(秒)
	QueueStatusExpire    int // 队列状态缓存过期时间(秒)
	LanguageConfigExpire int // 语言配置缓存过期时间(秒)
}

// 监控配置
type MonitorConf struct {
	EnableMetrics   bool   // 启用指标收集
	MetricsInterval int    // 指标收集间隔(秒)
	EnableTracing   bool   // 启用链路追踪
	TracingEndpoint string `json:",omitempty"` // 追踪服务端点
}

// 安全配置
type SecurityConf struct {
	MaxCodeLength     int                  // 最大代码长度
	MaxInputLength    int                  // 最大输入长度(10MB)
	ForbiddenPatterns []string             `json:",omitempty"` // 禁止的代码模式
	FileSystemLimits  FileSystemLimitsConf // 文件系统限制
}

// 文件系统限制配置
type FileSystemLimitsConf struct {
	MaxOpenFiles  int      // 最大打开文件数
	MaxFileSize   int      // 最大文件大小
	ReadOnlyPaths []string `json:",omitempty"` // 只读路径
	WritablePaths []string `json:",omitempty"` // 可写路径
}

// 集群配置
type ClusterConf struct {
	NodeId            string // 节点ID
	NodeName          string // 节点名称
	HeartbeatInterval int    // 心跳间隔(秒)
	NodeTimeout       int    // 节点超时时间(秒)
}

// 题目服务配置
type ProblemServiceConf struct {
	// RPC配置（推荐）
	RPC struct {
		Enabled  bool   `json:",omitempty"` // 是否启用RPC调用
		Endpoint string `json:",omitempty"` // RPC服务地址
		Timeout  int    `json:",omitempty"` // RPC超时时间(毫秒)
	} `json:",omitempty"`

	// HTTP配置（兼容旧版本）
	HTTP struct {
		Endpoint   string `json:",omitempty"` // HTTP服务地址
		Timeout    int    `json:",omitempty"` // HTTP超时时间(秒)
		MaxRetries int    `json:",omitempty"` // 最大重试次数
		ApiKey     string `json:",omitempty"` // JWT令牌或API密钥
	} `json:",omitempty"`

	UseMock bool // 是否使用模拟客户端（开发环境）
}

// Consul配置
type ConsulConf struct {
	Enabled         bool     `json:"enabled" yaml:"enabled"`           // 是否启用Consul
	Address         string   `json:"address" yaml:"address"`           // Consul地址
	ServiceName     string   `json:"service_name" yaml:"service_name"`      // 服务名称
	ServiceID       string   `json:"service_id" yaml:"service_id"`        // 服务ID
	HealthCheckURL  string   `json:"health_check_url" yaml:"health_check_url"`  // 健康检查URL
	HealthInterval  string   `json:"health_interval" yaml:"health_interval"`   // 健康检查间隔
	HealthTimeout   string   `json:"health_timeout" yaml:"health_timeout"`    // 健康检查超时
	DeregisterAfter string   `json:"deregister_after" yaml:"deregister_after"`  // 失败后注销时间
	Tags            []string `json:"tags" yaml:"tags"`             // 服务标签
}
