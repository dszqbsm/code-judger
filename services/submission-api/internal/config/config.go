package config

import (
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	// 数据库配置
	DataSource string

	// Redis配置
	RedisConf redis.RedisConf

	// 缓存配置
	CacheConf cache.CacheConf

	// Kafka配置
	KafkaConf KafkaConf

	// JWT配置
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	// 业务配置
	Business BusinessConf

	// WebSocket配置
	WebSocket WebSocketConf

	// 提交配置
	Submission SubmissionConf

	// 查重配置
	AntiCheat AntiCheatConf

	// 监控配置
	Monitor MonitorConf `json:",omitempty"`

	// 判题服务配置
	JudgeService JudgeServiceConf

	// Consul配置
	Consul ConsulConf

	// RPC配置
	RPC RPCConf
}

// Kafka配置
type KafkaConf struct {
	Brokers []string   `json:"brokers"`
	Topics  TopicConf  `json:"topics"`
	Groups  GroupConf  `json:"groups"`
}

// 主题配置
type TopicConf struct {
	JudgeTask      string `json:"judge_task"`
	JudgeResult    string `json:"judge_result"`
	StatusUpdate   string `json:"status_update"`
	Notification   string `json:"notification"`
	DeadLetter     string `json:"dead_letter"`
}

// 消费者组配置
type GroupConf struct {
	SubmissionResult string `json:"submission_result"`
	StatusUpdate     string `json:"status_update"`
}

// 业务配置
type BusinessConf struct {
	// 分页配置
	DefaultPageSize int `json:"default_page_size"`
	MaxPageSize     int `json:"max_page_size"`

	// 代码限制
	MaxCodeLength          int `json:"max_code_length"`
	MaxSubmissionPerMinute int `json:"max_submission_per_minute"`

	// 判题配置
	AverageJudgeTime int `json:"average_judge_time"` // 平均判题时间（秒）
	ConcurrentJudges int `json:"concurrent_judges"`  // 并发判题服务器数量

	// 文件上传
	MaxFileSize      int64    `json:"max_file_size"`
	AllowedFileTypes []string `json:"allowed_file_types"`
}

// WebSocket配置
type WebSocketConf struct {
	ReadTimeout       int `json:"read_timeout"`       // 读取超时时间(秒)
	WriteTimeout      int `json:"write_timeout"`      // 写入超时时间(秒)
	HeartbeatInterval int `json:"heartbeat_interval"` // 心跳间隔(秒)
	MaxConnections    int `json:"max_connections"`    // 最大连接数
	BufferSize        int `json:"buffer_size"`        // 缓冲区大小
}

// 提交配置
type SubmissionConf struct {
	// 支持的编程语言
	SupportedLanguages []LanguageConf `json:"supported_languages"`

	// 队列配置
	QueueSize     int `json:"queue_size"`
	MaxRetries    int `json:"max_retries"`
	RetryInterval int `json:"retry_interval"`

	// 统计配置
	StatsRetentionDays int `json:"stats_retention_days"`

	// 导出配置
	ExportMaxRows     int `json:"export_max_rows"`
	ExportExpireHours int `json:"export_expire_hours"`
}

// 语言配置
type LanguageConf struct {
	Name             string  `json:"name"`
	DisplayName      string  `json:"display_name"`
	FileExtension    string  `json:"file_extension"`
	TimeMultiplier   float64 `json:"time_multiplier"`
	MemoryMultiplier float64 `json:"memory_multiplier"`
	Enabled          bool    `json:"enabled"`
}

// 查重配置
type AntiCheatConf struct {
	Enabled             bool    `json:"enabled"`
	SimilarityThreshold float64 `json:"similarity_threshold"`
	BatchSize           int     `json:"batch_size"`
	CheckInterval       int     `json:"check_interval"`

	// 检测算法配置
	Algorithms AlgorithmConf `json:"algorithms"`

	// 机器学习模型配置
	MLModel MLModelConf `json:"ml_model"`
}

// 算法配置
type AlgorithmConf struct {
	EnableStringMatch  bool    `json:"enable_string_match"`
	EnableASTMatch     bool    `json:"enable_ast_match"`
	EnableFeatureMatch bool    `json:"enable_feature_match"`
	StringWeight       float64 `json:"string_weight"`
	ASTWeight          float64 `json:"ast_weight"`
	FeatureWeight      float64 `json:"feature_weight"`
}

// 机器学习模型配置
type MLModelConf struct {
	Enabled   bool    `json:"enabled"`
	ModelPath string  `json:"model_path"`
	Threshold float64 `json:"threshold"`
}

// 监控配置
type MonitorConf struct {
	EnableMetrics     bool   `json:"enable_metrics"`
	MetricsInterval   int    `json:"metrics_interval"`
	EnableTracing     bool   `json:"enable_tracing"`
	TracingEndpoint   string `json:"tracing_endpoint"`
	EnableHealthCheck bool   `json:"enable_health_check"`
}

// 判题服务配置
type JudgeServiceConf struct {
	Endpoint string `json:"endpoint"` // 判题服务地址
	Timeout  int    `json:"timeout"`  // 超时时间(秒)
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

// RPC配置
type RPCConf struct {
	Enabled        bool          `json:"enabled" yaml:"enabled"`         // 是否启用RPC
	DefaultTimeout time.Duration `json:"default_timeout" yaml:"default_timeout"` // 默认超时时间
	MaxRetries     int           `json:"max_retries" yaml:"max_retries"`     // 最大重试次数
	RetryDelay     time.Duration `json:"retry_delay" yaml:"retry_delay"`     // 重试延迟
}
