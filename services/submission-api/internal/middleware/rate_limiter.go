package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type RateLimiter struct {
	redisClient *redis.Redis
	maxRequests int
	window      time.Duration
}

func NewRateLimiter(redisClient *redis.Redis, maxRequestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
		maxRequests: maxRequestsPerMinute,
		window:      time.Minute,
	}
}

func (rl *RateLimiter) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取用户信息
		user, ok := GetUserFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":401,"message":"用户信息不存在"}`))
			return
		}

		// 检查提交频率限制
		if !rl.checkSubmitRate(user.UserID) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"code":429,"message":"提交频率过高，请稍后再试"}`))
			return
		}

		next(w, r)
	}
}

// checkSubmitRate 检查用户提交频率
func (rl *RateLimiter) checkSubmitRate(userID int64) bool {
	key := fmt.Sprintf("submit_rate:%d", userID)
	now := time.Now().Unix()
	window := int64(rl.window.Seconds())

	// 使用Lua脚本实现滑动窗口限流
	script := `
		local key = KEYS[1]
		local window = tonumber(ARGV[1])
		local limit = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		-- 移除过期的记录
		redis.call('zremrangebyscore', key, 0, now - window)
		
		-- 获取当前窗口内的请求数
		local current = redis.call('zcard', key)
		
		if current < limit then
			-- 添加当前请求
			redis.call('zadd', key, now, now)
			redis.call('expire', key, window)
			return 1
		else
			return 0
		end
	`

	result, err := rl.redisClient.Eval(script, []string{key}, strconv.FormatInt(window, 10), strconv.Itoa(rl.maxRequests), strconv.FormatInt(now, 10))
	if err != nil {
		// 如果Redis出错，默认允许请求通过
		return true
	}

	allowed, _ := result.(int64)
	return allowed == 1
}
