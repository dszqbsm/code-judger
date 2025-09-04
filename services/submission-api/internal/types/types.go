package types

// ===========================================
// 基础类型定义
// ===========================================

type BaseResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ===========================================
// 提交相关类型定义
// ===========================================

// 重新判题请求
type RejudgeSubmissionReq struct {
	SubmissionID int64 `path:"id" validate:"required"`
}

// 重新判题响应
type RejudgeSubmissionResp struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	Data    RejudgeSubmissionRespData `json:"data"`
}

type RejudgeSubmissionRespData struct {
	SubmissionID  int64  `json:"submission_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	QueuePosition int    `json:"queue_position"`
	EstimatedTime int    `json:"estimated_time"`
}

// 创建提交请求
type CreateSubmissionReq struct {
	ProblemID int64  `json:"problem_id" validate:"required"`
	Language  string `json:"language" validate:"required,oneof=cpp c java python go javascript"`
	Code      string `json:"code" validate:"required,max=65536"`
	ContestID int64  `json:"contest_id,omitempty"`
	IsShared  bool   `json:"is_shared,omitempty"`
}

// 创建提交响应
type CreateSubmissionResp struct {
	Code    int                      `json:"code"`
	Message string                   `json:"message"`
	Data    CreateSubmissionRespData `json:"data"`
}

type CreateSubmissionRespData struct {
	SubmissionID  int64  `json:"submission_id"`
	Status        string `json:"status"`
	QueuePosition int    `json:"queue_position"`
	EstimatedTime int    `json:"estimated_time"`
	CreatedAt     string `json:"created_at"`
}

// 获取提交详情请求
type GetSubmissionReq struct {
	SubmissionID int64 `path:"submission_id" validate:"required"`
}

// 获取提交详情响应
type GetSubmissionResp struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    GetSubmissionRespData `json:"data"`
}

type GetSubmissionRespData struct {
	SubmissionID    int64             `json:"submission_id"`
	ProblemID       int64             `json:"problem_id"`
	UserID          int64             `json:"user_id"`
	Username        string            `json:"username"`
	Language        string            `json:"language"`
	Code            string            `json:"code"`
	CodeLength      int               `json:"code_length"`
	Status          string            `json:"status"`
	ContestID       *int64            `json:"contest_id,omitempty"`
	Score           *int32            `json:"score,omitempty"`
	TimeUsed        *int32            `json:"time_used,omitempty"`
	MemoryUsed      *int32            `json:"memory_used,omitempty"`
	CompileOutput   *string           `json:"compile_output,omitempty"`
	RuntimeOutput   *string           `json:"runtime_output,omitempty"`
	ErrorMessage    *string           `json:"error_message,omitempty"`
	TestCasesPassed *int32            `json:"test_cases_passed,omitempty"`
	TestCasesTotal  *int32            `json:"test_cases_total,omitempty"`
	JudgeServer     *string           `json:"judge_server,omitempty"`
	Result          *SubmissionResult `json:"result,omitempty"`
	CompileInfo     *CompileInfo      `json:"compile_info,omitempty"`
	CreatedAt       string            `json:"created_at"`
	JudgedAt        *string           `json:"judged_at,omitempty"`
	UpdatedAt       string            `json:"updated_at"`
}

// 判题结果
type SubmissionResult struct {
	Verdict    string           `json:"verdict"`
	Score      int              `json:"score"`
	TimeUsed   int              `json:"time_used"`
	MemoryUsed int              `json:"memory_used"`
	TestCases  []TestCaseResult `json:"test_cases"`
}

// 测试用例结果
type TestCaseResult struct {
	CaseID     int    `json:"case_id"`
	Status     string `json:"status"`
	TimeUsed   int    `json:"time_used"`
	MemoryUsed int    `json:"memory_used"`
	Input      string `json:"input,omitempty"`
	Output     string `json:"output,omitempty"`
	Expected   string `json:"expected,omitempty"`
}

// 编译信息
type CompileInfo struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Time    int    `json:"time"`
}

// 获取提交列表请求
type GetSubmissionListReq struct {
	Page      int    `form:"page,default=1" validate:"min=1"`
	PageSize  int    `form:"page_size,default=20" validate:"min=1,max=100"`
	ProblemID int64  `form:"problem_id,optional"`
	ContestID int64  `form:"contest_id,optional"`
	Status    string `form:"status,optional"`
	Language  string `form:"language,optional"`
	UserID    int64  `form:"user_id,optional"`
}

// 获取提交列表响应
type GetSubmissionListResp struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	Data    GetSubmissionListRespData `json:"data"`
}

type GetSubmissionListRespData struct {
	Submissions []SubmissionSummary `json:"submissions"`
	Total       int64               `json:"total"`
	Page        int                 `json:"page"`
	PageSize    int                 `json:"page_size"`
}

// 提交摘要信息
type SubmissionSummary struct {
	SubmissionID int64   `json:"submission_id"`
	ProblemID    int64   `json:"problem_id"`
	ProblemTitle string  `json:"problem_title"`
	UserID       int64   `json:"user_id"`
	Username     string  `json:"username"`
	Language     string  `json:"language"`
	Status       string  `json:"status"`
	ContestID    *int64  `json:"contest_id,omitempty"`
	Score        int     `json:"score"`
	TimeUsed     int     `json:"time_used"`
	MemoryUsed   int     `json:"memory_used"`
	CreatedAt    string  `json:"created_at"`
	JudgedAt     *string `json:"judged_at,omitempty"`
}

// 取消提交请求
type CancelSubmissionReq struct {
	SubmissionID int64 `path:"submission_id" validate:"required"`
}

// 获取队列统计请求
type GetQueueStatsReq struct {
	// 无需参数
}

// 获取队列统计响应
type GetQueueStatsResp struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    GetQueueStatsRespData `json:"data"`
}

type GetQueueStatsRespData struct {
	TotalTasks         int64   `json:"total_tasks"`          // 总任务数
	PendingTasks       int64   `json:"pending_tasks"`        // 等待中任务数
	ProcessingTasks    int64   `json:"processing_tasks"`     // 处理中任务数
	CompletedTasks     int64   `json:"completed_tasks"`      // 已完成任务数
	CurrentQueueLength int64   `json:"current_queue_length"` // 当前队列长度
	AverageWaitTime    float64 `json:"average_wait_time"`    // 平均等待时间
	AverageJudgeTime   float64 `json:"average_judge_time"`   // 平均判题时间
	ActiveJudges       int     `json:"active_judges"`        // 活跃判题服务器数量
}

// 高级搜索请求
type SearchSubmissionsReq struct {
	Query     string  `form:"query,omitempty"`
	UserID    *int64  `form:"user_id,omitempty"`
	ProblemID *int64  `form:"problem_id,omitempty"`
	Language  string  `form:"language,omitempty"`
	Status    string  `form:"status,omitempty"`
	ContestID *int64  `form:"contest_id,omitempty"`
	TimeFrom  *string `form:"time_from,omitempty"`
	TimeTo    *string `form:"time_to,omitempty"`
	Page      int     `form:"page,default=1" validate:"min=1"`
	PageSize  int     `form:"page_size,default=20" validate:"min=1,max=100"`
}

// 高级搜索响应
type SearchSubmissionsResp struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	Data    GetSubmissionListRespData `json:"data"`
}

// 获取用户提交统计请求
type GetUserSubmissionStatsReq struct {
	UserID int64 `path:"user_id" validate:"required"`
}

// 获取用户提交统计响应
type GetUserSubmissionStatsResp struct {
	Code    int                            `json:"code"`
	Message string                         `json:"message"`
	Data    GetUserSubmissionStatsRespData `json:"data"`
}

type GetUserSubmissionStatsRespData struct {
	UserID              int64                    `json:"user_id"`
	Username            string                   `json:"username"`
	TotalSubmissions    int64                    `json:"total_submissions"`
	AcceptedSubmissions int64                    `json:"accepted_submissions"`
	AcceptanceRate      float64                  `json:"acceptance_rate"`
	LanguageStats       []LanguageSubmissionStat `json:"language_stats"`
	StatusStats         []StatusSubmissionStat   `json:"status_stats"`
	RecentActivity      []DailySubmissionStat    `json:"recent_activity"`
}

// 语言提交统计
type LanguageSubmissionStat struct {
	Language       string  `json:"language"`
	Count          int64   `json:"count"`
	AcceptedCount  int64   `json:"accepted_count"`
	AcceptanceRate float64 `json:"acceptance_rate"`
}

// 状态提交统计
type StatusSubmissionStat struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// 每日提交统计
type DailySubmissionStat struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// 获取题目提交统计请求
type GetProblemSubmissionStatsReq struct {
	ProblemID int64 `path:"problem_id" validate:"required"`
}

// 获取题目提交统计响应
type GetProblemSubmissionStatsResp struct {
	Code    int                               `json:"code"`
	Message string                            `json:"message"`
	Data    GetProblemSubmissionStatsRespData `json:"data"`
}

type GetProblemSubmissionStatsRespData struct {
	ProblemID           int64                    `json:"problem_id"`
	ProblemTitle        string                   `json:"problem_title"`
	TotalSubmissions    int64                    `json:"total_submissions"`
	AcceptedSubmissions int64                    `json:"accepted_submissions"`
	AcceptanceRate      float64                  `json:"acceptance_rate"`
	LanguageStats       []LanguageSubmissionStat `json:"language_stats"`
	StatusStats         []StatusSubmissionStat   `json:"status_stats"`
	DifficultyLevel     string                   `json:"difficulty_level"`
	AvgTimeUsed         float64                  `json:"avg_time_used"`
	AvgMemoryUsed       float64                  `json:"avg_memory_used"`
}

// 比较提交请求
type CompareSubmissionsReq struct {
	SubmissionID1 int64 `json:"submission_id_1" validate:"required"`
	SubmissionID2 int64 `json:"submission_id_2" validate:"required"`
}

// 比较提交响应
type CompareSubmissionsResp struct {
	Code    int                        `json:"code"`
	Message string                     `json:"message"`
	Data    CompareSubmissionsRespData `json:"data"`
}

type CompareSubmissionsRespData struct {
	SubmissionID1   int64            `json:"submission_id_1"`
	SubmissionID2   int64            `json:"submission_id_2"`
	SimilarityScore float64          `json:"similarity_score"`
	Differences     []CodeDifference `json:"differences"`
}

// 代码差异
type CodeDifference struct {
	Type     string `json:"type"` // added, removed, modified
	LineNum1 int    `json:"line_num_1"`
	LineNum2 int    `json:"line_num_2"`
	Content1 string `json:"content_1"`
	Content2 string `json:"content_2"`
}

// ===========================================
// 内部接口类型定义
// ===========================================

// 更新提交状态请求
type UpdateSubmissionStatusReq struct {
	SubmissionID int64             `path:"submission_id" validate:"required"`
	Status       string            `json:"status" validate:"required"`
	Result       *SubmissionResult `json:"result,omitempty"`
	CompileInfo  *CompileInfo      `json:"compile_info,omitempty"`
	ErrorMessage *string           `json:"error_message,omitempty"`
	JudgedAt     *string           `json:"judged_at,omitempty"`
}

// 批量更新提交状态请求
type BatchUpdateSubmissionStatusReq struct {
	Updates []UpdateSubmissionStatusReq `json:"updates" validate:"required,dive"`
}

// ===========================================
// 管理员接口类型定义
// ===========================================

// 获取系统提交概览响应
type GetSubmissionOverviewResp struct {
	Code    int                           `json:"code"`
	Message string                        `json:"message"`
	Data    GetSubmissionOverviewRespData `json:"data"`
}

type GetSubmissionOverviewRespData struct {
	TotalSubmissions     int64                    `json:"total_submissions"`
	TodaySubmissions     int64                    `json:"today_submissions"`
	QueuedSubmissions    int64                    `json:"queued_submissions"`
	JudgingSubmissions   int64                    `json:"judging_submissions"`
	SystemLoad           SystemLoadInfo           `json:"system_load"`
	LanguageDistribution []LanguageSubmissionStat `json:"language_distribution"`
	StatusDistribution   []StatusSubmissionStat   `json:"status_distribution"`
	HourlyStats          []HourlySubmissionStat   `json:"hourly_stats"`
}

// 系统负载信息
type SystemLoadInfo struct {
	QueueLength      int     `json:"queue_length"`
	ActiveJudges     int     `json:"active_judges"`
	AvgWaitTime      float64 `json:"avg_wait_time"`
	AvgJudgeTime     float64 `json:"avg_judge_time"`
	ThroughputPerMin float64 `json:"throughput_per_min"`
}

// 每小时提交统计
type HourlySubmissionStat struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

// 获取异常提交请求
type GetAnomalousSubmissionsReq struct {
	Type     string `form:"type,omitempty"` // timeout, memory_exceed, compile_error, etc.
	TimeFrom string `form:"time_from,omitempty"`
	TimeTo   string `form:"time_to,omitempty"`
	Page     int    `form:"page,default=1" validate:"min=1"`
	PageSize int    `form:"page_size,default=20" validate:"min=1,max=100"`
}

// 获取异常提交响应
type GetAnomalousSubmissionsResp struct {
	Code    int                             `json:"code"`
	Message string                          `json:"message"`
	Data    GetAnomalousSubmissionsRespData `json:"data"`
}

type GetAnomalousSubmissionsRespData struct {
	Submissions []AnomalousSubmission `json:"submissions"`
	Total       int64                 `json:"total"`
	Page        int                   `json:"page"`
	PageSize    int                   `json:"page_size"`
}

// 异常提交信息
type AnomalousSubmission struct {
	SubmissionID int64  `json:"submission_id"`
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	ProblemID    int64  `json:"problem_id"`
	Language     string `json:"language"`
	Status       string `json:"status"`
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
	CreatedAt    string `json:"created_at"`
	JudgedAt     string `json:"judged_at"`
}

// 批量重新判题请求
type BatchRejudgeReq struct {
	SubmissionIDs []int64 `json:"submission_ids,omitempty"`
	ProblemID     *int64  `json:"problem_id,omitempty"`
	ContestID     *int64  `json:"contest_id,omitempty"`
	TimeFrom      *string `json:"time_from,omitempty"`
	TimeTo        *string `json:"time_to,omitempty"`
	Reason        string  `json:"reason" validate:"required"`
}

// 批量重新判题响应
type BatchRejudgeResp struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    BatchRejudgeRespData `json:"data"`
}

type BatchRejudgeRespData struct {
	TaskID        string `json:"task_id"`
	AffectedCount int64  `json:"affected_count"`
	EstimatedTime int    `json:"estimated_time"`
	Status        string `json:"status"`
}

// 导出提交请求
type ExportSubmissionsReq struct {
	Format    string  `form:"format,default=csv" validate:"oneof=csv json excel"`
	UserID    *int64  `form:"user_id,omitempty"`
	ProblemID *int64  `form:"problem_id,omitempty"`
	ContestID *int64  `form:"contest_id,omitempty"`
	TimeFrom  *string `form:"time_from,omitempty"`
	TimeTo    *string `form:"time_to,omitempty"`
	Status    string  `form:"status,omitempty"`
}

// 导出提交响应
type ExportSubmissionsResp struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	Data    ExportSubmissionsRespData `json:"data"`
}

type ExportSubmissionsRespData struct {
	DownloadURL string `json:"download_url"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	ExpiresAt   string `json:"expires_at"`
}

// 检测代码抄袭请求
type DetectPlagiarismReq struct {
	ProblemID     *int64  `json:"problem_id,omitempty"`
	ContestID     *int64  `json:"contest_id,omitempty"`
	SubmissionIDs []int64 `json:"submission_ids,omitempty"`
	Threshold     float64 `json:"threshold" validate:"min=0,max=1"`
}

// 检测代码抄袭响应
type DetectPlagiarismResp struct {
	Code    int                      `json:"code"`
	Message string                   `json:"message"`
	Data    DetectPlagiarismRespData `json:"data"`
}

type DetectPlagiarismRespData struct {
	TaskID          string             `json:"task_id"`
	SimilarPairs    []SimilarityResult `json:"similar_pairs"`
	TotalPairs      int                `json:"total_pairs"`
	SuspiciousPairs int                `json:"suspicious_pairs"`
}

// 相似度结果
type SimilarityResult struct {
	SubmissionID1   int64    `json:"submission_id_1"`
	SubmissionID2   int64    `json:"submission_id_2"`
	UserID1         int64    `json:"user_id_1"`
	UserID2         int64    `json:"user_id_2"`
	Username1       string   `json:"username_1"`
	Username2       string   `json:"username_2"`
	SimilarityScore float64  `json:"similarity_score"`
	MatchedFeatures []string `json:"matched_features"`
	Confidence      float64  `json:"confidence"`
}

// ===========================================
// 判题相关代理接口类型定义
// ===========================================

// 获取提交判题结果请求
type GetSubmissionJudgeResultReq struct {
	SubmissionID int64 `path:"submission_id" validate:"required"`
}

// 获取提交判题结果响应
type GetSubmissionJudgeResultResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    JudgeResult `json:"data"`
}

type JudgeResult struct {
	SubmissionId int64            `json:"submission_id"`
	Status       string           `json:"status"`
	Score        int              `json:"score"`
	TimeUsed     int              `json:"time_used"`
	MemoryUsed   int              `json:"memory_used"`
	CompileInfo  CompileInfo      `json:"compile_info"`
	TestCases    []TestCaseResult `json:"test_cases"`
	JudgeInfo    JudgeInfo        `json:"judge_info"`
}

type JudgeInfo struct {
	JudgeServer     string `json:"judge_server"`
	JudgeTime       string `json:"judge_time"`
	LanguageVersion string `json:"language_version"`
}

// 获取提交判题状态请求
type GetSubmissionJudgeStatusReq struct {
	SubmissionID int64 `path:"submission_id" validate:"required"`
}

// 获取提交判题状态响应
type GetSubmissionJudgeStatusResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    JudgeStatus `json:"data"`
}

type JudgeStatus struct {
	SubmissionId    int64  `json:"submission_id"`
	Status          string `json:"status"`
	Progress        int    `json:"progress"`
	CurrentTestCase int    `json:"current_test_case"`
	TotalTestCases  int    `json:"total_test_cases"`
	Message         string `json:"message"`
}

// 重新判题请求（代理版本）
type RejudgeSubmissionProxyReq struct {
	SubmissionID int64 `path:"submission_id" validate:"required"`
}

// 重新判题响应（代理版本）
type RejudgeSubmissionProxyResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    RejudgeData `json:"data"`
}

type RejudgeData struct {
	SubmissionId int64  `json:"submission_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// 获取判题队列状态响应
type GetJudgeQueueStatusResp struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    JudgeQueueData `json:"data"`
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
