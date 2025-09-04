package config

// Config 题目服务配置结构
// 注意：当前服务使用自定义HTTP服务器，此配置暂未使用
// 保留用于未来可能的go-zero框架迁移
type Config struct {
	// 服务基本配置
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Timeout  int64  `json:"timeout"`
	MaxConns int    `json:"max_conns"`
	MaxBytes int64  `json:"max_bytes"`

	// 数据库配置
	DataSource string `json:"data_source"`

	// JWT认证配置
	Auth struct {
		AccessSecret string `json:"access_secret"`
		AccessExpire int64  `json:"access_expire"`
	} `json:"auth"`

	// 服务注册配置
	Consul struct {
		Host string `json:"host"`
		Key  string `json:"key"`
	} `json:"consul"`

	// 业务配置
	Business struct {
		// 分页配置
		DefaultPageSize int `json:"default_page_size"`
		MaxPageSize     int `json:"max_page_size"`

		// 缓存配置
		ProblemListCacheTTL   int `json:"problem_list_cache_ttl"`   // 题目列表缓存5分钟
		ProblemDetailCacheTTL int `json:"problem_detail_cache_ttl"` // 题目详情缓存30分钟

		// 文件上传配置
		MaxFileSize int64 `json:"max_file_size"` // 10MB
	} `json:"business"`
}
