package logic

import (
	"context"

	"github.com/online-judge/code-judger/services/user-rpc/internal/svc"
	"github.com/online-judge/code-judger/services/user-rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLogic {
	return &GetUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserLogic) GetUser(in *pb.GetUserReq) (*pb.GetUserResp, error) {
	// TODO: 实现获取用户信息逻辑
	user := &pb.UserInfo{
		UserId:   in.UserId,
		Username: "test_user",
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	}

	return &pb.GetUserResp{
		User: user,
	}, nil
}
