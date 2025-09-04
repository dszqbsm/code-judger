package problem

import (
	"context"
	"testing"

	"github.com/dszqbsm/code-judger/services/problem-api/internal/types"

	"github.com/stretchr/testify/assert"
)

func TestCreateProblemLogic_CreateProblem(t *testing.T) {
	// 这里使用内存数据库或Mock，实际测试中不连接真实数据库
	// c := config.Config{}
	// svcCtx := svc.NewServiceContext(c)

	tests := []struct {
		name    string
		req     *types.CreateProblemReq
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid problem creation",
			req: &types.CreateProblemReq{
				Title:        "测试题目",
				Description:  "这是一个测试题目的描述，需要至少10个字符",
				InputFormat:  "输入格式说明",
				OutputFormat: "输出格式说明",
				SampleInput:  "样例输入",
				SampleOutput: "样例输出",
				Difficulty:   "easy",
				TimeLimit:    1000,
				MemoryLimit:  128,
				Languages:    []string{"cpp", "java", "python"},
				Tags:         []string{"array", "sorting"},
				IsPublic:     true,
			},
			wantErr: false,
		},
		{
			name: "invalid title - too short",
			req: &types.CreateProblemReq{
				Title:        "",
				Description:  "这是一个测试题目的描述，需要至少10个字符",
				InputFormat:  "输入格式说明",
				OutputFormat: "输出格式说明",
				SampleInput:  "样例输入",
				SampleOutput: "样例输出",
				Difficulty:   "easy",
				TimeLimit:    1000,
				MemoryLimit:  128,
				Languages:    []string{"cpp"},
				Tags:         []string{"array"},
				IsPublic:     true,
			},
			wantErr: true,
			errMsg:  "题目标题长度必须在1-200字符之间",
		},
		{
			name: "invalid description - too short",
			req: &types.CreateProblemReq{
				Title:        "测试题目",
				Description:  "太短",
				InputFormat:  "输入格式说明",
				OutputFormat: "输出格式说明",
				SampleInput:  "样例输入",
				SampleOutput: "样例输出",
				Difficulty:   "easy",
				TimeLimit:    1000,
				MemoryLimit:  128,
				Languages:    []string{"cpp"},
				Tags:         []string{"array"},
				IsPublic:     true,
			},
			wantErr: true,
			errMsg:  "题目描述至少需要10个字符",
		},
		{
			name: "invalid difficulty",
			req: &types.CreateProblemReq{
				Title:        "测试题目",
				Description:  "这是一个测试题目的描述，需要至少10个字符",
				InputFormat:  "输入格式说明",
				OutputFormat: "输出格式说明",
				SampleInput:  "样例输入",
				SampleOutput: "样例输出",
				Difficulty:   "invalid",
				TimeLimit:    1000,
				MemoryLimit:  128,
				Languages:    []string{"cpp"},
				Tags:         []string{"array"},
				IsPublic:     true,
			},
			wantErr: true,
			errMsg:  "无效的难度级别: invalid",
		},
		{
			name: "invalid time limit - too low",
			req: &types.CreateProblemReq{
				Title:        "测试题目",
				Description:  "这是一个测试题目的描述，需要至少10个字符",
				InputFormat:  "输入格式说明",
				OutputFormat: "输出格式说明",
				SampleInput:  "样例输入",
				SampleOutput: "样例输出",
				Difficulty:   "easy",
				TimeLimit:    50,
				MemoryLimit:  128,
				Languages:    []string{"cpp"},
				Tags:         []string{"array"},
				IsPublic:     true,
			},
			wantErr: true,
			errMsg:  "时间限制必须在100-10000毫秒之间",
		},
		{
			name: "invalid memory limit - too low",
			req: &types.CreateProblemReq{
				Title:        "测试题目",
				Description:  "这是一个测试题目的描述，需要至少10个字符",
				InputFormat:  "输入格式说明",
				OutputFormat: "输出格式说明",
				SampleInput:  "样例输入",
				SampleOutput: "样例输出",
				Difficulty:   "easy",
				TimeLimit:    1000,
				MemoryLimit:  8,
				Languages:    []string{"cpp"},
				Tags:         []string{"array"},
				IsPublic:     true,
			},
			wantErr: true,
			errMsg:  "内存限制必须在16-512MB之间",
		},
		{
			name: "no languages specified",
			req: &types.CreateProblemReq{
				Title:        "测试题目",
				Description:  "这是一个测试题目的描述，需要至少10个字符",
				InputFormat:  "输入格式说明",
				OutputFormat: "输出格式说明",
				SampleInput:  "样例输入",
				SampleOutput: "样例输出",
				Difficulty:   "easy",
				TimeLimit:    1000,
				MemoryLimit:  128,
				Languages:    []string{},
				Tags:         []string{"array"},
				IsPublic:     true,
			},
			wantErr: true,
			errMsg:  "至少需要指定一种编程语言",
		},
		{
			name: "too many tags",
			req: &types.CreateProblemReq{
				Title:        "测试题目",
				Description:  "这是一个测试题目的描述，需要至少10个字符",
				InputFormat:  "输入格式说明",
				OutputFormat: "输出格式说明",
				SampleInput:  "样例输入",
				SampleOutput: "样例输出",
				Difficulty:   "easy",
				TimeLimit:    1000,
				MemoryLimit:  128,
				Languages:    []string{"cpp"},
				Tags:         []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7", "tag8", "tag9", "tag10", "tag11"},
				IsPublic:     true,
			},
			wantErr: true,
			errMsg:  "标签数量不能超过10个",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建logic实例
			logic := &CreateProblemLogic{
				ctx: context.Background(),
				// svcCtx: svcCtx, // 在实际测试中使用Mock
			}

			// 测试验证函数
			err := logic.validateRequest(tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
