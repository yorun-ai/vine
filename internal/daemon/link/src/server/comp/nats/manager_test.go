package nats

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	hubnatsserver "go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
)

type _TestInprocClient struct {
	Client
}

func (*_TestInprocClient) InitOption(option *_Option) {
	option.InprocMode = true
}

type _TestRemoteClient struct {
	Client

	Endpoint   string
	reconnects int
}

func (c *_TestRemoteClient) InitOption(option *_Option) {
	option.Endpoint = c.Endpoint
}

// onReconnect records the reconnect callback instead of restarting JetStream,
// which keeps the manager tests free of a live NATS server.
func (c *_TestRemoteClient) onReconnect(context.Context, *gonats.Conn) {
	c.reconnects++
}

func initTestClient(component app.ManagedComponent) *_ClientManager {
	manager := &_ClientManager{
		Context: context.Background(),
		HubInfo: &hubinfo.HubInfo{},
	}
	manager.InitComponent(component)
	return manager
}

func TestClientManagerUsesInprocServer(t *testing.T) {
	natsModule := &hubnatsserver.NATSServer{
		InprocFlag: &app.InternalInprocFlag{Enabled: true},
		Flag:       &hubflag.Flag{MQMode: hubflag.MQModeEmbedded},
	}
	natsModule.DIInit()
	t.Cleanup(natsModule.AfterAppStop)

	component := &_TestInprocClient{}
	manager := initTestClient(component)
	defer manager.AfterAppStop()

	if manager.conn == nil {
		t.Fatalf("expected inproc connection")
	}
}

func TestClientManagerBuildsRemoteConnectionWithReconnectOptions(t *testing.T) {
	oldNewNATSConnect := newNATSConnect
	defer func() {
		newNATSConnect = oldNewNATSConnect
	}()

	var gotEndpoint string
	var gotOptions gonats.Options
	var reconnectHandler gonats.ConnHandler
	newNATSConnect = func(endpoint string, options ...gonats.Option) (*gonats.Conn, error) {
		gotEndpoint = endpoint
		gotOptions = gonats.GetDefaultOptions()
		for _, option := range options {
			if err := option(&gotOptions); err != nil {
				return nil, err
			}
		}
		reconnectHandler = gotOptions.ReconnectedCB
		return new(gonats.Conn), nil
	}

	component := &_TestRemoteClient{Endpoint: "nats://127.0.0.1:4222"}
	component.ensuredStream = map[string]struct{}{
		"events": {},
	}
	initTestClient(component)

	assert.Equal(t, "nats://127.0.0.1:4222", gotEndpoint)
	assert.Equal(t, maxReconnects, gotOptions.MaxReconnect)
	assert.Equal(t, 5*time.Second, gotOptions.ReconnectWait)
	if reconnectHandler == nil {
		t.Fatalf("expected reconnect handler")
	}
}

func TestClientManagerKeepsConnectionWhenHubMQEndpointIsUnchanged(t *testing.T) {
	oldNewNATSConnect := newNATSConnect
	defer func() {
		newNATSConnect = oldNewNATSConnect
	}()

	dialed := make([]string, 0, 2)
	newNATSConnect = func(endpoint string, _ ...gonats.Option) (*gonats.Conn, error) {
		dialed = append(dialed, endpoint)
		return new(gonats.Conn), nil
	}

	component := &_TestRemoteClient{Endpoint: "nats://127.0.0.1:4222"}
	manager := initTestClient(component)
	manager.onHubInfoRefresh()

	assert.Equal(t, []string{"nats://127.0.0.1:4222"}, dialed)
	assert.Equal(t, 0, component.reconnects)
}

func TestClientManagerReconnectsWhenHubMQEndpointChanged(t *testing.T) {
	previousServer := newTestNATSServer(t)
	nextServer := newTestNATSServer(t)

	component := &_TestRemoteClient{Endpoint: "nats://" + previousServer.Addr().String()}
	manager := initTestClient(component)
	t.Cleanup(manager.AfterAppStop)
	previousConn := manager.conn
	assert.Equal(t, component.Endpoint, previousConn.ConnectedUrl())

	nextEndpoint := "nats://" + nextServer.Addr().String()
	component.Endpoint = nextEndpoint
	manager.onHubInfoRefresh()

	assert.NotSame(t, previousConn, manager.conn)
	assert.True(t, previousConn.IsClosed())
	assert.Equal(t, nextEndpoint, manager.conn.ConnectedUrl())
	assert.Equal(t, nextEndpoint, manager.option.Endpoint)
	assert.Equal(t, 1, component.reconnects)
}

func TestClientManagerKeepsAppliedConnectionWhenReplacementDialFails(t *testing.T) {
	previousServer := newTestNATSServer(t)
	nextServer := newTestNATSServer(t)

	component := &_TestRemoteClient{Endpoint: "nats://" + previousServer.Addr().String()}
	manager := initTestClient(component)
	previousConn := manager.conn
	previousOption := manager.option

	originalConnect := newNATSConnect
	failNext := true
	newNATSConnect = func(endpoint string, options ...gonats.Option) (*gonats.Conn, error) {
		if failNext {
			failNext = false
			return nil, errors.New("temporary dial failure")
		}
		return originalConnect(endpoint, options...)
	}
	t.Cleanup(func() { newNATSConnect = originalConnect })

	component.Endpoint = "nats://" + nextServer.Addr().String()
	assert.Panics(t, manager.onHubInfoRefresh)
	assert.Same(t, previousConn, manager.conn)
	assert.Same(t, previousOption, manager.option)
	assert.False(t, previousConn.IsClosed())

	manager.onHubInfoRefresh()
	assert.NotSame(t, previousConn, manager.conn)
	assert.True(t, previousConn.IsClosed())
	t.Cleanup(func() { manager.conn.Close() })
}

func TestWaitJetStreamReadyRetriesTransientErrors(t *testing.T) {
	for _, transient := range []error{jetstream.ErrJetStreamNotEnabled, gonats.ErrConnectionClosed, context.DeadlineExceeded} {
		t.Run(transient.Error(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				attempts := 0
				err := waitJetStreamReady(func(context.Context) error {
					attempts++
					if attempts < 3 {
						return transient
					}
					return nil
				}, t.Context(), 50*time.Millisecond, time.Millisecond)
				assert.NoError(t, err)
				assert.Equal(t, 3, attempts)
			})
		})
	}
}

func TestWaitJetStreamReadyReturnsServiceUnavailableOnTimeout(t *testing.T) {
	err := waitJetStreamReady(func(context.Context) error {
		return jetstream.ErrJetStreamNotEnabled
	}, context.Background(), 5*time.Millisecond, time.Millisecond)

	exErr, ok := err.(ex.Error)
	if !ok {
		t.Fatalf("expected ex.Error, got %T", err)
	}
	if exErr.Code() != ex.ServiceUnavailable {
		t.Fatalf("unexpected error code: %s", exErr.Code())
	}
}

func TestWaitJetStreamReadyReturnsNonRetryableError(t *testing.T) {
	expectedErr := errors.New("probe failed")

	err := waitJetStreamReady(func(context.Context) error {
		return expectedErr
	}, context.Background(), 50*time.Millisecond, time.Millisecond)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitJetStreamReadyReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitJetStreamReady(func(context.Context) error {
		return jetstream.ErrJetStreamNotEnabled
	}, ctx, 50*time.Millisecond, time.Millisecond)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: %v", err)
	}
}
