package types

// 通用响应结构
type BaseResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// 分页信息
type Pagination struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
	Pages int   `json:"pages"`
}

// 用户信息
type UserInfo struct {
	UserId   int64  `json:"user_id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// 题目统计信息
type ProblemStats struct {
	TotalSubmissions    int     `json:"total_submissions"`
	AcceptedSubmissions int     `json:"accepted_submissions"`
	AcceptanceRate      float64 `json:"acceptance_rate"`
}

// 题目基本信息
type ProblemInfo struct {
	Id           int64        `json:"id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	InputFormat  string       `json:"input_format"`
	OutputFormat string       `json:"output_format"`
	SampleInput  string       `json:"sample_input"`
	SampleOutput string       `json:"sample_output"`
	Difficulty   string       `json:"difficulty"`
	TimeLimit    int          `json:"time_limit"`    // 毫秒
	MemoryLimit  int          `json:"memory_limit"`  // MB
	Languages    []string     `json:"languages"`
	Tags         []string     `json:"tags"`
	Author       UserInfo     `json:"author"`
	Statistics   ProblemStats `json:"statistics"`
	CreatedAt    string       `json:"created_at"`
	UpdatedAt    string       `json:"updated_at"`
}

// 题目列表项
type ProblemListItem struct {
	Id             int64    `json:"id"`
	Title          string   `json:"title"`
	Difficulty     string   `json:"difficulty"`
	Tags           []string `json:"tags"`
	AcceptanceRate float64  `json:"acceptance_rate"`
	CreatedAt      string   `json:"created_at"`
}

// ==================== 创建题目 ====================
type CreateProblemReq struct {
	Title        string   `json:"title" validate:"required,min=1,max=200"`
	Description  string   `json:"description" validate:"required,min=10"`
	InputFormat  string   `json:"input_format" validate:"required"`
	OutputFormat string   `json:"output_format" validate:"required"`
	SampleInput  string   `json:"sample_input" validate:"required"`
	SampleOutput string   `json:"sample_output" validate:"required"`
	Difficulty   string   `json:"difficulty" validate:"required,oneof=easy medium hard"`
	TimeLimit    int      `json:"time_limit" validate:"required,min=100,max=10000"`    // 100ms-10s
	MemoryLimit  int      `json:"memory_limit" validate:"required,min=16,max=512"`     // 16MB-512MB
	Languages    []string `json:"languages" validate:"required,min=1"`
	Tags         []string `json:"tags" validate:"max=10"`
	IsPublic     bool     `json:"is_public"`
}

type CreateProblemResp struct {
	BaseResp
	Data CreateProblemData `json:"data"`
}

type CreateProblemData struct {
	ProblemId int64  `json:"problem_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// ==================== 获取题目列表 ====================
type GetProblemListReq struct {
	Page       int    `form:"page,default=1" validate:"min=1"`
	Limit      int    `form:"limit,default=20" validate:"min=1,max=100"`
	Difficulty string `form:"difficulty,optional" validate:"omitempty,oneof=easy medium hard"`
	Tags       string `form:"tags,optional"`      // 逗号分隔的标签
	Keyword    string `form:"keyword,optional"`   // 搜索关键词
	SortBy     string `form:"sort_by,default=created_at" validate:"oneof=created_at title difficulty acceptance_rate"`
	Order      string `form:"order,default=desc" validate:"oneof=asc desc"`
}

type GetProblemListResp struct {
	BaseResp
	Data GetProblemListData `json:"data"`
}

type GetProblemListData struct {
	Problems   []ProblemListItem `json:"problems"`
	Pagination Pagination        `json:"pagination"`
}

// ==================== 获取题目详情 ====================
type GetProblemDetailReq struct {
	Id int64 `path:"id" validate:"required,min=1"`
}

type GetProblemDetailResp struct {
	BaseResp
	Data ProblemInfo `json:"data"`
}

// ==================== 更新题目 ====================
type UpdateProblemReq struct {
	Id           int64    `path:"id" validate:"required,min=1"`
	Title        string   `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description  string   `json:"description,omitempty" validate:"omitempty,min=10"`
	InputFormat  string   `json:"input_format,omitempty"`
	OutputFormat string   `json:"output_format,omitempty"`
	SampleInput  string   `json:"sample_input,omitempty"`
	SampleOutput string   `json:"sample_output,omitempty"`
	Difficulty   string   `json:"difficulty,omitempty" validate:"omitempty,oneof=easy medium hard"`
	TimeLimit    int      `json:"time_limit,omitempty" validate:"omitempty,min=100,max=10000"`
	MemoryLimit  int      `json:"memory_limit,omitempty" validate:"omitempty,min=16,max=512"`
	Languages    []string `json:"languages,omitempty" validate:"omitempty,min=1"`
	Tags         []string `json:"tags,omitempty" validate:"omitempty,max=10"`
	IsPublic     bool     `json:"is_public,omitempty"`
}

type UpdateProblemResp struct {
	BaseResp
	Data UpdateProblemData `json:"data"`
}

type UpdateProblemData struct {
	ProblemId int64  `json:"problem_id"`
	UpdatedAt string `json:"updated_at"`
	Message   string `json:"message"`
}

// ==================== 删除题目 ====================
type DeleteProblemReq struct {
	Id int64 `path:"id" validate:"required,min=1"`
}

type DeleteProblemResp struct {
	BaseResp
	Data DeleteProblemData `json:"data"`
}

type DeleteProblemData struct {
	ProblemId int64  `json:"problem_id"`
	DeletedAt string `json:"deleted_at"`
	Message   string `json:"message"`
}

// ==================== 健康检查 ====================
type HealthResp struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// ==================== 服务指标 ====================
type MetricsResp struct {
	RequestCount     int64   `json:"request_count"`
	ErrorCount       int64   `json:"error_count"`
	AvgResponseTime  float64 `json:"avg_response_time"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	DatabaseConnPool int     `json:"database_conn_pool"`
}