package utils

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 响应状态码定义
const (
	CodeSuccess           = 200   // 成功
	CodeInvalidParams     = 400   // 参数错误
	CodeUnauthorized      = 401   // 未授权
	CodeForbidden         = 403   // 权限不足
	CodeNotFound          = 404   // 资源不存在
	CodeConflict          = 409   // 资源冲突
	CodeTooManyRequests   = 429   // 请求过多
	CodeInternalError     = 500   // 服务器内部错误
	CodeServiceUnavailable = 503  // 服务不可用
)

// 业务错误码定义
const (
	// 用户相关错误 (1000-1999)
	CodeUserNotFound         = 1001  // 用户不存在
	CodeUserAlreadyExists    = 1002  // 用户已存在
	CodeInvalidCredentials   = 1003  // 凭据无效
	CodeUserBanned           = 1004  // 用户被封禁
	CodeEmailNotVerified     = 1005  // 邮箱未验证
	CodePasswordTooWeak      = 1006  // 密码太弱
	CodeAccountLocked        = 1007  // 账户被锁定
	CodeInvalidToken         = 1008  // 无效令牌
	CodeTokenExpired         = 1009  // 令牌过期
	CodePermissionDenied     = 1010  // 权限拒绝

	// 题目相关错误 (2000-2999)
	CodeProblemNotFound      = 2001  // 题目不存在
	CodeProblemAlreadyExists = 2002  // 题目已存在

	// 提交相关错误 (3000-3999)
	CodeSubmissionNotFound   = 3001  // 提交不存在
	CodeLanguageNotSupported = 3002  // 语言不支持
)

// BaseResponse 基础响应结构
type BaseResponse struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 错误消息映射
var ErrorMessages = map[int64]string{
	CodeSuccess:              "操作成功",
	CodeInvalidParams:        "参数错误",
	CodeUnauthorized:         "未授权访问",
	CodeForbidden:            "权限不足",
	CodeNotFound:             "资源不存在",
	CodeConflict:             "资源冲突",
	CodeTooManyRequests:      "请求过多",
	CodeInternalError:        "服务器内部错误",
	CodeServiceUnavailable:   "服务不可用",
	
	// 用户相关
	CodeUserNotFound:         "用户不存在",
	CodeUserAlreadyExists:    "用户名或邮箱已存在",
	CodeInvalidCredentials:   "用户名或密码错误",
	CodeUserBanned:           "账户已被封禁",
	CodeEmailNotVerified:     "邮箱未验证",
	CodePasswordTooWeak:      "密码强度不够",
	CodeAccountLocked:        "账户已被锁定",
	CodeInvalidToken:         "无效的令牌",
	CodeTokenExpired:         "令牌已过期",
	CodePermissionDenied:     "权限不足",
	
	// 题目相关
	CodeProblemNotFound:      "题目不存在",
	CodeProblemAlreadyExists: "题目已存在",
	
	// 提交相关
	CodeSubmissionNotFound:   "提交记录不存在",
	CodeLanguageNotSupported: "不支持的编程语言",
}

// GetMessage 获取错误消息
func GetMessage(code int64) string {
	if msg, exists := ErrorMessages[code]; exists {
		return msg
	}
	return "未知错误"
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	response := BaseResponse{
		Code:    CodeSuccess,
		Message: GetMessage(CodeSuccess),
		Data:    data,
	}
	httpx.OkJson(w, response)
}

// Error 错误响应
func Error(w http.ResponseWriter, code int64, message ...string) {
	msg := GetMessage(code)
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	
	response := BaseResponse{
		Code:    code,
		Message: msg,
	}
	
	// 根据错误码设置HTTP状态码
	var httpStatus int
	switch {
	case code >= 400 && code < 500:
		httpStatus = int(code)
	case code == CodeUserNotFound || code == CodeProblemNotFound:
		httpStatus = http.StatusNotFound
	case code == CodeUserAlreadyExists:
		httpStatus = http.StatusConflict
	case code == CodeInvalidCredentials || code == CodeInvalidToken:
		httpStatus = http.StatusUnauthorized
	case code == CodePermissionDenied || code == CodeUserBanned:
		httpStatus = http.StatusForbidden
	case code == CodeInvalidParams:
		httpStatus = http.StatusBadRequest
	default:
		httpStatus = http.StatusInternalServerError
	}
	
	httpx.WriteJson(w, httpStatus, response)
}

// ErrorWithHttpStatus 带HTTP状态码的错误响应
func ErrorWithHttpStatus(w http.ResponseWriter, httpStatus int, code int64, message ...string) {
	msg := GetMessage(code)
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	
	response := BaseResponse{
		Code:    code,
		Message: msg,
	}
	
	httpx.WriteJson(w, httpStatus, response)
}