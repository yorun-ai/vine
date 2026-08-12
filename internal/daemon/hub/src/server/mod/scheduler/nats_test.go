package scheduler

import (
	"testing"
	"time"

	natsserverlib "github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func TestNATSTaskPublisherReturnsConfigurationError(t *testing.T) {
	publisher := &_NATSTaskPublisher{}

	err := publisher.PublishTask(newTestTaskMessage())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nats flag is nil")
}

func TestNATSTaskPublisherReturnsJetStreamError(t *testing.T) {
	server, err := natsserverlib.NewServer(&natsserverlib.Options{
		Port:   -1,
		NoSigs: true,
		NoLog:  true,
	})
	require.NoError(t, err)
	go server.Start()
	require.True(t, server.ReadyForConnections(time.Second))
	t.Cleanup(func() {
		hubnats.SetInprocServer(nil)
		server.Shutdown()
		server.WaitForShutdown()
	})
	hubnats.SetInprocServer(server)
	publisher := &_NATSTaskPublisher{Flag: &hubflag.Flag{}}

	err = publisher.PublishTask(newTestTaskMessage())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "read task nats jetstream stream")
}

func newTestTaskMessage() taskspec.NATSMessage {
	return taskspec.NATSMessage{
		TaskSkelName:    "demo.booker.RebuildCatalogIndexTask",
		TriggerSkelName: "rebuild",
		ArgumentsJson:   "{}",
	}
}
