package sandbox

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// cgroups资源控制实现
// 原理：通过Linux内核的cgroups机制实现精确的资源控制
// cgroups将进程划分为组，为每个组配置资源配额上限，由内核子系统实时监控并强制执行限制

// cgroup子系统类型
const (
	CgroupMemory = "memory" // 内存子系统
	CgroupCPU    = "cpu"    // CPU子系统
	CgroupCPUSet = "cpuset" // CPU集合子系统
	CgroupPIDs   = "pids"   // 进程ID子系统
	CgroupBlkIO  = "blkio"  // 块设备I/O子系统
)

// cgroup根路径
const (
	CgroupRootPath = "/sys/fs/cgroup"
	JudgeRootGroup = "judge" // 判题服务根组
)

// CgroupConfig cgroup配置结构体
type CgroupConfig struct {
	// 基础配置
	GroupName string // 控制组名称
	Language  string // 编程语言
	TaskID    string // 任务ID

	// 内存限制配置
	MemoryLimitBytes     int64 // 内存限制（字节）
	MemorySwapLimit      int64 // 内存+swap限制（字节）
	MemoryOOMKillDisable bool  // 禁用OOM杀死

	// CPU限制配置
	CPUQuotaUs  int64  // CPU配额（微秒）
	CPUPeriodUs int64  // CPU周期（微秒）
	CPUShares   int64  // CPU权重
	CPUSetCPUs  string // 允许使用的CPU核心

	// 进程限制配置
	PIDsMax int64 // 最大进程数

	// I/O限制配置
	BlkIOWeight   int64 // I/O权重
	BlkIOReadBps  int64 // 读取带宽限制（字节/秒）
	BlkIOWriteBps int64 // 写入带宽限制（字节/秒）
}

// CgroupManager cgroup管理器
type CgroupManager struct {
	config     *CgroupConfig
	groupPaths map[string]string // 各子系统的组路径
	created    bool              // 是否已创建
}

// CgroupStats cgroup统计信息
type CgroupStats struct {
	// 内存统计
	MemoryUsage    int64 // 当前内存使用量
	MemoryMaxUsage int64 // 内存使用峰值
	MemoryLimit    int64 // 内存限制
	MemoryOOMCount int64 // OOM事件次数

	// CPU统计
	CPUUsageTotal   int64   // 总CPU使用时间（纳秒）
	CPUUsageUser    int64   // 用户态CPU时间
	CPUUsageSystem  int64   // 内核态CPU时间
	CPUThrottled    int64   // CPU被限流的次数
	CPUUsagePercent float64 // CPU使用率百分比

	// 进程统计
	PIDsCurrent int64 // 当前进程数
	PIDsMax     int64 // 最大进程数限制

	// I/O统计
	BlkIOReadBytes  int64 // 读取字节数
	BlkIOWriteBytes int64 // 写入字节数
	BlkIOReadOps    int64 // 读取操作次数
	BlkIOWriteOps   int64 // 写入操作次数
}

// NewCgroupManager 创建新的cgroup管理器
func NewCgroupManager(config *CgroupConfig) *CgroupManager {
	manager := &CgroupManager{
		config:     config,
		groupPaths: make(map[string]string),
		created:    false,
	}

	// 构建各子系统的组路径
	manager.buildGroupPaths()

	logx.Infof("Created cgroup manager for group: %s", config.GroupName)
	return manager
}

// 构建各子系统的组路径
func (c *CgroupManager) buildGroupPaths() {
	subsystems := []string{CgroupMemory, CgroupCPU, CgroupCPUSet, CgroupPIDs, CgroupBlkIO}

	for _, subsystem := range subsystems {
		// 路径格式: /sys/fs/cgroup/{subsystem}/judge/{language}/{group_name}
		path := filepath.Join(CgroupRootPath, subsystem, JudgeRootGroup, c.config.Language, c.config.GroupName)
		c.groupPaths[subsystem] = path
	}

	logx.Debugf("Built cgroup paths: %+v", c.groupPaths)
}

// Create 创建cgroup控制组
func (c *CgroupManager) Create() error {
	if c.created {
		return fmt.Errorf("cgroup already created")
	}

	logx.Infof("Creating cgroup: %s", c.config.GroupName)

	// 1. 确保父级目录存在
	if err := c.ensureParentDirectories(); err != nil {
		return fmt.Errorf("failed to ensure parent directories: %w", err)
	}

	// 2. 创建各子系统的控制组目录
	for subsystem, path := range c.groupPaths {
		if err := os.MkdirAll(path, 0755); err != nil {
			// 清理已创建的目录
			c.cleanupPartialCreation()
			return fmt.Errorf("failed to create cgroup directory for %s: %w", subsystem, err)
		}
		logx.Debugf("Created cgroup directory: %s", path)
	}

	// 3. 设置资源限制
	if err := c.applyLimits(); err != nil {
		c.cleanupPartialCreation()
		return fmt.Errorf("failed to apply limits: %w", err)
	}

	c.created = true
	logx.Infof("Successfully created cgroup: %s", c.config.GroupName)
	return nil
}

// 确保父级目录存在
func (c *CgroupManager) ensureParentDirectories() error {
	parentDirs := []string{
		filepath.Join(CgroupRootPath, CgroupMemory, JudgeRootGroup),
		filepath.Join(CgroupRootPath, CgroupCPU, JudgeRootGroup),
		filepath.Join(CgroupRootPath, CgroupCPUSet, JudgeRootGroup),
		filepath.Join(CgroupRootPath, CgroupPIDs, JudgeRootGroup),
		filepath.Join(CgroupRootPath, CgroupBlkIO, JudgeRootGroup),
	}

	for _, dir := range parentDirs {
		languageDir := filepath.Join(dir, c.config.Language)
		if err := os.MkdirAll(languageDir, 0755); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to create parent directory %s: %w", languageDir, err)
		}
	}

	return nil
}

// 应用资源限制配置
func (c *CgroupManager) applyLimits() error {
	var errors []error

	// 应用内存限制
	if err := c.applyMemoryLimits(); err != nil {
		errors = append(errors, fmt.Errorf("memory limits: %w", err))
	}

	// 应用CPU限制
	if err := c.applyCPULimits(); err != nil {
		errors = append(errors, fmt.Errorf("CPU limits: %w", err))
	}

	// 应用进程数限制
	if err := c.applyPIDsLimits(); err != nil {
		errors = append(errors, fmt.Errorf("PIDs limits: %w", err))
	}

	// 应用I/O限制
	if err := c.applyBlkIOLimits(); err != nil {
		errors = append(errors, fmt.Errorf("BlkIO limits: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to apply some limits: %v", errors)
	}

	return nil
}

// 应用内存限制
func (c *CgroupManager) applyMemoryLimits() error {
	memoryPath := c.groupPaths[CgroupMemory]

	// 设置内存限制
	if c.config.MemoryLimitBytes > 0 {
		limitFile := filepath.Join(memoryPath, "memory.limit_in_bytes")
		if err := c.writeFile(limitFile, strconv.FormatInt(c.config.MemoryLimitBytes, 10)); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
		logx.Debugf("Set memory limit: %d bytes", c.config.MemoryLimitBytes)
	}

	// 设置内存+swap限制
	if c.config.MemorySwapLimit > 0 {
		swapFile := filepath.Join(memoryPath, "memory.memsw.limit_in_bytes")
		if err := c.writeFile(swapFile, strconv.FormatInt(c.config.MemorySwapLimit, 10)); err != nil {
			// swap限制可能不支持，记录警告但不失败
			logx.Errorf("Failed to set swap limit (may not be supported): %v", err)
		} else {
			logx.Debugf("Set memory+swap limit: %d bytes", c.config.MemorySwapLimit)
		}
	}

	// 禁用OOM杀死（如果配置）
	if c.config.MemoryOOMKillDisable {
		oomFile := filepath.Join(memoryPath, "memory.oom_control")
		if err := c.writeFile(oomFile, "1"); err != nil {
			logx.Errorf("Failed to disable OOM kill: %v", err)
		}
	}

	return nil
}

// 应用CPU限制
func (c *CgroupManager) applyCPULimits() error {
	cpuPath := c.groupPaths[CgroupCPU]

	// 设置CPU配额和周期
	if c.config.CPUQuotaUs > 0 {
		quotaFile := filepath.Join(cpuPath, "cpu.cfs_quota_us")
		if err := c.writeFile(quotaFile, strconv.FormatInt(c.config.CPUQuotaUs, 10)); err != nil {
			return fmt.Errorf("failed to set CPU quota: %w", err)
		}
		logx.Debugf("Set CPU quota: %d us", c.config.CPUQuotaUs)
	}

	if c.config.CPUPeriodUs > 0 {
		periodFile := filepath.Join(cpuPath, "cpu.cfs_period_us")
		if err := c.writeFile(periodFile, strconv.FormatInt(c.config.CPUPeriodUs, 10)); err != nil {
			return fmt.Errorf("failed to set CPU period: %w", err)
		}
		logx.Debugf("Set CPU period: %d us", c.config.CPUPeriodUs)
	}

	// 设置CPU权重
	if c.config.CPUShares > 0 {
		sharesFile := filepath.Join(cpuPath, "cpu.shares")
		if err := c.writeFile(sharesFile, strconv.FormatInt(c.config.CPUShares, 10)); err != nil {
			return fmt.Errorf("failed to set CPU shares: %w", err)
		}
		logx.Debugf("Set CPU shares: %d", c.config.CPUShares)
	}

	// 设置CPU核心绑定
	if c.config.CPUSetCPUs != "" {
		cpusetPath := c.groupPaths[CgroupCPUSet]
		cpusFile := filepath.Join(cpusetPath, "cpuset.cpus")
		if err := c.writeFile(cpusFile, c.config.CPUSetCPUs); err != nil {
			return fmt.Errorf("failed to set cpuset.cpus: %w", err)
		}

		// 同时需要设置cpuset.mems（内存节点）
		memsFile := filepath.Join(cpusetPath, "cpuset.mems")
		if err := c.writeFile(memsFile, "0"); err != nil {
			logx.Errorf("Failed to set cpuset.mems: %v", err)
		}

		logx.Debugf("Set CPU set: %s", c.config.CPUSetCPUs)
	}

	return nil
}

// 应用进程数限制
func (c *CgroupManager) applyPIDsLimits() error {
	if c.config.PIDsMax <= 0 {
		return nil
	}

	pidsPath := c.groupPaths[CgroupPIDs]
	maxFile := filepath.Join(pidsPath, "pids.max")

	if err := c.writeFile(maxFile, strconv.FormatInt(c.config.PIDsMax, 10)); err != nil {
		return fmt.Errorf("failed to set pids.max: %w", err)
	}

	logx.Debugf("Set PIDs max: %d", c.config.PIDsMax)
	return nil
}

// 应用I/O限制
func (c *CgroupManager) applyBlkIOLimits() error {
	blkioPath := c.groupPaths[CgroupBlkIO]

	// 设置I/O权重
	if c.config.BlkIOWeight > 0 {
		weightFile := filepath.Join(blkioPath, "blkio.weight")
		if err := c.writeFile(weightFile, strconv.FormatInt(c.config.BlkIOWeight, 10)); err != nil {
			return fmt.Errorf("failed to set blkio.weight: %w", err)
		}
		logx.Debugf("Set BlkIO weight: %d", c.config.BlkIOWeight)
	}

	// 设置读取带宽限制（需要指定设备）
	if c.config.BlkIOReadBps > 0 {
		// 这里简化处理，实际应该获取具体的块设备号
		// 格式: "major:minor bytes_per_second"
		// 示例: "8:0 1048576" 表示设备8:0限制为1MB/s
		logx.Debugf("BlkIO read BPS limit configured: %d (device-specific setup required)", c.config.BlkIOReadBps)
	}

	if c.config.BlkIOWriteBps > 0 {
		logx.Debugf("BlkIO write BPS limit configured: %d (device-specific setup required)", c.config.BlkIOWriteBps)
	}

	return nil
}

// AddProcess 将进程添加到cgroup
func (c *CgroupManager) AddProcess(pid int) error {
	if !c.created {
		return fmt.Errorf("cgroup not created")
	}

	logx.Infof("Adding process %d to cgroup: %s", pid, c.config.GroupName)

	// 将进程添加到各个子系统的cgroup中
	for subsystem, path := range c.groupPaths {
		procsFile := filepath.Join(path, "cgroup.procs")
		if err := c.writeFile(procsFile, strconv.Itoa(pid)); err != nil {
			return fmt.Errorf("failed to add process to %s cgroup: %w", subsystem, err)
		}
		logx.Debugf("Added process %d to %s cgroup", pid, subsystem)
	}

	logx.Infof("Successfully added process %d to cgroup", pid)
	return nil
}

// GetStats 获取cgroup统计信息
func (c *CgroupManager) GetStats() (*CgroupStats, error) {
	if !c.created {
		return nil, fmt.Errorf("cgroup not created")
	}

	stats := &CgroupStats{}

	// 获取内存统计
	if err := c.getMemoryStats(stats); err != nil {
		logx.Errorf("Failed to get memory stats: %v", err)
	}

	// 获取CPU统计
	if err := c.getCPUStats(stats); err != nil {
		logx.Errorf("Failed to get CPU stats: %v", err)
	}

	// 获取进程统计
	if err := c.getPIDsStats(stats); err != nil {
		logx.Errorf("Failed to get PIDs stats: %v", err)
	}

	// 获取I/O统计
	if err := c.getBlkIOStats(stats); err != nil {
		logx.Errorf("Failed to get BlkIO stats: %v", err)
	}

	return stats, nil
}

// 获取内存统计信息
func (c *CgroupManager) getMemoryStats(stats *CgroupStats) error {
	memoryPath := c.groupPaths[CgroupMemory]

	// 当前内存使用量
	if usage, err := c.readInt64File(filepath.Join(memoryPath, "memory.usage_in_bytes")); err == nil {
		stats.MemoryUsage = usage
	}

	// 内存使用峰值
	if maxUsage, err := c.readInt64File(filepath.Join(memoryPath, "memory.max_usage_in_bytes")); err == nil {
		stats.MemoryMaxUsage = maxUsage
	}

	// 内存限制
	if limit, err := c.readInt64File(filepath.Join(memoryPath, "memory.limit_in_bytes")); err == nil {
		stats.MemoryLimit = limit
	}

	// OOM事件次数
	if oomCount, err := c.readMemoryOOMCount(memoryPath); err == nil {
		stats.MemoryOOMCount = oomCount
	}

	return nil
}

// 获取CPU统计信息
func (c *CgroupManager) getCPUStats(stats *CgroupStats) error {
	cpuPath := c.groupPaths[CgroupCPU]

	// CPU总使用时间
	if usage, err := c.readInt64File(filepath.Join(cpuPath, "cpuacct.usage")); err == nil {
		stats.CPUUsageTotal = usage
	}

	// 用户态和内核态时间
	if userSystem, err := c.readCPUUserSystem(cpuPath); err == nil {
		stats.CPUUsageUser = userSystem[0]
		stats.CPUUsageSystem = userSystem[1]
	}

	// CPU限流统计
	if throttled, err := c.readCPUThrottled(cpuPath); err == nil {
		stats.CPUThrottled = throttled
	}

	// 计算CPU使用率（需要两次采样）
	// 这里简化处理，实际应该基于时间差计算
	stats.CPUUsagePercent = 0.0

	return nil
}

// 获取进程数统计信息
func (c *CgroupManager) getPIDsStats(stats *CgroupStats) error {
	pidsPath := c.groupPaths[CgroupPIDs]

	// 当前进程数
	if current, err := c.readInt64File(filepath.Join(pidsPath, "pids.current")); err == nil {
		stats.PIDsCurrent = current
	}

	// 最大进程数限制
	if max, err := c.readInt64File(filepath.Join(pidsPath, "pids.max")); err == nil {
		stats.PIDsMax = max
	}

	return nil
}

// 获取I/O统计信息
func (c *CgroupManager) getBlkIOStats(stats *CgroupStats) error {
	blkioPath := c.groupPaths[CgroupBlkIO]

	// 读取I/O统计（简化处理）
	if ioStats, err := c.readBlkIOStats(blkioPath); err == nil {
		stats.BlkIOReadBytes = ioStats["read_bytes"]
		stats.BlkIOWriteBytes = ioStats["write_bytes"]
		stats.BlkIOReadOps = ioStats["read_ops"]
		stats.BlkIOWriteOps = ioStats["write_ops"]
	}

	return nil
}

// Cleanup 清理cgroup控制组
func (c *CgroupManager) Cleanup() error {
	if !c.created {
		return nil // 已经清理或未创建
	}

	logx.Infof("Cleaning up cgroup: %s", c.config.GroupName)

	// 等待所有进程退出
	if err := c.waitForProcessesExit(); err != nil {
		logx.Errorf("Some processes may still be running: %v", err)
	}

	// 删除各子系统的控制组目录
	var errors []error
	for subsystem, path := range c.groupPaths {
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove %s cgroup: %w", subsystem, err))
		} else {
			logx.Debugf("Removed cgroup directory: %s", path)
		}
	}

	c.created = false

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	logx.Infof("Successfully cleaned up cgroup: %s", c.config.GroupName)
	return nil
}

// 等待进程退出
func (c *CgroupManager) waitForProcessesExit() error {
	maxWait := 5 * time.Second
	checkInterval := 100 * time.Millisecond
	timeout := time.Now().Add(maxWait)

	for time.Now().Before(timeout) {
		allEmpty := true

		for subsystem, path := range c.groupPaths {
			procsFile := filepath.Join(path, "cgroup.procs")
			if content, err := c.readFile(procsFile); err == nil {
				if strings.TrimSpace(content) != "" {
					allEmpty = false
					logx.Debugf("Processes still running in %s cgroup", subsystem)
					break
				}
			}
		}

		if allEmpty {
			return nil
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("timeout waiting for processes to exit")
}

// 辅助函数：写入文件
func (c *CgroupManager) writeFile(path, content string) error {
	return ioutil.WriteFile(path, []byte(content), 0644)
}

// 辅助函数：读取文件
func (c *CgroupManager) readFile(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// 辅助函数：读取int64文件
func (c *CgroupManager) readInt64File(path string) (int64, error) {
	content, err := c.readFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(content, 10, 64)
}

// 读取内存OOM计数
func (c *CgroupManager) readMemoryOOMCount(memoryPath string) (int64, error) {
	oomControlFile := filepath.Join(memoryPath, "memory.oom_control")
	content, err := c.readFile(oomControlFile)
	if err != nil {
		return 0, err
	}

	// 解析oom_control文件内容
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "oom_kill ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return strconv.ParseInt(parts[1], 10, 64)
			}
		}
	}

	return 0, fmt.Errorf("oom_kill count not found")
}

// 读取CPU用户态和内核态时间
func (c *CgroupManager) readCPUUserSystem(cpuPath string) ([2]int64, error) {
	statFile := filepath.Join(cpuPath, "cpuacct.stat")
	content, err := c.readFile(statFile)
	if err != nil {
		return [2]int64{}, err
	}

	var user, system int64
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			value, _ := strconv.ParseInt(fields[1], 10, 64)
			switch fields[0] {
			case "user":
				user = value
			case "system":
				system = value
			}
		}
	}

	return [2]int64{user, system}, nil
}

// 读取CPU限流统计
func (c *CgroupManager) readCPUThrottled(cpuPath string) (int64, error) {
	statFile := filepath.Join(cpuPath, "cpu.stat")
	content, err := c.readFile(statFile)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "nr_throttled ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return strconv.ParseInt(parts[1], 10, 64)
			}
		}
	}

	return 0, nil
}

// 读取I/O统计信息
func (c *CgroupManager) readBlkIOStats(blkioPath string) (map[string]int64, error) {
	stats := make(map[string]int64)

	// 读取I/O字节统计
	bytesFile := filepath.Join(blkioPath, "blkio.throttle.io_service_bytes")
	if content, err := c.readFile(bytesFile); err == nil {
		c.parseBlkIOStats(content, stats, "bytes")
	}

	// 读取I/O操作统计
	opsFile := filepath.Join(blkioPath, "blkio.throttle.io_serviced")
	if content, err := c.readFile(opsFile); err == nil {
		c.parseBlkIOStats(content, stats, "ops")
	}

	return stats, nil
}

// 解析I/O统计信息
func (c *CgroupManager) parseBlkIOStats(content string, stats map[string]int64, suffix string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			operation := strings.ToLower(fields[1])
			value, err := strconv.ParseInt(fields[2], 10, 64)
			if err != nil {
				continue
			}

			key := fmt.Sprintf("%s_%s", operation, suffix)
			stats[key] += value
		}
	}
}

// 清理部分创建的资源
func (c *CgroupManager) cleanupPartialCreation() {
	for _, path := range c.groupPaths {
		os.RemoveAll(path)
	}
}

// IsProcessInGroup 检查进程是否在指定的cgroup中
func (c *CgroupManager) IsProcessInGroup(pid int) bool {
	if !c.created {
		return false
	}

	// 检查进程是否在memory cgroup中（代表性检查）
	memoryPath := c.groupPaths[CgroupMemory]
	procsFile := filepath.Join(memoryPath, "cgroup.procs")

	content, err := c.readFile(procsFile)
	if err != nil {
		return false
	}

	pidStr := strconv.Itoa(pid)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == pidStr {
			return true
		}
	}

	return false
}
