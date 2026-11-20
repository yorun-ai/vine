package control

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/internal/app"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	vrpchttp "go.yorun.ai/vrpc/transport/http"
)

func runTestNATSServerDIInit(t *testing.T, server *natsserver.NATSServer) {
	t.Helper()

	previous := hubnats.InprocServer()
	t.Cleanup(func() {
		server.AfterAppStop()
		hubnats.SetInprocServer(previous)
	})
	server.DIInit()
}

func TestHubInfoServiceReturnsWatchPortAndExternalNATS(t *testing.T) {
	service := &InfoServiceServerImpl{
		InprocFlag: &app.InternalInprocFlag{},
		Flag: &flag.Flag{
			ControlListen:  ":7071",
			WatchListen:    ":7072",
			MQNatsEndpoint: "nats://127.0.0.1:4222",
		},
		NATSServer: &natsserver.NATSServer{},
	}

	info := service.GetInfo()

	assert.Equal(t, skeled.Info{
		Version:        buildinfo.MustVineVersion(),
		WatchPort:      7072,
		MqNatsEndpoint: "nats://127.0.0.1:4222",
	}, info)
}

func TestHubInfoServiceReturnsNATSServerPortWhenEnabled(t *testing.T) {
	service := &InfoServiceServerImpl{
		InprocFlag: &app.InternalInprocFlag{},
		Flag:       &flag.Flag{ControlListen: ":7071", WatchListen: ":7072", MQMode: flag.MQModeEmbedded},
		NATSServer: &natsserver.NATSServer{
			InprocFlag: &app.InternalInprocFlag{},
			Flag:       &flag.Flag{MQMode: flag.MQModeEmbedded},
		},
	}

	runTestNATSServerDIInit(t, service.NATSServer)

	info := service.GetInfo()

	assert.Equal(t, 7072, info.WatchPort)
	assert.True(t, info.MqEmbedded)
	assert.Equal(t, service.NATSServer.Port(), info.MqNatsPort)
	assert.Empty(t, info.MqNatsEndpoint)
}

func TestHubInfoServiceWireCompatibility(t *testing.T) {
	service := &InfoServiceServerImpl{
		InprocFlag: &app.InternalInprocFlag{},
		Flag:       &flag.Flag{ControlListen: ":7071", WatchListen: ":7072", MQNatsEndpoint: "nats://localhost:4222"},
	}
	info := service.GetInfo()
	for _, contentType := range []string{vrpchttp.ContentTypeJson, vrpchttp.ContentTypeCbor} {
		t.Run(contentType, func(t *testing.T) {
			body, err := vrpchttp.EncodeResponse(info, nil, contentType)
			require.NoError(t, err)
			payload, err := vrpchttp.DecodeResponse(body, contentType)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, payload.Unmarshal(payload.ResultBytes, &fields))
			assert.NotContains(t, fields, "apiPort")
			assert.NotContains(t, fields, "redisPort")
			assert.NotContains(t, fields, "natsPort")
			assert.NotContains(t, fields, "mqEndpoint")

			// The Watch and MQ fields used by v0.27.1 clients still decode.
			var legacy struct {
				WatchPort      int    `json:"watchPort"`
				MqNatsEndpoint string `json:"mqNatsEndpoint"`
			}
			require.NoError(t, payload.Unmarshal(payload.ResultBytes, &legacy))
			assert.Equal(t, 7072, legacy.WatchPort)
			assert.Equal(t, "nats://localhost:4222", legacy.MqNatsEndpoint)

			fields["apiPort"] = 7071
			fields["redisPort"] = info.WatchPort
			fields["natsPort"] = info.MqNatsPort
			fields["mqEndpoint"] = info.MqNatsEndpoint
			body, err = vrpchttp.EncodeResponse(fields, nil, contentType)
			require.NoError(t, err)
			payload, err = vrpchttp.DecodeResponse(body, contentType)
			require.NoError(t, err)
			var decoded skeled.Info
			require.NoError(t, payload.Unmarshal(payload.ResultBytes, &decoded))
			assert.Equal(t, info, decoded)
		})
	}
}

func TestHubInfoServiceInprocEmbeddedMQ(t *testing.T) {
	service := &InfoServiceServerImpl{
		InprocFlag: &app.InternalInprocFlag{Enabled: true},
		Flag:       &flag.Flag{ControlListen: ":7071", WatchListen: ":7072", MQMode: flag.MQModeEmbedded},
	}
	info := service.GetInfo()
	assert.True(t, info.MqEmbedded)
	assert.Zero(t, info.MqNatsPort)
	assert.Empty(t, info.MqNatsEndpoint)
}

func TestHubInfoAdvertisesLockConfiguration(t *testing.T) {
	for _, mode := range []string{"embedded", "redis", "disable"} {
		t.Run(mode, func(t *testing.T) {
			flags := &flag.Flag{ControlListen: ":7071", WatchListen: ":7072", MQMode: flag.MQModeNATS, LockMode: mode}
			if mode == "redis" {
				flags.LockRedisEndpoint = "rediss://user:secret@redis:6380/2"
			}
			service := &InfoServiceServerImpl{InprocFlag: &app.InternalInprocFlag{}, Flag: flags}
			info := service.GetInfo()
			assert.Equal(t, mode, info.LockMode)
			assert.Equal(t, flags.LockRedisEndpoint, info.LockRedisEndpoint)
		})
	}
}
