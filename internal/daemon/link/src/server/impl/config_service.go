package impl

import (
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/config"
)

type ConfigServiceServerImpl struct {
	skeled.DefaultConfigServiceServer

	Context spec.Context   `inject:""`
	Reader  *config.Reader `inject:""`
}

func (s *ConfigServiceServerImpl) GetEternal(key string) string {
	return s.Reader.GetEternal(s.Context.Client().InstanceId(), key)
}

func (s *ConfigServiceServerImpl) GetInstant(key string) string {
	return s.Reader.GetInstant(s.Context.Client().InstanceId(), key)
}
