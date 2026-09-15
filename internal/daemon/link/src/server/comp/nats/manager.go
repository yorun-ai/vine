package nats

import (
	"context"
	"errors"
	"sync"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/util/vpre"
)

const (
	jetStreamReadyTimeout  = 1 * time.Second
	jetStreamReadyInterval = 100 * time.Millisecond
	maxReconnects          = -1 // nats.go treats negative MaxReconnects as retrying forever.
	reconnectWait          = 5 * time.Second
)

type _ClientOps interface {
	setConn(conn *gonats.Conn)
	onReconnect(ctx context.Context, conn *gonats.Conn)
	waitJetStreamReady(ctx context.Context, timeout time.Duration, interval time.Duration) error
}

type _ClientManager struct {
	app.BaseComponentManager

	Context context.Context  `inject:""`
	HubInfo *hubinfo.HubInfo `inject:""`

	repairMutex sync.Mutex
	client      app.ManagedComponent
	option      *_Option
	conn        *gonats.Conn
}

func (m *_ClientManager) InitComponent(component app.ManagedComponent) {
	m.client = component
	m.option = &_Option{}

	spec := component.(_ClientSpec)
	spec.InitOption(m.option)

	clientOps := component.(_ClientOps)
	if m.option.InprocMode {
		m.conn = hubnats.ConnectInproc()
		clientOps.setConn(m.conn)
		return
	}

	m.connect(m.option)
	m.HubInfo.OnRefresh(m.onHubInfoRefresh)
}

// connect dials the endpoint described by option and installs the new
// connection, the JetStream context and the consumer bookkeeping on the client.
func (m *_ClientManager) connect(option *_Option) {
	conn := m.newConnection(option)
	m.conn = conn
	m.client.(_ClientOps).setConn(conn)
}

func (m *_ClientManager) newConnection(option *_Option) *gonats.Conn {
	vpre.CheckNotEmpty(option.Endpoint, "nats endpoint is empty")

	connectOptions := []gonats.Option{
		gonats.MaxReconnects(maxReconnects),
		gonats.ReconnectWait(reconnectWait),
		gonats.ReconnectHandler(func(conn *gonats.Conn) {
			m.client.(_ClientOps).onReconnect(m.Context, conn)
		}),
	}
	if option.TLSConfig != nil {
		connectOptions = append(connectOptions, gonats.Secure(option.TLSConfig), gonats.TLSHandshakeFirst())
	}
	conn, err := newNATSConnect(option.Endpoint, connectOptions...)
	vpre.CheckNilError(err, "connect nats failed")
	return conn
}

// onHubInfoRefresh reconnects after Hub moved its MQ endpoint, which happens
// when a restarted Hub advertises a new embedded NATS port. An unchanged
// endpoint keeps the healthy connection, so nats.go can keep reconnecting on
// its own.
func (m *_ClientManager) onHubInfoRefresh() {
	m.repairMutex.Lock()
	defer m.repairMutex.Unlock()

	spec, ok := m.client.(_ClientSpec)
	if !ok {
		return
	}

	next := &_Option{}
	spec.InitOption(next)
	if next.InprocMode || next.Endpoint == m.option.Endpoint {
		return
	}

	conn := m.newConnection(next)
	previous := m.conn
	m.conn = conn
	m.option = next
	m.client.(_ClientOps).setConn(conn)
	natsLogger.Info("link reconnected to Hub MQ after its endpoint changed", "endpoint", next.Endpoint)
	m.client.(_ClientOps).onReconnect(m.Context, conn)
	if previous != nil {
		previous.Close()
	}
}

func (m *_ClientManager) Component() app.ManagedComponent {
	return m.client
}

func (m *_ClientManager) BeforeAppStart() error {
	return m.client.(_ClientOps).waitJetStreamReady(m.Context, jetStreamReadyTimeout, jetStreamReadyInterval)
}

func (m *_ClientManager) AfterAppStop() {
	m.repairMutex.Lock()
	defer m.repairMutex.Unlock()

	m.conn.Close()
}

func (c *_Client) waitJetStreamReady(ctx context.Context, timeout time.Duration, interval time.Duration) error {
	return waitJetStreamReady(func(ctx context.Context) error {
		c.mutex.Lock()
		js := c.jetStream
		c.mutex.Unlock()
		_, err := js.AccountInfo(ctx)
		return err
	}, ctx, timeout, interval)
}

func waitJetStreamReady(probe func(ctx context.Context) error, ctx context.Context, timeout time.Duration, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		probeCtx, cancel := context.WithTimeout(ctx, interval)
		err := probe(probeCtx)
		cancel()
		if err == nil {
			return nil
		}
		if !isJetStreamReadyRetryableError(err) {
			return err
		}
		lastErr = err
		if time.Now().After(deadline) {
			return ex.New(ex.ServiceUnavailable, "nats jetstream not ready", ex.WithDetail(lastErr.Error()))
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func isJetStreamReadyRetryableError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, gonats.ErrNoResponders) ||
		errors.Is(err, gonats.ErrConnectionClosed) ||
		errors.Is(err, gonats.ErrConnectionDraining) ||
		errors.Is(err, jetstream.ErrJetStreamNotEnabled) ||
		errors.Is(err, jetstream.ErrJetStreamNotEnabledForAccount)
}
