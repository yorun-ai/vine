package nats

import (
	"context"
	"errors"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
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

type _ClientMinder struct {
	app.BaseFrameworkComponentMinder

	Context context.Context `inject:""`

	client app.FrameworkComponent
	option *_Option
	conn   *gonats.Conn
}

func (m *_ClientMinder) InitComponent(component app.FrameworkComponent) {
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

	vpre.CheckNotEmpty(m.option.Endpoint, "nats endpoint is empty")
	conn, err := newNATSConnect(
		m.option.Endpoint,
		gonats.MaxReconnects(maxReconnects),
		gonats.ReconnectWait(reconnectWait),
		gonats.ReconnectHandler(func(conn *gonats.Conn) {
			clientOps.onReconnect(m.Context, conn)
		}),
	)
	vpre.CheckNilError(err, "connect nats failed")
	m.conn = conn
	clientOps.setConn(m.conn)
}

func (m *_ClientMinder) Component() app.FrameworkComponent {
	return m.client
}

func (m *_ClientMinder) BeforeAppStart() error {
	return m.client.(_ClientOps).waitJetStreamReady(m.Context, jetStreamReadyTimeout, jetStreamReadyInterval)
}

func (m *_ClientMinder) AfterAppStop() {
	m.conn.Close()
}

func (c *_Client) waitJetStreamReady(ctx context.Context, timeout time.Duration, interval time.Duration) error {
	return waitJetStreamReady(func(ctx context.Context) error {
		_, err := c.jetStream.AccountInfo(ctx)
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
