package anticheat

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"code-judger/services/submission-api/internal/config"
	"code-judger/services/submission-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type Detector struct {
	config           config.AntiCheatConf
	submissionModel  models.SubmissionModel
	featureExtractor *FeatureExtractor
}

type SimilarityResult struct {
	SubmissionID1   int64    `json:"submission_id_1"`
	SubmissionID2   int64    `json:"submission_id_2"`
	UserID1         int64    `json:"user_id_1"`
	UserID2         int64    `json:"user_id_2"`
	SimilarityScore float64  `json:"similarity_score"`
	MatchedFeatures []string `json:"matched_features"`
	Confidence      float64  `json:"confidence"`
}

type FeatureExtractor struct {
	patterns map[string]*regexp.Regexp
}

func NewDetector(config config.AntiCheatConf, submissionModel models.SubmissionModel) *Detector {
	return &Detector{
		config:           config,
		submissionModel:  submissionModel,
		featureExtractor: NewFeatureExtractor(),
	}
}

func NewFeatureExtractor() *FeatureExtractor {
	patterns := map[string]*regexp.Regexp{
		"for_loop":         regexp.MustCompile(`for\s*\(`),
		"while_loop":       regexp.MustCompile(`while\s*\(`),
		"if_statement":     regexp.MustCompile(`if\s*\(`),
		"function_call":    regexp.MustCompile(`\w+\s*\(`),
		"cpp_include":      regexp.MustCompile(`#include\s*<.*?>`),
		"cpp_namespace":    regexp.MustCompile(`using\s+namespace\s+\w+;`),
		"java_import":      regexp.MustCompile(`import\s+[\w.]+;`),
		"java_class":       regexp.MustCompile(`class\s+\w+`),
		"python_import":    regexp.MustCompile(`import\s+\w+|from\s+\w+\s+import`),
		"python_list_comp": regexp.MustCompile(`\[.*for.*in.*\]`),
	}

	return &FeatureExtractor{
		patterns: patterns,
	}
}

// DetectSimilarity 检测两个提交的相似度
func (d *Detector) DetectSimilarity(ctx context.Context, submission1, submission2 *models.Submission) (*SimilarityResult, error) {
	if !d.config.Enabled {
		return nil, fmt.Errorf("查重检测功能未启用")
	}

	// 预处理代码
	code1 := d.preprocessCode(submission1.Code, submission1.Language)
	code2 := d.preprocessCode(submission2.Code, submission2.Language)

	// 多层次相似度检测
	similarities := make(map[string]float64)

	// 1. 字符串级别相似度
	if d.config.Algorithms.EnableStringMatch {
		similarities["string"] = d.calculateStringSimilarity(code1, code2)
	}

	// 2. 特征级别相似度
	if d.config.Algorithms.EnableFeatureMatch {
		features1 := d.featureExtractor.Extract(code1, submission1.Language)
		features2 := d.featureExtractor.Extract(code2, submission2.Language)
		similarities["features"] = d.calculateFeatureSimilarity(features1, features2)
	}

	// 3. 综合相似度计算
	finalScore := d.calculateFinalScore(similarities)

	// 4. 置信度评估
	confidence := d.calculateConfidence(similarities)

	// 5. 获取匹配特征
	features1 := d.featureExtractor.Extract(code1, submission1.Language)
	features2 := d.featureExtractor.Extract(code2, submission2.Language)
	matchedFeatures := d.getMatchedFeatures(features1, features2)

	result := &SimilarityResult{
		SubmissionID1:   submission1.Id,
		SubmissionID2:   submission2.Id,
		UserID1:         submission1.UserId,
		UserID2:         submission2.UserId,
		SimilarityScore: finalScore,
		MatchedFeatures: matchedFeatures,
		Confidence:      confidence,
	}

	return result, nil
}

// BatchDetection 批量查重检测
func (d *Detector) BatchDetection(ctx context.Context, problemID int64, contestID *int64) ([]*SimilarityResult, error) {
	if !d.config.Enabled {
		return nil, fmt.Errorf("查重检测功能未启用")
	}

	// 获取待检测的提交
	submissions, err := d.getSubmissionsForDetection(ctx, problemID, contestID)
	if err != nil {
		return nil, err
	}

	var results []*SimilarityResult

	// 两两比较
	for i := 0; i < len(submissions); i++ {
		for j := i + 1; j < len(submissions); j++ {
			// 跳过同一用户的提交
			if submissions[i].UserId == submissions[j].UserId {
				continue
			}

			result, err := d.DetectSimilarity(ctx, submissions[i], submissions[j])
			if err != nil {
				logx.Errorf("检测相似度失败: %v", err)
				continue
			}

			// 只保留相似度超过阈值的结果
			if result.SimilarityScore >= d.config.SimilarityThreshold {
				results = append(results, result)
			}
		}
	}

	// 按相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].SimilarityScore > results[j].SimilarityScore
	})

	return results, nil
}

// preprocessCode 代码预处理
func (d *Detector) preprocessCode(code, language string) string {
	// 1. 移除注释
	code = d.removeComments(code, language)

	// 2. 标准化空白字符
	code = strings.ReplaceAll(code, "\t", " ")
	code = regexp.MustCompile(`\s+`).ReplaceAllString(code, " ")

	// 3. 变量名标准化（可选）
	// code = d.normalizeVariableNames(code, language)

	return strings.TrimSpace(code)
}

// removeComments 移除注释
func (d *Detector) removeComments(code, language string) string {
	switch language {
	case "cpp", "c", "java", "javascript", "go":
		// 移除单行注释
		code = regexp.MustCompile(`//.*`).ReplaceAllString(code, "")
		// 移除多行注释
		code = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(code, "")
	case "python":
		// 移除Python注释
		code = regexp.MustCompile(`#.*`).ReplaceAllString(code, "")
		// 移除多行字符串注释
		code = regexp.MustCompile(`"""[\s\S]*?"""|'''[\s\S]*?'''`).ReplaceAllString(code, "")
	}
	return code
}

// calculateStringSimilarity 计算字符串相似度（编辑距离）
func (d *Detector) calculateStringSimilarity(code1, code2 string) float64 {
	if len(code1) == 0 && len(code2) == 0 {
		return 1.0
	}

	if len(code1) == 0 || len(code2) == 0 {
		return 0.0
	}

	editDistance := d.calculateEditDistance(code1, code2)
	maxLen := math.Max(float64(len(code1)), float64(len(code2)))

	return 1.0 - float64(editDistance)/maxLen
}

// calculateEditDistance 计算编辑距离
func (d *Detector) calculateEditDistance(s1, s2 string) int {
	m, n := len(s1), len(s2)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// 初始化
	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	// 动态规划计算编辑距离
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = min3(dp[i-1][j], dp[i][j-1], dp[i-1][j-1]) + 1
			}
		}
	}

	return dp[m][n]
}

// Extract 提取代码特征
func (fe *FeatureExtractor) Extract(code, language string) map[string]int {
	features := make(map[string]int)

	// 1. 语法特征
	features["for_loops"] = len(fe.patterns["for_loop"].FindAllString(code, -1))
	features["while_loops"] = len(fe.patterns["while_loop"].FindAllString(code, -1))
	features["if_statements"] = len(fe.patterns["if_statement"].FindAllString(code, -1))
	features["function_calls"] = len(fe.patterns["function_call"].FindAllString(code, -1))

	// 2. 结构特征
	features["nested_depth"] = fe.calculateNestedDepth(code)
	features["line_count"] = len(strings.Split(code, "\n"))
	features["char_count"] = len(code)

	// 3. 语言特定特征
	switch language {
	case "cpp", "c":
		features["includes"] = len(fe.patterns["cpp_include"].FindAllString(code, -1))
		features["namespaces"] = len(fe.patterns["cpp_namespace"].FindAllString(code, -1))
	case "java":
		features["imports"] = len(fe.patterns["java_import"].FindAllString(code, -1))
		features["classes"] = len(fe.patterns["java_class"].FindAllString(code, -1))
	case "python":
		features["imports"] = len(fe.patterns["python_import"].FindAllString(code, -1))
		features["list_comprehensions"] = len(fe.patterns["python_list_comp"].FindAllString(code, -1))
	}

	return features
}

// calculateNestedDepth 计算嵌套深度
func (fe *FeatureExtractor) calculateNestedDepth(code string) int {
	maxDepth := 0
	currentDepth := 0

	for _, char := range code {
		switch char {
		case '{', '(':
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		case '}', ')':
			currentDepth--
		}
	}

	return maxDepth
}

// calculateFeatureSimilarity 计算特征相似度
func (d *Detector) calculateFeatureSimilarity(features1, features2 map[string]int) float64 {
	// 使用Jaccard相似度
	intersection := 0
	union := 0

	allKeys := make(map[string]bool)
	for k := range features1 {
		allKeys[k] = true
	}
	for k := range features2 {
		allKeys[k] = true
	}

	for key := range allKeys {
		val1, exists1 := features1[key]
		val2, exists2 := features2[key]

		if !exists1 {
			val1 = 0
		}
		if !exists2 {
			val2 = 0
		}

		intersection += min(val1, val2)
		union += max(val1, val2)
	}

	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

// calculateFinalScore 计算最终相似度分数
func (d *Detector) calculateFinalScore(similarities map[string]float64) float64 {
	totalWeight := 0.0
	weightedSum := 0.0

	if score, exists := similarities["string"]; exists {
		weight := d.config.Algorithms.StringWeight
		weightedSum += score * weight
		totalWeight += weight
	}

	if score, exists := similarities["features"]; exists {
		weight := d.config.Algorithms.FeatureWeight
		weightedSum += score * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSum / totalWeight
}

// calculateConfidence 计算置信度
func (d *Detector) calculateConfidence(similarities map[string]float64) float64 {
	// 简单的置信度计算：基于参与计算的相似度指标数量
	count := len(similarities)
	if count == 0 {
		return 0.0
	}

	// 置信度与参与计算的指标数量成正比
	confidence := float64(count) / 3.0 // 假设最多有3种相似度计算方法
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// getMatchedFeatures 获取匹配的特征
func (d *Detector) getMatchedFeatures(features1, features2 map[string]int) []string {
	var matched []string

	for key, val1 := range features1 {
		if val2, exists := features2[key]; exists && val1 > 0 && val2 > 0 {
			// 如果两个代码都包含此特征，且数量相近
			ratio := float64(min(val1, val2)) / float64(max(val1, val2))
			if ratio >= 0.8 { // 80%以上相似度认为是匹配
				matched = append(matched, key)
			}
		}
	}

	return matched
}

// getSubmissionsForDetection 获取用于检测的提交
func (d *Detector) getSubmissionsForDetection(ctx context.Context, problemID int64, contestID *int64) ([]*models.Submission, error) {
	condition := &models.SearchCondition{
		ProblemID: &problemID,
		Page:      1,
		PageSize:  d.config.BatchSize,
	}

	if contestID != nil {
		condition.ContestID = contestID
	}

	submissions, _, err := d.submissionModel.Search(ctx, condition)
	if err != nil {
		return nil, fmt.Errorf("获取提交记录失败: %v", err)
	}

	// 只检测已通过的提交
	var acceptedSubmissions []*models.Submission
	for _, submission := range submissions {
		if submission.Status == "accepted" {
			acceptedSubmissions = append(acceptedSubmissions, submission)
		}
	}

	return acceptedSubmissions, nil
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
