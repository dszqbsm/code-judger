package types

// 通用响应结构
type BaseResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ==================== 判题任务提交 ====================
type SubmitJudgeReq struct {
	SubmissionId int64  `json:"submission_id" validate:"required,min=1"`
	ProblemId    int64  `json:"problem_id" validate:"required,min=1"`
	UserId       int64  `json:"user_id" validate:"required,min=1"`
	Language     string `json:"language" validate:"required,oneof=cpp c java python go javascript"`
	Code         string `json:"code" validate:"required,min=1"`
	// 移除 TimeLimit、MemoryLimit、TestCases
	// 这些参数应该通过 ProblemId 从题目服务获取
}

type TestCase struct {
	CaseId         int    `json:"case_id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	TimeLimit      int    `json:"time_limit,omitempty"`   // 可选，覆盖全局时间限制
	MemoryLimit    int    `json:"memory_limit,omitempty"` // 可选，覆盖全局内存限制
}

// 题目信息（从题目服务获取）
type ProblemInfo struct {
	ProblemId   int64      `json:"problem_id"`
	Title       string     `json:"title"`
	TimeLimit   int        `json:"time_limit"`   // 毫秒
	MemoryLimit int        `json:"memory_limit"` // MB
	Languages   []string   `json:"languages"`    // 支持的编程语言
	TestCases   []TestCase `json:"test_cases"`
	IsPublic    bool       `json:"is_public"`
}

type SubmitJudgeResp struct {
	BaseResp
	Data SubmitJudgeData `json:"data"`
}

type SubmitJudgeData struct {
	SubmissionId  int64  `json:"submission_id"`
	Status        string `json:"status"`
	QueuePosition int    `json:"queue_position"`
	EstimatedTime int    `json:"estimated_time"` // 预计等待时间(秒)
}

// ==================== 判题结果查询 ====================
type GetJudgeResultReq struct {
	SubmissionId int64 `path:"submission_id" validate:"required,min=1"`
}

type GetJudgeResultResp struct {
	BaseResp
	Data JudgeResult `json:"data"`
}

type JudgeResult struct {
	SubmissionId int64            `json:"submission_id"`
	Status       string           `json:"status"`
	Score        int              `json:"score"`
	TimeUsed     int              `json:"time_used"`   // 最大时间使用(毫秒)
	MemoryUsed   int              `json:"memory_used"` // 最大内存使用(KB)
	CompileInfo  CompileInfo      `json:"compile_info"`
	TestCases    []TestCaseResult `json:"test_cases"`
	JudgeInfo    JudgeInfo        `json:"judge_info"`
}

type CompileInfo struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Time    int    `json:"time"` // 编译时间(毫秒)
}

type TestCaseResult struct {
	CaseId      int    `json:"case_id"`
	Status      string `json:"status"`
	TimeUsed    int    `json:"time_used"`   // 毫秒
	MemoryUsed  int    `json:"memory_used"` // KB
	Input       string `json:"input"`
	Output      string `json:"output"`
	Expected    string `json:"expected"`
	ErrorOutput string `json:"error_output,omitempty"`
}

type JudgeInfo struct {
	JudgeServer     string `json:"judge_server"`
	JudgeTime       string `json:"judge_time"`
	LanguageVersion string `json:"language_version"`
}

// ==================== 判题状态查询 ====================
type GetJudgeStatusReq struct {
	SubmissionId int64 `path:"submission_id" validate:"required,min=1"`
}

type GetJudgeStatusResp struct {
	BaseResp
	Data JudgeStatus `json:"data"`
}

type JudgeStatus struct {
	SubmissionId    int64  `json:"submission_id"`
	Status          string `json:"status"`
	Progress        int    `json:"progress"`          // 进度百分比
	CurrentTestCase int    `json:"current_test_case"` // 当前测试用例编号
	TotalTestCases  int    `json:"total_test_cases"`  // 总测试用例数
	Message         string `json:"message"`
}

// ==================== 取消判题任务 ====================
type CancelJudgeReq struct {
	SubmissionId int64 `path:"submission_id" validate:"required,min=1"`
}

type CancelJudgeResp struct {
	BaseResp
	Data CancelJudgeData `json:"data"`
}

type CancelJudgeData struct {
	SubmissionId int64  `json:"submission_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// ==================== 重新判题 ====================
type RejudgeReq struct {
	SubmissionId int64 `path:"submission_id" validate:"required,min=1"`
}

type RejudgeResp struct {
	BaseResp
	Data RejudgeData `json:"data"`
}

type RejudgeData struct {
	SubmissionId int64  `json:"submission_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// ==================== 判题节点状态 ====================
type GetJudgeNodesReq struct {
}

type GetJudgeNodesResp struct {
	BaseResp
	Data JudgeNodesData `json:"data"`
}

type JudgeNodesData struct {
	Nodes []JudgeNode `json:"nodes"`
	Total int         `json:"total"`
}

type JudgeNode struct {
	NodeId        string  `json:"node_id"`
	NodeName      string  `json:"node_name"`
	Status        string  `json:"status"` // online, offline, busy
	CpuUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	ActiveTasks   int     `json:"active_tasks"`
	TotalTasks    int     `json:"total_tasks"`
	LastHeartbeat string  `json:"last_heartbeat"`
}

// ==================== 判题队列状态 ====================
type GetJudgeQueueReq struct {
}

type GetJudgeQueueResp struct {
	BaseResp
	Data JudgeQueueData `json:"data"`
}

type JudgeQueueData struct {
	QueueLength    int         `json:"queue_length"`
	PendingTasks   int         `json:"pending_tasks"`
	RunningTasks   int         `json:"running_tasks"`
	CompletedTasks int         `json:"completed_tasks"`
	FailedTasks    int         `json:"failed_tasks"`
	QueueItems     []QueueItem `json:"queue_items"`
}

type QueueItem struct {
	SubmissionId  int64  `json:"submission_id"`
	UserId        int64  `json:"user_id"`
	ProblemId     int64  `json:"problem_id"`
	Language      string `json:"language"`
	Priority      int    `json:"priority"`
	QueueTime     string `json:"queue_time"`
	EstimatedTime int    `json:"estimated_time"`
}

// ==================== 健康检查 ====================
type HealthCheckReq struct {
}

type HealthCheckResp struct {
	BaseResp
	Data HealthData `json:"data"`
}

type HealthData struct {
	Status     string     `json:"status"`
	Timestamp  string     `json:"timestamp"`
	Version    string     `json:"version"`
	Uptime     int64      `json:"uptime"`
	SystemInfo SystemInfo `json:"system_info"`
}

type SystemInfo struct {
	CpuUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	DiskUsage      float64 `json:"disk_usage"`
	GoroutineCount int     `json:"goroutine_count"`
}

// ==================== 支持语言查询 ====================
type GetLanguagesReq struct {
}

type GetLanguagesResp struct {
	BaseResp
	Data LanguagesData `json:"data"`
}

type LanguagesData struct {
	Languages []LanguageConfig `json:"languages"`
}

type LanguageConfig struct {
	Name             string  `json:"name"`
	DisplayName      string  `json:"display_name"`
	Version          string  `json:"version"`
	FileExtension    string  `json:"file_extension"`
	CompileCommand   string  `json:"compile_command"`
	ExecuteCommand   string  `json:"execute_command"`
	TimeMultiplier   float64 `json:"time_multiplier"`
	MemoryMultiplier float64 `json:"memory_multiplier"`
	IsEnabled        bool    `json:"is_enabled"`
}
