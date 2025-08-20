package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	DataSource string
	RedisConf  struct {
		Host string
		Type string
		Pass string
	}
	CacheConf cache.CacheConf
}
