package logic

import (
	"context"

	"github.com/online-judge/code-judger/services/user-rpc/internal/svc"
	"github.com/online-judge/code-judger/services/user-rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUsersLogic {
	return &GetUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUsersLogic) GetUsers(in *pb.GetUsersReq) (*pb.GetUsersResp, error) {
	// TODO: 实现批量获取用户信息逻辑
	var users []*pb.UserInfo
	for _, userId := range in.UserIds {
		user := &pb.UserInfo{
			UserId:   userId,
			Username: "test_user",
			Email:    "test@example.com",
			Role:     "user",
			Status:   "active",
		}
		users = append(users, user)
	}

	return &pb.GetUsersResp{
		Users: users,
	}, nil
}
