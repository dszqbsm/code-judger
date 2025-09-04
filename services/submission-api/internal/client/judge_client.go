package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// 判题服务客户端接口
type JudgeServiceClient interface {
	GetJudgeResult(ctx context.Context, submissionID int64) (*JudgeResultResp, error)
	GetJudgeStatus(ctx context.Context, submissionID int64) (*JudgeStatusResp, error)
	CancelJudge(ctx context.Context, submissionID int64) (*CancelJudgeResp, error)
	RejudgeSubmission(ctx context.Context, submissionID int64) (*RejudgeResp, error)
	GetJudgeQueue(ctx context.Context) (*JudgeQueueResp, error)
}

// HTTP客户端实现
type HttpJudgeClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

func NewHttpJudgeClient(baseURL string, timeout time.Duration) JudgeServiceClient {
	return &HttpJudgeClient{
		baseURL: baseURL,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

// 判题结果响应
type JudgeResultResp struct {
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

type CompileInfo struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Time    int    `json:"time"`
}

type TestCaseResult struct {
	CaseId      int    `json:"case_id"`
	Status      string `json:"status"`
	TimeUsed    int    `json:"time_used"`
	MemoryUsed  int    `json:"memory_used"`
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

// 判题状态响应
type JudgeStatusResp struct {
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

// 取消判题响应
type CancelJudgeResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    CancelJudgeData `json:"data"`
}

type CancelJudgeData struct {
	SubmissionId int64  `json:"submission_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// 重新判题响应
type RejudgeResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    RejudgeData `json:"data"`
}

type RejudgeData struct {
	SubmissionId int64  `json:"submission_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// 队列状态响应
type JudgeQueueResp struct {
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

// 获取判题结果
func (c *HttpJudgeClient) GetJudgeResult(ctx context.Context, submissionID int64) (*JudgeResultResp, error) {
	url := fmt.Sprintf("%s/api/v1/judge/result/%d", c.baseURL, submissionID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "submission-api/1.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logx.WithContext(ctx).Errorf("调用判题服务失败: %v", err)
		return nil, fmt.Errorf("调用判题服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result JudgeResultResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// 获取判题状态
func (c *HttpJudgeClient) GetJudgeStatus(ctx context.Context, submissionID int64) (*JudgeStatusResp, error) {
	url := fmt.Sprintf("%s/api/v1/judge/status/%d", c.baseURL, submissionID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "submission-api/1.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logx.WithContext(ctx).Errorf("调用判题服务失败: %v", err)
		return nil, fmt.Errorf("调用判题服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result JudgeStatusResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// 取消判题
func (c *HttpJudgeClient) CancelJudge(ctx context.Context, submissionID int64) (*CancelJudgeResp, error) {
	url := fmt.Sprintf("%s/api/v1/judge/cancel/%d", c.baseURL, submissionID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "submission-api/1.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logx.WithContext(ctx).Errorf("调用判题服务失败: %v", err)
		return nil, fmt.Errorf("调用判题服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result CancelJudgeResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// 重新判题
func (c *HttpJudgeClient) RejudgeSubmission(ctx context.Context, submissionID int64) (*RejudgeResp, error) {
	url := fmt.Sprintf("%s/api/v1/judge/rejudge/%d", c.baseURL, submissionID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "submission-api/1.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logx.WithContext(ctx).Errorf("调用判题服务失败: %v", err)
		return nil, fmt.Errorf("调用判题服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result RejudgeResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// 获取队列状态
func (c *HttpJudgeClient) GetJudgeQueue(ctx context.Context) (*JudgeQueueResp, error) {
	url := fmt.Sprintf("%s/api/v1/judge/queue", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "submission-api/1.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logx.WithContext(ctx).Errorf("调用判题服务失败: %v", err)
		return nil, fmt.Errorf("调用判题服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result JudgeQueueResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}




