package heartbeat

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

type _TestPortalRegistryClient struct {
	mutex         sync.Mutex
	registrations []skeled.PortalRegistration
	heartbeats    []skeled.PortalStatus
	unregistered  []skel.UUID
	registered    bool
	registerFails bool
}

func (c *_TestPortalRegistryClient) Register(registration skeled.PortalRegistration, _ ...client.InvokeOption) {
	c.mutex.Lock()
	c.registrations = append(c.registrations, registration)
	fails := c.registerFails
	c.mutex.Unlock()

	if fails {
		// A Hub without this service fails the call, which is what Portal sees
		// when it runs against an older Hub.
		panic("portal registration is not served")
	}
}

func (c *_TestPortalRegistryClient) Unregister(instanceId skel.UUID, _ ...client.InvokeOption) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.unregistered = append(c.unregistered, instanceId)
}

func (c *_TestPortalRegistryClient) Heartbeat(status skeled.PortalStatus, _ ...client.InvokeOption) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.heartbeats = append(c.heartbeats, status)
	return c.registered
}

func (c *_TestPortalRegistryClient) state() (int, int, int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return len(c.registrations), len(c.heartbeats), len(c.unregistered)
}

type _TestInfoServiceClient struct {
	mutex    sync.Mutex
	info     skeled.Info
	getCalls int
}

func (c *_TestInfoServiceClient) GetInfo(_ ...client.InvokeOption) skeled.Info {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.getCalls++
	return c.info
}

func (c *_TestInfoServiceClient) calls() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.getCalls
}

const testPortalInstanceId = "11111111-1111-1111-1111-111111111111"

func newTestHeartbeat(client *_TestPortalRegistryClient, infoClient *_TestInfoServiceClient, flags *flag.Flag) *Heartbeat {
	appInfo, err := meta.NewApp("vine.portal", "1.2.3", testPortalInstanceId)
	if err != nil {
		panic(err)
	}
	return &Heartbeat{
		Context:              context.Background(),
		Flag:                 flags,
		App:                  appInfo,
		HubInfo:              &hubinfo.HubInfo{Flag: flags, InfoServiceClient: infoClient},
		PortalRegistryClient: client,
	}
}

func TestHeartbeatRegistersOnStartAndUnregistersOnStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &_TestPortalRegistryClient{registered: true}
		component := newTestHeartbeat(client, &_TestInfoServiceClient{}, &flag.Flag{})
		component.Context = t.Context()

		prev := heartbeatInterval
		heartbeatInterval = 10 * time.Millisecond
		defer func() { heartbeatInterval = prev }()

		component.AfterAppStart()
		synctest.Sleep(30 * time.Millisecond)

		registrations, heartbeats, unregistered := client.state()
		require.Equal(t, 1, registrations)
		assert.GreaterOrEqual(t, heartbeats, 1)
		assert.Equal(t, 0, unregistered)
		client.mutex.Lock()
		assert.Equal(t, skel.NewUUID(uuid.MustParse(testPortalInstanceId)), client.registrations[0].InstanceId)
		assert.Equal(t, "1.2.3", client.registrations[0].Version)
		client.mutex.Unlock()

		component.BeforeAppStop()

		_, _, unregistered = client.state()
		assert.Equal(t, 1, unregistered)
	})
}

func TestHeartbeatRegistersAgainWhenHubLosesRegistration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &_TestPortalRegistryClient{}
		infoClient := &_TestInfoServiceClient{}
		component := newTestHeartbeat(client, infoClient, &flag.Flag{})
		component.Context = t.Context()

		prev := heartbeatInterval
		heartbeatInterval = 10 * time.Millisecond
		defer func() { heartbeatInterval = prev }()

		component.AfterAppStart()
		synctest.Sleep(10 * time.Millisecond)

		// Hub answered that it no longer knows this instance, which is what a Hub
		// restart looks like, so the heartbeat re-reads the Hub information a
		// restarted Hub may have changed and registers again.
		registrations, _, _ := client.state()
		assert.GreaterOrEqual(t, registrations, 2)
		assert.GreaterOrEqual(t, infoClient.calls(), 1)

		component.BeforeAppStop()
	})
}

func TestHeartbeatRegistersWithoutHeartbeatInStandaloneMode(t *testing.T) {
	client := &_TestPortalRegistryClient{}
	component := newTestHeartbeat(client, &_TestInfoServiceClient{}, &flag.Flag{HubInprocMode: true})

	component.AfterAppStart()
	component.BeforeAppStop()

	registrations, heartbeats, unregistered := client.state()
	assert.Equal(t, 1, registrations)
	assert.Equal(t, 0, heartbeats)
	assert.Equal(t, 1, unregistered)
}

func TestHeartbeatKeepsRunningWhenHubLacksTheRegistration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// A Hub that does not serve this service must not stop Portal: startup
		// succeeds and the heartbeat keeps trying until Hub answers.
		client := &_TestPortalRegistryClient{registered: false, registerFails: true}
		component := newTestHeartbeat(client, &_TestInfoServiceClient{}, &flag.Flag{})
		component.Context = t.Context()

		prev := heartbeatInterval
		heartbeatInterval = 10 * time.Millisecond
		defer func() { heartbeatInterval = prev }()

		component.AfterAppStart()
		synctest.Sleep(30 * time.Millisecond)

		registrations, heartbeats, _ := client.state()
		assert.GreaterOrEqual(t, registrations, 2)
		assert.GreaterOrEqual(t, heartbeats, 1)

		// Hub learns the service, so the next attempt registers for real.
		client.mutex.Lock()
		client.registerFails = false
		client.mutex.Unlock()
		synctest.Sleep(10 * time.Millisecond)

		client.mutex.Lock()
		lastRegistration := client.registrations[len(client.registrations)-1]
		client.mutex.Unlock()
		assert.Equal(t, skel.NewUUID(uuid.MustParse(testPortalInstanceId)), lastRegistration.InstanceId)

		component.BeforeAppStop()
	})
}
