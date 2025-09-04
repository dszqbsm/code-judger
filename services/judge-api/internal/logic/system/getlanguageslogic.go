package system

import (
	"context"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/languages"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLanguagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLanguagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLanguagesLogic {
	return &GetLanguagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetLanguagesLogic) GetLanguages(req *types.GetLanguagesReq) (resp *types.GetLanguagesResp, err error) {
	// 从判题引擎获取语言配置信息
	systemInfo := l.svcCtx.JudgeEngine.GetSystemInfo()
	languageConfigs, ok := systemInfo["language_configs"].([]languages.LanguageConfigInfo)
	if !ok {
		logx.Error("Failed to get language configs from judge engine")
		return &types.GetLanguagesResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "获取语言配置失败",
			},
		}, nil
	}

	// 转换为API响应格式
	apiLanguages := make([]types.LanguageConfig, len(languageConfigs))
	for i, config := range languageConfigs {
		// 从配置中获取编译和执行命令
		compilerConf, exists := l.svcCtx.Config.JudgeEngine.Compilers[config.Name]
		compileCommand := ""
		executeCommand := ""
		if exists {
			compileCommand = compilerConf.CompileCommand
			executeCommand = compilerConf.ExecuteCommand
		}

		apiLanguages[i] = types.LanguageConfig{
			Name:             config.Name,
			DisplayName:      config.DisplayName,
			Version:          config.Version,
			FileExtension:    config.FileExtension,
			CompileCommand:   compileCommand,
			ExecuteCommand:   executeCommand,
			TimeMultiplier:   config.TimeMultiplier,
			MemoryMultiplier: config.MemoryMultiplier,
			IsEnabled:        config.IsEnabled,
		}
	}

	logx.Infof("Retrieved %d supported languages", len(apiLanguages))

	return &types.GetLanguagesResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: types.LanguagesData{
			Languages: apiLanguages,
		},
	}, nil
}
