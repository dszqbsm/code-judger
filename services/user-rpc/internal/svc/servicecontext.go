package svc

import (
	"github.com/dszqbsm/code-judger/services/user-rpc/internal/config"
)

type ServiceContext struct {
	Config config.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
