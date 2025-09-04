package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dszqbsm/code-judger/common/rpc"
)

func main() {
	fmt.Println("开始RPC调用测试...")

	// 测试参数
	consulAddr := "localhost:8500"
	timeout := 10 * time.Second

	// 测试判题服务RPC客户端
	fmt.Println("\n=== 测试判题服务RPC客户端 ===")
	testJudgeRPCClient(consulAddr, timeout)

	// 测试题目服务RPC客户端
	fmt.Println("\n=== 测试题目服务RPC客户端 ===")
	testProblemRPCClient(consulAddr, timeout)

	fmt.Println("\n测试完成!")
}

func testJudgeRPCClient(consulAddr string, timeout time.Duration) {
	// 创建判题服务RPC客户端
	judgeClient, err := rpc.NewHTTPRPCClient("judge-api", consulAddr, timeout)
	if err != nil {
		log.Printf("创建判题服务RPC客户端失败: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 测试获取判题结果
	fmt.Println("1. 测试获取判题结果...")
	submissionID := int64(1)
	path := fmt.Sprintf("/api/v1/judge/result/%d", submissionID)
	var result map[string]interface{}
	err = judgeClient.Get(ctx, path, &result)
	if err != nil {
		log.Printf("获取判题结果失败: %v", err)
	} else {
		fmt.Printf("判题结果: %+v\n", result)
	}

	fmt.Println("RPC客户端创建成功，可以进行服务调用")
}

func testProblemRPCClient(consulAddr string, timeout time.Duration) {
	// 创建题目服务RPC客户端
	problemClient, err := rpc.NewHTTPRPCClient("problem-api", consulAddr, timeout)
	if err != nil {
		log.Printf("创建题目服务RPC客户端失败: %v", err)
		return
	}

	fmt.Println("题目服务RPC客户端创建成功，可以进行服务调用")
	_ = problemClient // 避免未使用变量警告
}

// 模拟Consul服务发现测试
func testConsulServiceDiscovery() {
	fmt.Println("\n=== 测试Consul服务发现 ===")
	
	// 这里可以添加直接使用Consul API的测试代码
	// 验证服务注册和发现功能
}

// 模拟断路器测试
func testCircuitBreaker() {
	fmt.Println("\n=== 测试断路器功能 ===")
	
	// 这里可以添加断路器功能的测试代码
	// 模拟服务故障和恢复场景
}

// 模拟负载均衡测试
func testLoadBalancer() {
	fmt.Println("\n=== 测试负载均衡 ===")
	
	// 这里可以添加负载均衡功能的测试代码
	// 验证轮询、随机等负载均衡策略
}
