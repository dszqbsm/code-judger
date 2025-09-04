package server

import (
	"context"

	"github.com/dszqbsm/code-judger/services/user-rpc/internal/logic"
	"github.com/dszqbsm/code-judger/services/user-rpc/internal/svc"
	"github.com/dszqbsm/code-judger/services/user-rpc/pb"
)

type UserServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedUserServer
}

func NewUserServer(svcCtx *svc.ServiceContext) *UserServer {
	return &UserServer{
		svcCtx: svcCtx,
	}
}

func (s *UserServer) GetUser(ctx context.Context, in *pb.GetUserReq) (*pb.GetUserResp, error) {
	l := logic.NewGetUserLogic(ctx, s.svcCtx)
	return l.GetUser(in)
}

func (s *UserServer) GetUsers(ctx context.Context, in *pb.GetUsersReq) (*pb.GetUsersResp, error) {
	l := logic.NewGetUsersLogic(ctx, s.svcCtx)
	return l.GetUsers(in)
}

func (s *UserServer) VerifyUser(ctx context.Context, in *pb.VerifyUserReq) (*pb.VerifyUserResp, error) {
	l := logic.NewVerifyUserLogic(ctx, s.svcCtx)
	return l.VerifyUser(in)
}
