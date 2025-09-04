package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dszqbsm/code-judger/common/consul"
	"github.com/zeromicro/go-zero/core/logx"
)

// HTTPRPCClient HTTP RPC客户端
type HTTPRPCClient struct {
	serviceName     string
	serviceResolver *consul.ServiceResolver
	httpClient      *http.Client
	timeout         time.Duration
}

// NewHTTPRPCClient 创建HTTP RPC客户端
func NewHTTPRPCClient(serviceName string, consulAddr string, timeout time.Duration) (*HTTPRPCClient, error) {
	// 创建服务解析器
	resolver, err := consul.NewServiceResolver(consulAddr, consul.NewRoundRobinBalancer())
	if err != nil {
		return nil, fmt.Errorf("创建服务解析器失败: %w", err)
	}

	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}

	return &HTTPRPCClient{
		serviceName:     serviceName,
		serviceResolver: resolver,
		httpClient:      httpClient,
		timeout:         timeout,
	}, nil
}

// Call 调用远程服务
func (c *HTTPRPCClient) Call(ctx context.Context, method string, path string, request interface{}, response interface{}) error {
	// 解析服务实例
	instance, err := c.serviceResolver.ResolveService(c.serviceName)
	if err != nil {
		return fmt.Errorf("解析服务实例失败: %w", err)
	}

	// 构建URL
	url := fmt.Sprintf("http://%s%s", instance.GetEndpoint(), path)

	// 序列化请求数据
	var body io.Reader
	if request != nil {
		data, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("序列化请求数据失败: %w", err)
		}
		body = strings.NewReader(string(data))
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "rpc-client/1.0.0")
	req.Header.Set("X-Service-Name", c.serviceName)

	// 执行请求
	logx.WithContext(ctx).Infof("RPC调用: %s %s", method, url)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("执行HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应数据失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP请求失败: status=%d, body=%s", resp.StatusCode, string(respData))
	}

	// 反序列化响应数据
	if response != nil {
		if err := json.Unmarshal(respData, response); err != nil {
			return fmt.Errorf("反序列化响应数据失败: %w", err)
		}
	}

	logx.WithContext(ctx).Infof("RPC调用成功: %s %s", method, url)
	return nil
}

// Get 发送GET请求
func (c *HTTPRPCClient) Get(ctx context.Context, path string, response interface{}) error {
	return c.Call(ctx, "GET", path, nil, response)
}

// Post 发送POST请求
func (c *HTTPRPCClient) Post(ctx context.Context, path string, request interface{}, response interface{}) error {
	return c.Call(ctx, "POST", path, request, response)
}

// Put 发送PUT请求
func (c *HTTPRPCClient) Put(ctx context.Context, path string, request interface{}, response interface{}) error {
	return c.Call(ctx, "PUT", path, request, response)
}

// Delete 发送DELETE请求
func (c *HTTPRPCClient) Delete(ctx context.Context, path string, response interface{}) error {
	return c.Call(ctx, "DELETE", path, nil, response)
}

// ClientPool RPC客户端池
type ClientPool struct {
	clients map[string]*HTTPRPCClient
	config  ClientPoolConfig
}

// ClientPoolConfig 客户端池配置
type ClientPoolConfig struct {
	ConsulAddr     string        `json:"consul_addr"`
	DefaultTimeout time.Duration `json:"default_timeout"`
}

// NewClientPool 创建RPC客户端池
func NewClientPool(config ClientPoolConfig) *ClientPool {
	return &ClientPool{
		clients: make(map[string]*HTTPRPCClient),
		config:  config,
	}
}

// GetClient 获取RPC客户端
func (p *ClientPool) GetClient(serviceName string) (*HTTPRPCClient, error) {
	// 检查是否已存在客户端
	if client, exists := p.clients[serviceName]; exists {
		return client, nil
	}

	// 创建新的客户端
	client, err := NewHTTPRPCClient(serviceName, p.config.ConsulAddr, p.config.DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("创建RPC客户端失败: %w", err)
	}

	// 缓存客户端
	p.clients[serviceName] = client
	return client, nil
}

// GetOrCreateClient 获取或创建RPC客户端
func (p *ClientPool) GetOrCreateClient(serviceName string, timeout time.Duration) (*HTTPRPCClient, error) {
	// 检查是否已存在客户端
	if client, exists := p.clients[serviceName]; exists {
		return client, nil
	}

	// 创建新的客户端
	client, err := NewHTTPRPCClient(serviceName, p.config.ConsulAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("创建RPC客户端失败: %w", err)
	}

	// 缓存客户端
	p.clients[serviceName] = client
	return client, nil
}

// RemoveClient 移除客户端
func (p *ClientPool) RemoveClient(serviceName string) {
	delete(p.clients, serviceName)
}

// Close 关闭客户端池
func (p *ClientPool) Close() {
	p.clients = make(map[string]*HTTPRPCClient)
}

// CircuitBreaker 熔断器接口
type CircuitBreaker interface {
	Call(ctx context.Context, fn func() error) error
	IsOpen() bool
}

// SimpleCircuitBreaker 简单熔断器实现
type SimpleCircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failureCount     int
	lastFailureTime  time.Time
	state            CircuitState
}

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewSimpleCircuitBreaker 创建简单熔断器
func NewSimpleCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *SimpleCircuitBreaker {
	return &SimpleCircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
	}
}

// Call 执行调用
func (cb *SimpleCircuitBreaker) Call(ctx context.Context, fn func() error) error {
	// 检查熔断器状态
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.failureCount = 0
		} else {
			return fmt.Errorf("熔断器开启，拒绝调用")
		}
	}

	// 执行调用
	err := fn()
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// IsOpen 检查熔断器是否开启
func (cb *SimpleCircuitBreaker) IsOpen() bool {
	return cb.state == StateOpen
}

// onSuccess 成功处理
func (cb *SimpleCircuitBreaker) onSuccess() {
	cb.failureCount = 0
	cb.state = StateClosed
}

// onFailure 失败处理
func (cb *SimpleCircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		RetryDelay:    100 * time.Millisecond,
		BackoffFactor: 2.0,
	}
}

// WithRetry 带重试功能的RPC调用
func WithRetry(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	delay := config.RetryDelay

	for i := 0; i <= config.MaxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay = time.Duration(float64(delay) * config.BackoffFactor)
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		logx.WithContext(ctx).Errorf("RPC调用失败 (重试 %d/%d): %v", i+1, config.MaxRetries+1, err)
	}

	return fmt.Errorf("RPC调用最终失败: %w", lastErr)
}
