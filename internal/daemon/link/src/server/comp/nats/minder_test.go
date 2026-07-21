package nats

import (
	"context"
	"errors"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	hubnatsserver "go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

type _TestInprocClient struct {
	Client
}

func (*_TestInprocClient) InitOption(option *_Option) {
	option.InprocMode = true
}

type _TestRemoteClient struct {
	Client

	Endpoint string
}

func (c *_TestRemoteClient) InitOption(option *_Option) {
	option.Endpoint = c.Endpoint
}

func initTestClient(component app.FrameworkComponent) *_ClientMinder {
	minder := &_ClientMinder{
		Context: context.Background(),
	}
	minder.InitComponent(component)
	return minder
}

func TestClientMinderUsesInprocServer(t *testing.T) {
	natsModule := &hubnatsserver.NATSServer{
		InprocFlag: &app.InternalInprocFlag{Enabled: true},
		Flag:       &hubflag.Flag{MQEmbeddedNats: true},
	}
	natsModule.DIInit()
	t.Cleanup(natsModule.AfterAppStop)

	component := &_TestInprocClient{}
	minder := initTestClient(component)
	defer minder.AfterAppStop()

	if minder.conn == nil {
		t.Fatalf("expected inproc connection")
	}
}

func TestClientMinderBuildsRemoteConnection(t *testing.T) {
	server, err := natsserver.NewServer(&natsserver.Options{
		Port:      -1,
		NoSigs:    true,
		NoLog:     true,
		JetStream: true,
		StoreDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new nats server failed: %v", err)
	}
	go server.Start()
	if !server.ReadyForConnections(2 * time.Second) {
		t.Fatalf("nats server not ready")
	}
	defer server.Shutdown()

	component := &_TestRemoteClient{
		Endpoint: "nats://" + server.Addr().String(),
	}
	minder := initTestClient(component)
	defer minder.AfterAppStop()

	if minder.conn == nil {
		t.Fatalf("expected remote connection")
	}
}

func TestClientMinderBuildsRemoteConnectionWithReconnectOptions(t *testing.T) {
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

	component := &_TestRemoteClient{
		Endpoint: "nats://127.0.0.1:4222",
	}
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

func TestWaitJetStreamReadyRetriesUntilReady(t *testing.T) {
	callCount := 0

	err := waitJetStreamReady(func(context.Context) error {
		callCount++
		if callCount < 3 {
			return jetstream.ErrJetStreamNotEnabled
		}
		return nil
	}, context.Background(), 50*time.Millisecond, time.Millisecond)

	if err != nil {
		t.Fatalf("waitJetStreamReady() error = %v", err)
	}
	if callCount != 3 {
		t.Fatalf("unexpected call count: %d", callCount)
	}
}

func TestWaitJetStreamReadyRetriesConnectionErrors(t *testing.T) {
	callCount := 0

	err := waitJetStreamReady(func(context.Context) error {
		callCount++
		if callCount < 2 {
			return gonats.ErrConnectionClosed
		}
		return nil
	}, context.Background(), 50*time.Millisecond, time.Millisecond)

	if err != nil {
		t.Fatalf("waitJetStreamReady() error = %v", err)
	}
	if callCount != 2 {
		t.Fatalf("unexpected call count: %d", callCount)
	}
}

func TestWaitJetStreamReadyRetriesProbeDeadlineExceeded(t *testing.T) {
	callCount := 0

	err := waitJetStreamReady(func(context.Context) error {
		callCount++
		if callCount < 3 {
			return context.DeadlineExceeded
		}
		return nil
	}, context.Background(), 50*time.Millisecond, time.Millisecond)

	if err != nil {
		t.Fatalf("waitJetStreamReady() error = %v", err)
	}
	if callCount != 3 {
		t.Fatalf("unexpected call count: %d", callCount)
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
