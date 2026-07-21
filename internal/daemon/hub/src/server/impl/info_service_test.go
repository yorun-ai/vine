package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
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
			APIListen:         ":7071",
			RedisListen:       ":7073",
			MQExternalNatsURL: "nats://127.0.0.1:4222",
		},
		NATSServer: &natsserver.NATSServer{},
	}

	info := service.GetInfo()

	assert.Equal(t, skeled.Info{
		ApiPort:    7071,
		RedisPort:  7073,
		NatsPort:   0,
		MqEndpoint: "nats://127.0.0.1:4222",
	}, info)
}

func TestHubInfoServiceReturnsNATSServerPortWhenEnabled(t *testing.T) {
	service := &InfoServiceServerImpl{
		InprocFlag: &app.InternalInprocFlag{},
		Flag: &flag.Flag{
			APIListen:      ":7071",
			RedisListen:    ":7073",
			MQEmbeddedNats: true,
		},
		NATSServer: &natsserver.NATSServer{
			InprocFlag: &app.InternalInprocFlag{},
			Flag:       &flag.Flag{MQEmbeddedNats: true},
		},
	}

	runTestNATSServerDIInit(t, service.NATSServer)
	t.Cleanup(service.NATSServer.AfterAppStop)

	info := service.GetInfo()

	assert.Equal(t, 7071, info.ApiPort)
	assert.Equal(t, 7073, info.RedisPort)
	assert.Equal(t, service.NATSServer.Port(), info.NatsPort)
	assert.Empty(t, info.MqEndpoint)
}
