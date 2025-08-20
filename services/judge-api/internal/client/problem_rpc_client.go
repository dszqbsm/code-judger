package client

import (
	"context"
	"fmt"
	"time"

	"github.com/online-judge/code-judger/services/judge-api/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// ProblemRpc客户端接口（这里需要引入生成的pb包）
// 注意：实际使用时需要先生成protobuf代码
type ProblemRpcClient interface {
	GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error)
}

// Go-Zero RPC客户端实现
type ZeroRpcProblemClient struct {
	rpcClient zrpc.Client
}

func NewZeroRpcProblemClient(endpoint string, timeout time.Duration) ProblemServiceClient {
	// 创建zRPC客户端配置
	conf := zrpc.RpcClientConf{
		Endpoints: []string{endpoint},
		Timeout:   int64(timeout / time.Millisecond),
	}

	// 创建RPC客户端
	rpcClient := zrpc.MustNewClient(conf)

	return &ZeroRpcProblemClient{
		rpcClient: rpcClient,
	}
}

func (c *ZeroRpcProblemClient) GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error) {
	logx.WithContext(ctx).Infof("Calling problem RPC service for problem_id=%d", problemId)

	startTime := time.Now()

	// 这里需要调用生成的RPC客户端
	// 由于protobuf代码还未生成，这里提供一个示例实现

	// 实际代码应该是：
	// req := &pb.GetProblemDetailReq{ProblemId: problemId}
	// resp, err := c.problemClient.GetProblemDetail(ctx, req)
	// if err != nil {
	//     return nil, fmt.Errorf("RPC调用失败: %w", err)
	// }
	// return convertProblemInfo(resp.Problem), nil

	// 模拟RPC调用（实际开发中删除）
	err := c.simulateRpcCall(ctx, problemId)
	if err != nil {
		return nil, err
	}

	duration := time.Since(startTime)
	logx.WithContext(ctx).Infof("Problem RPC call completed in %v", duration)

	// 返回模拟数据（实际开发中替换为真实转换）
	return &types.ProblemInfo{
		ProblemId:   problemId,
		Title:       fmt.Sprintf("RPC获取的题目 %d", problemId),
		TimeLimit:   1500, // 1.5秒
		MemoryLimit: 256,  // 256MB
		Languages:   []string{"cpp", "c", "java", "python", "go"},
		TestCases: []types.TestCase{
			{
				CaseId:         1,
				Input:          "1 2",
				ExpectedOutput: "3",
			},
			{
				CaseId:         2,
				Input:          "5 10",
				ExpectedOutput: "15",
			},
		},
		IsPublic: true,
	}, nil
}

func (c *ZeroRpcProblemClient) simulateRpcCall(ctx context.Context, problemId int64) error {
	// 模拟RPC调用延迟
	time.Sleep(20 * time.Millisecond) // RPC通常比HTTP快很多

	// 模拟题目不存在
	if problemId <= 0 || problemId > 10000 {
		return fmt.Errorf("题目不存在: %d", problemId)
	}

	// 模拟网络错误
	if problemId == 999 {
		return fmt.Errorf("RPC调用超时")
	}

	return nil
}

// 原生gRPC客户端实现（如果不使用go-zero的zRPC）
type GrpcProblemClient struct {
	conn   *grpc.ClientConn
	client interface{} // 这里应该是生成的pb.ProblemClient
}

func NewGrpcProblemClient(endpoint string, timeout time.Duration) (ProblemServiceClient, error) {
	// 创建gRPC连接
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithInsecure(), // 生产环境应该使用TLS
		grpc.WithBlock(),
		grpc.WithTimeout(timeout),
	)
	if err != nil {
		return nil, fmt.Errorf("连接RPC服务失败: %w", err)
	}

	// 创建RPC客户端
	// client := pb.NewProblemClient(conn)

	return &GrpcProblemClient{
		conn:   conn,
		client: nil, // pb客户端
	}, nil
}

func (c *GrpcProblemClient) GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error) {
	// 实现gRPC调用
	logx.WithContext(ctx).Infof("Calling gRPC problem service for problem_id=%d", problemId)

	// 这里是示例，实际需要调用生成的protobuf客户端
	time.Sleep(25 * time.Millisecond) // 模拟gRPC调用

	if problemId <= 0 {
		return nil, fmt.Errorf("题目不存在: %d", problemId)
	}

	return &types.ProblemInfo{
		ProblemId:   problemId,
		Title:       fmt.Sprintf("gRPC获取的题目 %d", problemId),
		TimeLimit:   1200,
		MemoryLimit: 256,
		Languages:   []string{"cpp", "java", "python"},
		TestCases: []types.TestCase{
			{
				CaseId:         1,
				Input:          "test input",
				ExpectedOutput: "test output",
			},
		},
		IsPublic: true,
	}, nil
}

// 类型转换函数（protobuf -> 内部类型）
func convertProblemInfo(pbProblem interface{}) *types.ProblemInfo {
	// 这里需要根据实际生成的protobuf类型进行转换
	// 例如：
	// return &types.ProblemInfo{
	//     ProblemId:   pbProblem.ProblemId,
	//     Title:       pbProblem.Title,
	//     TimeLimit:   int(pbProblem.TimeLimit),
	//     MemoryLimit: int(pbProblem.MemoryLimit),
	//     Languages:   pbProblem.Languages,
	//     TestCases:   convertTestCases(pbProblem.TestCases),
	//     IsPublic:    pbProblem.IsPublic,
	// }
	return nil
}

func convertTestCases(pbTestCases interface{}) []types.TestCase {
	// 转换测试用例
	return nil
}
