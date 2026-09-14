package control

import (
	"go.yorun.ai/vine/buildinfo"
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
		Version:    buildinfo.MustVineVersion(),
		ApiPort:    s.Flag.ControlPort(),
		RedisPort:  s.Flag.WatchPort(), // Keep the same port for older Link and Portal clients.
		WatchPort:  s.Flag.WatchPort(),
		NatsPort:   natsPort,
		MqEndpoint: mqEndpoint,
	}
}
