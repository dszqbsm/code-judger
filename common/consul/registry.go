package consul

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/zeromicro/go-zero/core/logx"
)

// ServiceRegistry Consul服务注册器
type ServiceRegistry struct {
	client     *api.Client
	serviceID  string
	serverAddr string
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	ServiceName string            `json:"service_name"`
	ServiceID   string            `json:"service_id"`
	Address     string            `json:"address"`
	Port        int               `json:"port"`
	Tags        []string          `json:"tags"`
	Health      HealthCheck       `json:"health"`
	Meta        map[string]string `json:"meta"`
}

// HealthCheck 健康检查配置
type HealthCheck struct {
	HTTP                           string        `json:"http"`
	Interval                       time.Duration `json:"interval"`
	Timeout                        time.Duration `json:"timeout"`
	DeregisterCriticalServiceAfter time.Duration `json:"deregister_critical_service_after"`
}

// NewServiceRegistry 创建新的服务注册器
func NewServiceRegistry(consulAddr, serverAddr string) (*ServiceRegistry, error) {
	config := api.DefaultConfig()
	config.Address = consulAddr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建Consul客户端失败: %w", err)
	}

	return &ServiceRegistry{
		client:     client,
		serverAddr: serverAddr,
	}, nil
}

// Register 注册服务到Consul
func (r *ServiceRegistry) Register(serviceInfo ServiceInfo) error {
	// 生成唯一的服务ID
	if serviceInfo.ServiceID == "" {
		serviceInfo.ServiceID = fmt.Sprintf("%s-%s-%d", serviceInfo.ServiceName, serviceInfo.Address, serviceInfo.Port)
	}
	
	r.serviceID = serviceInfo.ServiceID

	// 构建服务注册信息
	registration := &api.AgentServiceRegistration{
		ID:      serviceInfo.ServiceID,
		Name:    serviceInfo.ServiceName,
		Tags:    serviceInfo.Tags,
		Address: serviceInfo.Address,
		Port:    serviceInfo.Port,
		Meta:    serviceInfo.Meta,
	}

	// 添加健康检查
	if serviceInfo.Health.HTTP != "" {
		registration.Check = &api.AgentServiceCheck{
			HTTP:                           serviceInfo.Health.HTTP,
			Interval:                       serviceInfo.Health.Interval.String(),
			Timeout:                        serviceInfo.Health.Timeout.String(),
			DeregisterCriticalServiceAfter: serviceInfo.Health.DeregisterCriticalServiceAfter.String(),
		}
	}

	// 注册服务
	err := r.client.Agent().ServiceRegister(registration)
	if err != nil {
		return fmt.Errorf("服务注册失败: %w", err)
	}

	logx.Infof("服务注册成功: ServiceName=%s, ServiceID=%s, Address=%s:%d", 
		serviceInfo.ServiceName, serviceInfo.ServiceID, serviceInfo.Address, serviceInfo.Port)

	return nil
}

// Deregister 从Consul注销服务
func (r *ServiceRegistry) Deregister() error {
	if r.serviceID == "" {
		return nil
	}

	err := r.client.Agent().ServiceDeregister(r.serviceID)
	if err != nil {
		return fmt.Errorf("服务注销失败: %w", err)
	}

	logx.Infof("服务注销成功: ServiceID=%s", r.serviceID)
	return nil
}

// ServiceDiscovery Consul服务发现器
type ServiceDiscovery struct {
	client *api.Client
}

// NewServiceDiscovery 创建新的服务发现器
func NewServiceDiscovery(consulAddr string) (*ServiceDiscovery, error) {
	config := api.DefaultConfig()
	config.Address = consulAddr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建Consul客户端失败: %w", err)
	}

	return &ServiceDiscovery{
		client: client,
	}, nil
}

// DiscoverService 发现服务实例
func (d *ServiceDiscovery) DiscoverService(serviceName string) ([]*ServiceInstance, error) {
	services, _, err := d.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("服务发现失败: %w", err)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("未找到可用的服务实例: %s", serviceName)
	}

	var instances []*ServiceInstance
	for _, service := range services {
		instance := &ServiceInstance{
			ServiceID:   service.Service.ID,
			ServiceName: service.Service.Service,
			Address:     service.Service.Address,
			Port:        service.Service.Port,
			Tags:        service.Service.Tags,
			Meta:        service.Service.Meta,
		}
		instances = append(instances, instance)
	}

	logx.Infof("发现服务实例: ServiceName=%s, Count=%d", serviceName, len(instances))
	return instances, nil
}

// ServiceInstance 服务实例信息
type ServiceInstance struct {
	ServiceID   string            `json:"service_id"`
	ServiceName string            `json:"service_name"`
	Address     string            `json:"address"`
	Port        int               `json:"port"`
	Tags        []string          `json:"tags"`
	Meta        map[string]string `json:"meta"`
}

// GetEndpoint 获取服务端点
func (s *ServiceInstance) GetEndpoint() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Select(instances []*ServiceInstance) *ServiceInstance
}

// RoundRobinBalancer 轮询负载均衡器
type RoundRobinBalancer struct {
	current int
}

// NewRoundRobinBalancer 创建轮询负载均衡器
func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{current: 0}
}

// Select 选择服务实例
func (r *RoundRobinBalancer) Select(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	instance := instances[r.current%len(instances)]
	r.current++
	return instance
}

// ServiceResolver 服务解析器
type ServiceResolver struct {
	discovery    *ServiceDiscovery
	balancer     LoadBalancer
	serviceCache map[string][]*ServiceInstance
	lastUpdate   map[string]time.Time
	cacheTTL     time.Duration
}

// NewServiceResolver 创建服务解析器
func NewServiceResolver(consulAddr string, balancer LoadBalancer) (*ServiceResolver, error) {
	discovery, err := NewServiceDiscovery(consulAddr)
	if err != nil {
		return nil, err
	}

	if balancer == nil {
		balancer = NewRoundRobinBalancer()
	}

	return &ServiceResolver{
		discovery:    discovery,
		balancer:     balancer,
		serviceCache: make(map[string][]*ServiceInstance),
		lastUpdate:   make(map[string]time.Time),
		cacheTTL:     30 * time.Second, // 缓存30秒
	}, nil
}

// ResolveService 解析服务实例
func (r *ServiceResolver) ResolveService(serviceName string) (*ServiceInstance, error) {
	// 检查缓存
	instances, cached := r.getFromCache(serviceName)
	if !cached {
		// 从Consul获取最新服务列表
		var err error
		instances, err = r.discovery.DiscoverService(serviceName)
		if err != nil {
			return nil, err
		}
		
		// 更新缓存
		r.updateCache(serviceName, instances)
	}

	// 使用负载均衡器选择实例
	instance := r.balancer.Select(instances)
	if instance == nil {
		return nil, fmt.Errorf("无可用的服务实例: %s", serviceName)
	}

	return instance, nil
}

// getFromCache 从缓存获取服务实例
func (r *ServiceResolver) getFromCache(serviceName string) ([]*ServiceInstance, bool) {
	lastUpdate, exists := r.lastUpdate[serviceName]
	if !exists || time.Since(lastUpdate) > r.cacheTTL {
		return nil, false
	}

	instances, exists := r.serviceCache[serviceName]
	return instances, exists
}

// updateCache 更新缓存
func (r *ServiceResolver) updateCache(serviceName string, instances []*ServiceInstance) {
	r.serviceCache[serviceName] = instances
	r.lastUpdate[serviceName] = time.Now()
}

// HealthChecker 健康检查器
type HealthChecker struct {
	registry     *ServiceRegistry
	checkFunc    func() bool
	interval     time.Duration
	stopCh       chan struct{}
	healthStatus bool
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(registry *ServiceRegistry, checkFunc func() bool, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		registry:     registry,
		checkFunc:    checkFunc,
		interval:     interval,
		stopCh:       make(chan struct{}),
		healthStatus: true,
	}
}

// Start 启动健康检查
func (h *HealthChecker) Start() {
	go h.run()
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	close(h.stopCh)
}

// run 运行健康检查循环
func (h *HealthChecker) run() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			healthy := h.checkFunc()
			if healthy != h.healthStatus {
				h.healthStatus = healthy
				if healthy {
					logx.Info("服务健康状态恢复")
				} else {
					logx.Error("服务健康状态异常")
				}
			}
		case <-h.stopCh:
			return
		}
	}
}

// GetServerInfo 获取服务器信息
func GetServerInfo(addr string) (string, int, error) {
	host, portStr, err := parseAddr(addr)
	if err != nil {
		return "", 0, err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("解析端口失败: %w", err)
	}

	return host, port, nil
}

// parseAddr 解析地址
func parseAddr(addr string) (string, string, error) {
	// 简单的地址解析，支持 "host:port" 格式
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("无效的地址格式: %s", addr)
}
