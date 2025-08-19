# 判题服务 (Judge API)

基于go-zero框架开发的高性能在线判题系统核心服务，采用系统调用安全沙箱技术，支持多种编程语言的代码执行和判题。

## 🚀 特性

### 核心功能
- **高性能判题引擎**: 基于系统调用的安全沙箱，进程启动时间 < 10ms
- **多语言支持**: C/C++、Java、Python、Go、JavaScript等主流编程语言
- **任务调度系统**: 支持优先级队列、并发执行、任务重试机制
- **实时状态监控**: WebSocket实时推送判题状态和结果
- **安全沙箱隔离**: 五层安全防护，防止恶意代码攻击

### 技术特色
- **系统调用沙箱**: 采用fork + chroot + seccomp + ptrace组合方案
- **精确资源控制**: 毫秒级时间监控，KB级内存统计
- **高并发处理**: 支持5000+并发判题任务
- **微服务架构**: 基于go-zero框架，支持水平扩展

## 📋 系统要求

### 基础环境
- **操作系统**: Linux (内核版本 >= 4.0)
- **Go版本**: Go 1.21+
- **内存**: 最小2GB，推荐4GB+
- **CPU**: 最小2核，推荐4核+
- **磁盘**: 最小10GB可用空间

### 依赖服务
- **MySQL 8.0+**: 数据存储
- **Redis 6.0+**: 缓存和会话管理
- **Kafka**: 消息队列（可选）

### 编译器环境
```bash
# C/C++
sudo apt-get install gcc g++ build-essential

# Java
sudo apt-get install openjdk-11-jdk

# Python
sudo apt-get install python3 python3-pip

# Go
wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Node.js
curl -fsSL https://deb.nodesource.com/setup_16.x | sudo -E bash -
sudo apt-get install -y nodejs
```

## 🛠️ 快速开始

### 1. 克隆项目
```bash
git clone <项目地址>
cd code-judger/services/judge-api
```

### 2. 配置服务
```bash
# 复制配置文件
cp etc/judge-api.yaml.example etc/judge-api.yaml

# 编辑配置文件
vim etc/judge-api.yaml
```

### 3. 启动服务

#### 开发环境
```bash
# 使用启动脚本（推荐）
./scripts/start-judge-api.sh dev

# 或手动启动
go run main.go -f etc/judge-api.yaml
```

#### 生产环境
```bash
# 后台运行
./scripts/start-judge-api.sh prod

# 查看日志
tail -f /var/log/judge-api/judge-api.log

# 停止服务
pkill -f judge-api
```

### 4. 验证服务
```bash
# 健康检查
curl http://localhost:8889/api/v1/judge/health

# 查看支持的语言
curl http://localhost:8889/api/v1/judge/languages

# 查看队列状态
curl http://localhost:8889/api/v1/judge/queue
```

## 📚 API文档

### 核心接口

#### 提交判题任务
```bash
POST /api/v1/judge/submit
Content-Type: application/json

{
  "submission_id": 12345,
  "problem_id": 1001,
  "user_id": 2001,
  "language": "cpp",
  "code": "#include<iostream>\nusing namespace std;\nint main(){...}",
  "time_limit": 1000,
  "memory_limit": 128,
  "test_cases": [
    {
      "case_id": 1,
      "input": "3 4",
      "expected_output": "7"
    }
  ]
}
```

#### 查询判题结果
```bash
GET /api/v1/judge/result/{submission_id}
```

#### 查询判题状态
```bash
GET /api/v1/judge/status/{submission_id}
```

#### 取消判题任务
```bash
DELETE /api/v1/judge/cancel/{submission_id}
```

### 系统管理接口

#### 获取节点状态
```bash
GET /api/v1/judge/nodes
```

#### 获取队列状态
```bash
GET /api/v1/judge/queue
```

#### 健康检查
```bash
GET /api/v1/judge/health
```

#### 支持的语言
```bash
GET /api/v1/judge/languages
```

## ⚙️ 配置说明

### 基础配置
```yaml
Name: judge-api
Host: 0.0.0.0
Port: 8889
Timeout: 30000

# MySQL数据库配置
DataSource: oj_user:oj_password@tcp(mysql:3306)/oj_judge?charset=utf8mb4&parseTime=true

# Redis配置
RedisConf:
  Host: redis:6379
  Type: node
```

### 判题引擎配置
```yaml
JudgeEngine:
  # 工作目录配置
  WorkDir: /tmp/judge
  TempDir: /tmp/judge/temp
  DataDir: /tmp/judge/data
  
  # 沙箱配置
  Sandbox:
    EnableSeccomp: true
    EnableChroot: true
    EnablePtrace: true
    JailUser: "nobody"
    JailUID: 65534
    JailGID: 65534
```

### 语言配置
```yaml
Compilers:
  cpp:
    Name: "C++"
    Version: "g++ 9.4.0"
    FileExtension: ".cpp"
    CompileCommand: "g++ -o {executable} {source} -std=c++17 -O2"
    TimeMultiplier: 1.0
    MemoryMultiplier: 1.0
```

### 任务队列配置
```yaml
TaskQueue:
  MaxWorkers: 10              # 最大工作协程数
  QueueSize: 1000            # 队列大小
  TaskTimeout: 300           # 任务超时时间(秒)
  RetryTimes: 3              # 重试次数
```

## 🔒 安全机制

### 五层安全防护

1. **进程隔离**: fork子进程 + 权限降级 + PID命名空间
2. **系统调用过滤**: seccomp-bpf精确控制允许的系统调用
3. **文件系统隔离**: chroot监狱 + 只读文件系统
4. **网络隔离**: 网络命名空间隔离，完全断网
5. **资源限制**: rlimit + cgroups双重资源控制

### 系统调用白名单
```go
// C/C++允许的系统调用
var CppSyscalls = []int{
    0,   // read
    1,   // write  
    2,   // open
    3,   // close
    59,  // execve
    60,  // exit
    231, // exit_group
}
```

### 资源限制
- **时间限制**: 100ms - 10s
- **内存限制**: 16MB - 512MB  
- **文件大小**: 最大10MB
- **进程数量**: 根据语言特性限制

## 📊 性能指标

### 系统性能
- **进程启动时间**: < 10ms
- **并发能力**: 5000+ 并发任务
- **响应时间**: 简单程序判题 < 1秒
- **内存效率**: 仅程序本身内存占用
- **CPU效率**: 直接系统调用，无虚拟化损耗

### 语言性能倍数
| 语言 | 时间倍数 | 内存倍数 | 说明 |
|------|----------|----------|------|
| C/C++ | 1.0x | 1.0x | 原生性能 |
| Java | 2.0x | 2.0x | JVM启动开销 |
| Python | 3.0x | 1.5x | 解释执行较慢 |
| Go | 1.5x | 1.2x | 编译型语言 |
| JavaScript | 2.5x | 1.8x | V8引擎 |

## 🔧 故障排查

### 常见问题

#### 1. 编译失败
```bash
# 检查编译器是否安装
which gcc g++ javac python3 go node

# 检查编译器版本
gcc --version
g++ --version
javac -version
```

#### 2. 权限错误
```bash
# 检查工作目录权限
ls -la /tmp/judge

# 创建必要目录
sudo mkdir -p /tmp/judge/{temp,data}
sudo chmod 755 /tmp/judge
```

#### 3. 端口被占用
```bash
# 查看端口占用
lsof -i :8889

# 停止占用进程
pkill -f judge-api
```

#### 4. 内存不足
```bash
# 检查系统内存
free -h

# 检查进程内存使用
ps aux | grep judge-api
```

### 日志分析
```bash
# 查看服务日志
tail -f /var/log/judge-api/judge-api.log

# 查看系统日志
journalctl -u judge-api -f

# 查看错误日志
grep "ERROR" /var/log/judge-api/judge-api.log
```

## 📈 监控告警

### 关键指标
- **队列长度**: 等待判题的任务数量
- **执行成功率**: 成功执行的任务比例
- **平均执行时间**: 任务平均处理时间
- **系统资源使用**: CPU、内存、磁盘使用率

### Prometheus指标
```bash
# 访问指标端点
curl http://localhost:9091/metrics
```

### 告警规则
```yaml
# 队列积压告警
- alert: JudgeQueueTooLong
  expr: judge_queue_length > 100
  for: 5m
  
# 执行失败率告警  
- alert: JudgeFailureRateHigh
  expr: judge_failure_rate > 0.1
  for: 5m
```

## 🤝 开发指南

### 项目结构
```
judge-api/
├── internal/
│   ├── config/          # 配置结构
│   ├── handler/         # HTTP处理器
│   ├── logic/           # 业务逻辑
│   ├── svc/             # 服务上下文
│   ├── types/           # 类型定义
│   ├── sandbox/         # 安全沙箱
│   ├── languages/       # 语言执行器
│   ├── scheduler/       # 任务调度器
│   └── judge/           # 判题引擎
├── etc/                 # 配置文件
├── scripts/             # 启动脚本
└── main.go              # 服务入口
```

### 添加新语言支持

1. **在配置文件中添加语言配置**
```yaml
Compilers:
  rust:
    Name: "Rust"
    Version: "rustc 1.70.0"
    FileExtension: ".rs"
    CompileCommand: "rustc -o {executable} {source}"
    TimeMultiplier: 1.2
    MemoryMultiplier: 1.1
```

2. **实现语言执行器**
```go
type RustExecutor struct {
    *BaseLanguageExecutor
}

func NewRustExecutor(config config.CompilerConf) *RustExecutor {
    // 实现构造函数
}

func (e *RustExecutor) Compile(ctx context.Context, code string, workDir string) (*CompileResult, error) {
    // 实现编译逻辑
}
```

3. **注册执行器**
```go
// 在 NewLanguageManager 中添加
case "rust":
    manager.executors[lang] = NewRustExecutor(conf)
```

### 贡献代码
1. Fork 项目
2. 创建特性分支
3. 提交代码变更
4. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 支持

- **Issues**: [GitHub Issues](https://github.com/your-org/code-judger/issues)
- **讨论**: [GitHub Discussions](https://github.com/your-org/code-judger/discussions)
- **Wiki**: [项目Wiki](https://github.com/your-org/code-judger/wiki)

---

**注意**: 本服务需要Linux环境运行，Windows和macOS环境可能需要额外配置。生产环境建议使用Docker部署。
