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
	mqEndpoint := s.Flag.MQNatsEndpoint
	if s.Flag.MQMode == flag.MQModeEmbedded && !s.InprocFlag.Enabled {
		natsPort = s.NATSServer.Port()
		mqEndpoint = ""
	}
	return skeled.Info{
		Version:           buildinfo.MustVineVersion(),
		ApiPort:           s.Flag.ControlPort(),
		WatchPort:         s.Flag.WatchPort(),
		MqEmbedded:        s.Flag.MQMode == flag.MQModeEmbedded,
		MqNatsPort:        natsPort,
		MqNatsEndpoint:    mqEndpoint,
		LockMode:          s.Flag.LockMode,
		LockRedisEndpoint: s.Flag.LockRedisEndpoint,
		RedisPort:         s.Flag.WatchPort(), // Keep the same port for older Link and Portal clients.
		NatsPort:          natsPort,
		MqEndpoint:        mqEndpoint,
	}
}
