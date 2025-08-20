package logic

import (
	"context"

	"github.com/online-judge/code-judger/services/user-rpc/internal/svc"
	"github.com/online-judge/code-judger/services/user-rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVerifyUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyUserLogic {
	return &VerifyUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *VerifyUserLogic) VerifyUser(in *pb.VerifyUserReq) (*pb.VerifyUserResp, error) {
	// TODO: 实现用户验证逻辑
	success := in.Username != "" && in.Password != ""
	
	var user *pb.UserInfo
	if success {
		user = &pb.UserInfo{
			UserId:   1001,
			Username: in.Username,
			Email:    "test@example.com",
			Role:     "user",
			Status:   "active",
		}
	}

	return &pb.VerifyUserResp{
		Success: success,
		User:    user,
	}, nil
}
