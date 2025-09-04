package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	"github.com/dszqbsm/code-judger/common/utils"
	"github.com/dszqbsm/code-judger/services/problem-api/models"
)

type ProblemAPI struct {
	db            *sql.DB
	problemModel  models.ProblemModel
	testCaseModel models.TestCaseModel
	jwtManager    *utils.JWTManager
	internalAPIKey string      // 内部API密钥
	allowedIPs     []string    // 允许的IP白名单
}

type BaseResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	// 连接数据库
	db, err := sql.Open("mysql", "oj_user:oj_password@tcp(localhost:3306)/oj_problems?charset=utf8mb4&parseTime=true&loc=Local")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试数据库连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// 初始化JWT管理器
	jwtManager := utils.NewJWTManager(
		"oj-access-secret-key-2024", // 与用户服务保持一致的密钥
		"",                          // 题目服务不需要刷新令牌
		3600,                        // 1小时过期
		0,
	)

	api := &ProblemAPI{
		db:            db,
		problemModel:  models.NewProblemModel(db),
		testCaseModel: models.NewTestCaseModel(db),
		jwtManager:    jwtManager,
		internalAPIKey: "internal-service-secret-key-2024", // 生产环境从环境变量读取
		allowedIPs:    []string{"127.0.0.1", "::1", "172.17.0.0/16", "10.0.0.0/8"}, // Docker网络和本地
	}

	// 创建路由
	r := mux.NewRouter()
	api.setupRoutes(r)

	// 启动服务器
	fmt.Println("Problem API server starting on :8891...")
	log.Fatal(http.ListenAndServe(":8891", r))
}

func (api *ProblemAPI) setupRoutes(r *mux.Router) {
	// 健康检查（无需认证）
	r.HandleFunc("/api/v1/health", api.healthCheck).Methods("GET")

	// 题目管理接口（需要JWT认证）
	r.HandleFunc("/api/v1/problems", api.jwtMiddleware(api.createProblem)).Methods("POST")
	r.HandleFunc("/api/v1/problems", api.jwtMiddleware(api.getProblemList)).Methods("GET")
	r.HandleFunc("/api/v1/problems/{id}", api.jwtMiddleware(api.getProblemDetail)).Methods("GET")
	r.HandleFunc("/api/v1/problems/{id}", api.jwtMiddleware(api.updateProblem)).Methods("PUT")
	r.HandleFunc("/api/v1/problems/{id}", api.jwtMiddleware(api.deleteProblem)).Methods("DELETE")

	// 测试用例管理接口（需要JWT认证）
	r.HandleFunc("/api/v1/problems/{id}/test-cases", api.jwtMiddleware(api.uploadTestCases)).Methods("POST")
	r.HandleFunc("/api/v1/problems/{id}/test-cases", api.jwtMiddleware(api.getTestCases)).Methods("GET")
	r.HandleFunc("/api/v1/test-cases/{id}", api.jwtMiddleware(api.getTestCaseDetail)).Methods("GET")
	r.HandleFunc("/api/v1/test-cases/{id}", api.jwtMiddleware(api.updateTestCase)).Methods("PUT")
	r.HandleFunc("/api/v1/test-cases/{id}", api.jwtMiddleware(api.deleteTestCase)).Methods("DELETE")

	// 内部接口（供判题服务调用，需要内部认证）
	r.HandleFunc("/internal/v1/problems/{id}", api.internalAuthMiddleware(api.getProblemDetailForJudge)).Methods("GET")
	r.HandleFunc("/internal/v1/problems/{id}/test-cases", api.internalAuthMiddleware(api.getTestCasesForJudge)).Methods("GET")
}

// JWT认证中间件
func (api *ProblemAPI) jwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取Authorization头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api.writeError(w, http.StatusUnauthorized, "缺少认证信息")
			return
		}

		// 检查Bearer前缀
		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		} else {
			api.writeError(w, http.StatusUnauthorized, "认证格式错误")
			return
		}

		// 验证JWT token
		_, err := api.jwtManager.ParseAccessToken(tokenString)
		if err != nil {
			api.writeError(w, http.StatusUnauthorized, "无效的认证令牌")
			return
		}

		// 认证通过，继续处理请求
		next(w, r)
	}
}

// 内部服务认证中间件
func (api *ProblemAPI) internalAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 检查IP白名单
		clientIP := api.getClientIP(r)
		if !api.isIPAllowed(clientIP) {
			log.Printf("Unauthorized internal API access from IP: %s", clientIP)
			api.writeError(w, http.StatusForbidden, "访问被拒绝：IP地址未授权")
			return
		}

		// 2. 检查API密钥
		apiKey := r.Header.Get("X-Internal-API-Key")
		if apiKey == "" {
			// 兼容旧的User-Agent方式，但记录警告
			userAgent := r.Header.Get("User-Agent")
			if !strings.Contains(userAgent, "judge-service") && !strings.Contains(userAgent, "judge-api") {
				log.Printf("Missing internal API key from IP: %s, User-Agent: %s", clientIP, userAgent)
				api.writeError(w, http.StatusUnauthorized, "缺少内部API密钥")
				return
			}
			log.Printf("Warning: Using deprecated User-Agent auth from IP: %s", clientIP)
		} else if apiKey != api.internalAPIKey {
			log.Printf("Invalid internal API key from IP: %s", clientIP)
			api.writeError(w, http.StatusUnauthorized, "无效的内部API密钥")
			return
		}

		// 3. 检查请求频率（简单的速率限制）
		// TODO: 实现基于Redis的分布式速率限制

		// 4. 记录访问日志
		log.Printf("Internal API access: IP=%s, Path=%s, User-Agent=%s", 
			clientIP, r.URL.Path, r.Header.Get("User-Agent"))

		// 认证通过，继续处理请求
		next(w, r)
	}
}

// 获取客户端真实IP
func (api *ProblemAPI) getClientIP(r *http.Request) string {
	// 优先检查代理头
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// 使用连接的远程地址
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// 检查IP是否在白名单中
func (api *ProblemAPI) isIPAllowed(clientIP string) bool {
	for _, allowedIP := range api.allowedIPs {
		if strings.Contains(allowedIP, "/") {
			// CIDR格式
			_, network, err := net.ParseCIDR(allowedIP)
			if err != nil {
				continue
			}
			if network.Contains(net.ParseIP(clientIP)) {
				return true
			}
		} else {
			// 直接IP比较
			if clientIP == allowedIP {
				return true
			}
		}
	}
	return false
}

func (api *ProblemAPI) healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format("2006-01-02T15:04:05Z07:00"),
		"version":   "v1.0.0",
	}
	api.writeJSON(w, http.StatusOK, response)
}

func (api *ProblemAPI) createProblem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		InputFormat  string   `json:"input_format"`
		OutputFormat string   `json:"output_format"`
		SampleInput  string   `json:"sample_input"`
		SampleOutput string   `json:"sample_output"`
		Difficulty   string   `json:"difficulty"`
		TimeLimit    int      `json:"time_limit"`
		MemoryLimit  int      `json:"memory_limit"`
		Languages    []string `json:"languages"`
		Tags         []string `json:"tags"`
		IsPublic     bool     `json:"is_public"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// 验证必填字段
	if req.Title == "" || req.Description == "" {
		api.writeError(w, http.StatusBadRequest, "Title and description are required")
		return
	}

	// 创建题目对象
	problem := &models.Problem{
		Title:       req.Title,
		Description: req.Description,
		Difficulty:  req.Difficulty,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		CreatedBy:   1, // 临时硬编码
		IsPublic:    req.IsPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.InputFormat != "" {
		problem.InputFormat = sql.NullString{String: req.InputFormat, Valid: true}
	}
	if req.OutputFormat != "" {
		problem.OutputFormat = sql.NullString{String: req.OutputFormat, Valid: true}
	}
	if req.SampleInput != "" {
		problem.SampleInput = sql.NullString{String: req.SampleInput, Valid: true}
	}
	if req.SampleOutput != "" {
		problem.SampleOutput = sql.NullString{String: req.SampleOutput, Valid: true}
	}

	if len(req.Languages) > 0 {
		languagesJSON, _ := json.Marshal(req.Languages)
		problem.Languages = sql.NullString{String: string(languagesJSON), Valid: true}
	}

	if len(req.Tags) > 0 {
		tagsJSON, _ := json.Marshal(req.Tags)
		problem.Tags = sql.NullString{String: string(tagsJSON), Valid: true}
	}

	result, err := api.problemModel.Insert(r.Context(), problem)
	if err != nil {
		log.Printf("Failed to create problem: %v", err)
		api.writeError(w, http.StatusInternalServerError, "Failed to create problem")
		return
	}

	problemId, _ := result.LastInsertId()

	response := BaseResp{
		Code:    200,
		Message: "题目创建成功",
		Data: map[string]interface{}{
			"problem_id": problemId,
			"title":      req.Title,
			"status":     "draft",
			"created_at": time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	api.writeJSON(w, http.StatusCreated, response)
}

func (api *ProblemAPI) getProblemList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	difficulty := r.URL.Query().Get("difficulty")
	keyword := r.URL.Query().Get("keyword")

	filters := &models.ProblemFilters{
		Difficulty: difficulty,
		Keyword:    keyword,
		SortBy:     "created_at",
		Order:      "desc",
	}

	isPublic := true
	filters.IsPublic = &isPublic

	problems, total, err := api.problemModel.FindByPage(r.Context(), page, limit, filters)
	if err != nil {
		log.Printf("Failed to get problem list: %v", err)
		api.writeError(w, http.StatusInternalServerError, "Failed to get problem list")
		return
	}

	// 转换为响应格式
	var problemList []map[string]interface{}
	for _, problem := range problems {
		var tags []string
		if problem.Tags.Valid {
			json.Unmarshal([]byte(problem.Tags.String), &tags)
		}

		item := map[string]interface{}{
			"id":              problem.Id,
			"title":           problem.Title,
			"difficulty":      problem.Difficulty,
			"tags":            tags,
			"acceptance_rate": problem.AcceptanceRate,
			"created_at":      problem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		problemList = append(problemList, item)
	}

	totalPages := (int(total) + limit - 1) / limit
	pagination := map[string]interface{}{
		"page":  page,
		"limit": limit,
		"total": total,
		"pages": totalPages,
	}

	response := BaseResp{
		Code:    200,
		Message: "获取成功",
		Data: map[string]interface{}{
			"problems":   problemList,
			"pagination": pagination,
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *ProblemAPI) getProblemDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid problem ID")
		return
	}

	problem, err := api.problemModel.FindOne(r.Context(), id)
	if err != nil {
		log.Printf("Failed to get problem detail: %v", err)
		api.writeError(w, http.StatusNotFound, "Problem not found")
		return
	}

	if problem.DeletedAt.Valid {
		api.writeError(w, http.StatusNotFound, "Problem has been deleted")
		return
	}

	// 解析JSON字段
	var languages, tags []string
	if problem.Languages.Valid {
		json.Unmarshal([]byte(problem.Languages.String), &languages)
	}
	if problem.Tags.Valid {
		json.Unmarshal([]byte(problem.Tags.String), &tags)
	}

	problemInfo := map[string]interface{}{
		"id":            problem.Id,
		"title":         problem.Title,
		"description":   problem.Description,
		"input_format":  problem.InputFormat.String,
		"output_format": problem.OutputFormat.String,
		"sample_input":  problem.SampleInput.String,
		"sample_output": problem.SampleOutput.String,
		"difficulty":    problem.Difficulty,
		"time_limit":    problem.TimeLimit,
		"memory_limit":  problem.MemoryLimit,
		"languages":     languages,
		"tags":          tags,
		"author": map[string]interface{}{
			"user_id":  problem.CreatedBy,
			"username": fmt.Sprintf("user%d", problem.CreatedBy),
			"name":     fmt.Sprintf("用户%d", problem.CreatedBy),
		},
		"statistics": map[string]interface{}{
			"total_submissions":    problem.SubmissionCount,
			"accepted_submissions": problem.AcceptedCount,
			"acceptance_rate":      problem.AcceptanceRate,
		},
		"created_at": problem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at": problem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	response := BaseResp{
		Code:    200,
		Message: "获取成功",
		Data:    problemInfo,
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *ProblemAPI) updateProblem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid problem ID")
		return
	}

	// 获取现有题目
	existing, err := api.problemModel.FindOne(r.Context(), id)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "Problem not found")
		return
	}

	if existing.DeletedAt.Valid {
		api.writeError(w, http.StatusNotFound, "Problem has been deleted")
		return
	}

	var req struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		InputFormat  string   `json:"input_format"`
		OutputFormat string   `json:"output_format"`
		SampleInput  string   `json:"sample_input"`
		SampleOutput string   `json:"sample_output"`
		Difficulty   string   `json:"difficulty"`
		TimeLimit    int      `json:"time_limit"`
		MemoryLimit  int      `json:"memory_limit"`
		Languages    []string `json:"languages"`
		Tags         []string `json:"tags"`
		IsPublic     bool     `json:"is_public"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// 更新字段
	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Difficulty != "" {
		existing.Difficulty = req.Difficulty
	}
	if req.TimeLimit > 0 {
		existing.TimeLimit = req.TimeLimit
	}
	if req.MemoryLimit > 0 {
		existing.MemoryLimit = req.MemoryLimit
	}

	existing.IsPublic = req.IsPublic
	existing.UpdatedAt = time.Now()

	if len(req.Languages) > 0 {
		languagesJSON, _ := json.Marshal(req.Languages)
		existing.Languages = sql.NullString{String: string(languagesJSON), Valid: true}
	}

	if len(req.Tags) > 0 {
		tagsJSON, _ := json.Marshal(req.Tags)
		existing.Tags = sql.NullString{String: string(tagsJSON), Valid: true}
	}

	err = api.problemModel.Update(r.Context(), existing)
	if err != nil {
		log.Printf("Failed to update problem: %v", err)
		api.writeError(w, http.StatusInternalServerError, "Failed to update problem")
		return
	}

	response := BaseResp{
		Code:    200,
		Message: "题目更新成功",
		Data: map[string]interface{}{
			"problem_id": id,
			"updated_at": existing.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"message":    "题目信息已更新",
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *ProblemAPI) deleteProblem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid problem ID")
		return
	}

	// 检查题目是否存在
	existing, err := api.problemModel.FindOne(r.Context(), id)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "Problem not found")
		return
	}

	if existing.DeletedAt.Valid {
		api.writeError(w, http.StatusNotFound, "Problem already deleted")
		return
	}

	// 执行软删除
	err = api.problemModel.SoftDelete(r.Context(), id)
	if err != nil {
		log.Printf("Failed to delete problem: %v", err)
		api.writeError(w, http.StatusInternalServerError, "Failed to delete problem")
		return
	}

	response := BaseResp{
		Code:    200,
		Message: "题目删除成功",
		Data: map[string]interface{}{
			"problem_id": id,
			"deleted_at": time.Now().Format("2006-01-02T15:04:05Z07:00"),
			"message":    "题目已被标记为删除状态",
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *ProblemAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *ProblemAPI) writeError(w http.ResponseWriter, status int, message string) {
	response := BaseResp{
		Code:    status,
		Message: message,
	}
	api.writeJSON(w, status, response)
}

// ================== 测试用例管理接口 ==================

// uploadTestCases 上传测试用例
func (api *ProblemAPI) uploadTestCases(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	problemId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的题目ID")
		return
	}

	// 检查题目是否存在
	problem, err := api.problemModel.FindOne(r.Context(), problemId)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "题目不存在")
		return
	}

	if problem.DeletedAt.Valid {
		api.writeError(w, http.StatusNotFound, "题目已删除")
		return
	}

	var req models.TestCaseUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的JSON格式")
		return
	}

	// 验证请求数据
	if len(req.TestCases) == 0 {
		api.writeError(w, http.StatusBadRequest, "测试用例不能为空")
		return
	}

	// 验证每个测试用例
	for i, testCase := range req.TestCases {
		if testCase.InputData == "" {
			api.writeError(w, http.StatusBadRequest, fmt.Sprintf("第%d个测试用例的输入数据不能为空", i+1))
			return
		}
		if testCase.ExpectedOutput == "" {
			api.writeError(w, http.StatusBadRequest, fmt.Sprintf("第%d个测试用例的期望输出不能为空", i+1))
			return
		}
		if testCase.Score <= 0 {
			req.TestCases[i].Score = 10 // 默认分值
		}
	}

	// 如果选择替换所有测试用例，先删除现有的
	if req.ReplaceAll {
		err = api.testCaseModel.DeleteByProblemId(r.Context(), problemId)
		if err != nil {
			log.Printf("Failed to delete existing test cases: %v", err)
			api.writeError(w, http.StatusInternalServerError, "删除现有测试用例失败")
			return
		}
	}

	// 准备测试用例数据
	var testCases []*models.TestCase
	for i, reqTestCase := range req.TestCases {
		testCase := &models.TestCase{
			ProblemId:      problemId,
			InputData:      reqTestCase.InputData,
			ExpectedOutput: reqTestCase.ExpectedOutput,
			IsSample:       reqTestCase.IsSample,
			Score:          reqTestCase.Score,
			SortOrder:      reqTestCase.SortOrder,
		}
		if testCase.SortOrder == 0 {
			testCase.SortOrder = i + 1 // 默认排序
		}
		testCases = append(testCases, testCase)
	}

	// 批量插入测试用例
	err = api.testCaseModel.BatchInsert(r.Context(), testCases)
	if err != nil {
		log.Printf("Failed to insert test cases: %v", err)
		api.writeError(w, http.StatusInternalServerError, "插入测试用例失败")
		return
	}

	// 统计测试用例数量
	count, err := api.testCaseModel.CountByProblemId(r.Context(), problemId)
	if err != nil {
		count = int64(len(req.TestCases))
	}

	response := BaseResp{
		Code:    200,
		Message: "测试用例上传成功",
		Data: map[string]interface{}{
			"problem_id":       problemId,
			"uploaded_count":   len(req.TestCases),
			"total_count":      count,
			"replaced_all":     req.ReplaceAll,
			"uploaded_at":      time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	api.writeJSON(w, http.StatusCreated, response)
}

// getTestCases 获取题目的测试用例列表
func (api *ProblemAPI) getTestCases(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	problemId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的题目ID")
		return
	}

	// 检查题目是否存在
	problem, err := api.problemModel.FindOne(r.Context(), problemId)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "题目不存在")
		return
	}

	if problem.DeletedAt.Valid {
		api.writeError(w, http.StatusNotFound, "题目已删除")
		return
	}

	// 查询参数
	includeData := r.URL.Query().Get("include_data") == "true"
	onlySamples := r.URL.Query().Get("only_samples") == "true"

	var testCases []*models.TestCase
	if onlySamples {
		testCases, err = api.testCaseModel.FindSamplesByProblemId(r.Context(), problemId)
	} else {
		testCases, err = api.testCaseModel.FindByProblemId(r.Context(), problemId)
	}

	if err != nil {
		log.Printf("Failed to get test cases: %v", err)
		api.writeError(w, http.StatusInternalServerError, "获取测试用例失败")
		return
	}

	// 转换为响应格式
	var responseList []models.TestCaseResponse
	for _, testCase := range testCases {
		resp := models.TestCaseResponse{
			Id:        testCase.Id,
			ProblemId: testCase.ProblemId,
			IsSample:  testCase.IsSample,
			Score:     testCase.Score,
			SortOrder: testCase.SortOrder,
			CreatedAt: testCase.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// 根据参数决定是否包含具体数据
		if includeData {
			resp.InputData = testCase.InputData
			resp.ExpectedOutput = testCase.ExpectedOutput
		} else {
			// 只显示数据长度，不显示具体内容（保护测试用例数据）
			resp.InputData = fmt.Sprintf("输入数据长度: %d 字符", len(testCase.InputData))
			resp.ExpectedOutput = fmt.Sprintf("输出数据长度: %d 字符", len(testCase.ExpectedOutput))
		}

		responseList = append(responseList, resp)
	}

	response := BaseResp{
		Code:    200,
		Message: "获取成功",
		Data: map[string]interface{}{
			"problem_id":   problemId,
			"total_count":  len(testCases),
			"test_cases":   responseList,
			"include_data": includeData,
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}

// getTestCaseDetail 获取测试用例详情
func (api *ProblemAPI) getTestCaseDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的测试用例ID")
		return
	}

	testCase, err := api.testCaseModel.FindOne(r.Context(), id)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "测试用例不存在")
		return
	}

	response := BaseResp{
		Code:    200,
		Message: "获取成功",
		Data: models.TestCaseResponse{
			Id:             testCase.Id,
			ProblemId:      testCase.ProblemId,
			InputData:      testCase.InputData,
			ExpectedOutput: testCase.ExpectedOutput,
			IsSample:       testCase.IsSample,
			Score:          testCase.Score,
			SortOrder:      testCase.SortOrder,
			CreatedAt:      testCase.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}

// updateTestCase 更新测试用例
func (api *ProblemAPI) updateTestCase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的测试用例ID")
		return
	}

	// 检查测试用例是否存在
	existing, err := api.testCaseModel.FindOne(r.Context(), id)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "测试用例不存在")
		return
	}

	var req models.TestCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的JSON格式")
		return
	}

	// 验证数据
	if req.InputData == "" || req.ExpectedOutput == "" {
		api.writeError(w, http.StatusBadRequest, "输入数据和期望输出不能为空")
		return
	}

	if req.Score <= 0 {
		req.Score = existing.Score // 保持原有分值
	}

	// 更新测试用例
	existing.InputData = req.InputData
	existing.ExpectedOutput = req.ExpectedOutput
	existing.IsSample = req.IsSample
	existing.Score = req.Score
	existing.SortOrder = req.SortOrder

	err = api.testCaseModel.Update(r.Context(), existing)
	if err != nil {
		log.Printf("Failed to update test case: %v", err)
		api.writeError(w, http.StatusInternalServerError, "更新测试用例失败")
		return
	}

	response := BaseResp{
		Code:    200,
		Message: "更新成功",
		Data: map[string]interface{}{
			"test_case_id": id,
			"updated_at":   time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}

// deleteTestCase 删除测试用例
func (api *ProblemAPI) deleteTestCase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的测试用例ID")
		return
	}

	// 检查测试用例是否存在
	_, err = api.testCaseModel.FindOne(r.Context(), id)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "测试用例不存在")
		return
	}

	err = api.testCaseModel.Delete(r.Context(), id)
	if err != nil {
		log.Printf("Failed to delete test case: %v", err)
		api.writeError(w, http.StatusInternalServerError, "删除测试用例失败")
		return
	}

	response := BaseResp{
		Code:    200,
		Message: "删除成功",
		Data: map[string]interface{}{
			"test_case_id": id,
			"deleted_at":   time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}

// ================== 内部接口（供判题服务调用） ==================

// getProblemDetailForJudge 获取题目详细信息（供判题服务调用）
func (api *ProblemAPI) getProblemDetailForJudge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	problemId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的题目ID")
		return
	}

	// 认证已在中间件中完成

	// 查找题目详情
	problem, err := api.problemModel.FindOne(r.Context(), problemId)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "题目不存在")
		return
	}

	// 判题服务可能需要处理已删除题目的遗留提交，所以这里不检查删除状态
	// 但会在响应中标注删除状态
	isDeleted := problem.DeletedAt.Valid

	// 解析JSON字段
	var languages, tags []string
	if problem.Languages.Valid {
		json.Unmarshal([]byte(problem.Languages.String), &languages)
	}
	if problem.Tags.Valid {
		json.Unmarshal([]byte(problem.Tags.String), &tags)
	}

	// 转换为判题服务需要的格式（简化版，只包含判题所需的核心信息）
	judgeResponse := map[string]interface{}{
		"id":            problem.Id,
		"title":         problem.Title,
		"description":   problem.Description,
		"input_format":  problem.InputFormat.String,
		"output_format": problem.OutputFormat.String,
		"sample_input":  problem.SampleInput.String,
		"sample_output": problem.SampleOutput.String,
		"difficulty":    problem.Difficulty,
		"time_limit":    problem.TimeLimit,    // 毫秒
		"memory_limit":  problem.MemoryLimit,  // MB
		"languages":     languages,
		"tags":          tags,
		"is_public":     problem.IsPublic,
		"is_deleted":    isDeleted,
		"created_by":    problem.CreatedBy,
		"created_at":    problem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":    problem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if isDeleted {
		judgeResponse["deleted_at"] = problem.DeletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	response := BaseResp{
		Code:    200,
		Message: "获取成功",
		Data: map[string]interface{}{
			"problem":      judgeResponse,
			"requested_at": time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}

// getTestCasesForJudge 获取题目的测试用例（供判题服务调用）
func (api *ProblemAPI) getTestCasesForJudge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	problemId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "无效的题目ID")
		return
	}

	// 认证已在中间件中完成

	// 检查题目是否存在（无需检查删除状态，判题服务可能需要处理已删除题目的遗留提交）
	_, err = api.problemModel.FindOne(r.Context(), problemId)
	if err != nil {
		api.writeError(w, http.StatusNotFound, "题目不存在")
		return
	}

	// 查询参数
	includeHidden := r.URL.Query().Get("include_hidden") == "true"

	var testCases []*models.TestCase
	if includeHidden {
		// 获取所有测试用例（包括隐藏用例）
		testCases, err = api.testCaseModel.FindByProblemId(r.Context(), problemId)
	} else {
		// 只获取示例用例
		testCases, err = api.testCaseModel.FindSamplesByProblemId(r.Context(), problemId)
	}

	if err != nil {
		log.Printf("Failed to get test cases for judge: %v", err)
		api.writeError(w, http.StatusInternalServerError, "获取测试用例失败")
		return
	}

	// 转换为判题服务需要的格式
	var judgeTestCases []map[string]interface{}
	for _, testCase := range testCases {
		judgeCase := map[string]interface{}{
			"id":              testCase.Id,
			"input_data":      testCase.InputData,
			"expected_output": testCase.ExpectedOutput,
			"is_sample":       testCase.IsSample,
			"score":           testCase.Score,
			"sort_order":      testCase.SortOrder,
		}
		judgeTestCases = append(judgeTestCases, judgeCase)
	}

	response := BaseResp{
		Code:    200,
		Message: "获取成功",
		Data: map[string]interface{}{
			"problem_id":      problemId,
			"total_count":     len(testCases),
			"test_cases":      judgeTestCases,
			"include_hidden":  includeHidden,
			"requested_at":    time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	api.writeJSON(w, http.StatusOK, response)
}
