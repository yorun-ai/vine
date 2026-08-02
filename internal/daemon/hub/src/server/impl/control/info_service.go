package control

import (
	"go.yorun.ai/vine/internal/app"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

type InfoServiceServerImpl struct {
	skeled.DefaultInfoServiceServer

	InprocFlag *app.InternalInprocFlag `inject:""`
	Flag       *flag.Flag              `inject:""`
	NATSServer *natsserver.NATSServer  `inject:""`
}

func (s *InfoServiceServerImpl) GetInfo() skeled.Info {
	natsPort := 0
	mqEndpoint := s.Flag.MQExternalNatsURL
	if s.Flag.MQEmbeddedNats && !s.InprocFlag.Enabled {
		natsPort = s.NATSServer.Port()
		mqEndpoint = ""
	}
	return skeled.Info{
		ApiPort:    s.Flag.ControlPort(),
		RedisPort:  s.Flag.RedisPort(),
		NatsPort:   natsPort,
		MqEndpoint: mqEndpoint,
	}
}
