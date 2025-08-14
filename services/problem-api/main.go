package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	"github.com/online-judge/code-judger/services/problem-api/models"
)

type ProblemAPI struct {
	db           *sql.DB
	problemModel models.ProblemModel
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

	api := &ProblemAPI{
		db:           db,
		problemModel: models.NewProblemModel(db),
	}

	// 创建路由
	r := mux.NewRouter()
	api.setupRoutes(r)

	// 启动服务器
	fmt.Println("Problem API server starting on :8889...")
	log.Fatal(http.ListenAndServe(":8889", r))
}

func (api *ProblemAPI) setupRoutes(r *mux.Router) {
	// 健康检查
	r.HandleFunc("/api/v1/health", api.healthCheck).Methods("GET")

	// 题目管理接口
	r.HandleFunc("/api/v1/problems", api.createProblem).Methods("POST")
	r.HandleFunc("/api/v1/problems", api.getProblemList).Methods("GET")
	r.HandleFunc("/api/v1/problems/{id}", api.getProblemDetail).Methods("GET")
	r.HandleFunc("/api/v1/problems/{id}", api.updateProblem).Methods("PUT")
	r.HandleFunc("/api/v1/problems/{id}", api.deleteProblem).Methods("DELETE")
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
