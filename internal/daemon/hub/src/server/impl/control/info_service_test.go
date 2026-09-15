package control

import (
	"go.yorun.ai/vine/buildinfo"
	"testing"

	"go.yorun.ai/vine/util/vcode"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/app"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func runTestNATSServerDIInit(t *testing.T, server *natsserver.NATSServer) {
	t.Helper()

	defer func() {
		if re := recover(); re != nil {
			t.Skipf("embedded nats server did not start in current environment: %v", re)
		}
	}()
	server.DIInit()
}

func TestHubInfoServiceReturnsPortsFromFlag(t *testing.T) {
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
		ApiPort:        7071,
		RedisPort:      7072,
		WatchPort:      7072,
		NatsPort:       0,
		MqEndpoint:     "nats://127.0.0.1:4222",
		MqNatsEndpoint: "nats://127.0.0.1:4222",
	}, info)
}

func TestHubInfoServiceReturnsNATSServerPortWhenEnabled(t *testing.T) {
	service := &InfoServiceServerImpl{
		InprocFlag: &app.InternalInprocFlag{},
		Flag: &flag.Flag{
			ControlListen: ":7071",
			WatchListen:   ":7072",
			MQMode:        flag.MQModeEmbedded,
		},
		NATSServer: &natsserver.NATSServer{
			InprocFlag: &app.InternalInprocFlag{},
			Flag:       &flag.Flag{MQMode: flag.MQModeEmbedded},
		},
	}

	runTestNATSServerDIInit(t, service.NATSServer)
	t.Cleanup(service.NATSServer.AfterAppStop)

	info := service.GetInfo()

	assert.Equal(t, 7071, info.ApiPort)
	assert.Equal(t, 7072, info.RedisPort)
	assert.Equal(t, info.RedisPort, info.WatchPort)
	assert.Equal(t, service.NATSServer.Port(), info.NatsPort)
	assert.Empty(t, info.MqEndpoint)
	assert.True(t, info.MqEmbedded)
	assert.Equal(t, info.NatsPort, info.MqNatsPort)
	assert.Empty(t, info.MqNatsEndpoint)
}

func TestHubInfoServicePreservesOldClientPort(t *testing.T) {
	service := &InfoServiceServerImpl{
		InprocFlag: &app.InternalInprocFlag{},
		Flag:       &flag.Flag{ControlListen: ":7071", WatchListen: ":7072", MQNatsEndpoint: "nats://localhost:4222"},
	}
	type oldInfo struct {
		ApiPort    int    `json:"apiPort"`
		RedisPort  int    `json:"redisPort"`
		NatsPort   int    `json:"natsPort"`
		MqEndpoint string `json:"mqEndpoint"`
	}
	info := service.GetInfo()
	decoded := vcode.MustUnmarshalJsonS[oldInfo](vcode.MustMarshalJsonS(info))
	assert.Equal(t, 7072, decoded.RedisPort)
	assert.Equal(t, info.WatchPort, decoded.RedisPort)
	assert.Equal(t, info.MqEndpoint, decoded.MqEndpoint)
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
	assert.Zero(t, info.NatsPort)
	assert.Empty(t, info.MqEndpoint)
	assert.False(t, info.RedisEmbedded)
	assert.Zero(t, info.RedisPort2)
}
