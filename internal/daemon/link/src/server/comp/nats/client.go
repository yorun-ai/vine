package nats

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/daemon"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
)

var natsLogger = logger.New("daemon:link:nats")

type _Option struct {
	InprocMode bool
	Endpoint   string
	TLSConfig  *tls.Config
}

type _ClientSpec interface {
	InitOption(option *_Option)

	mustBeClient()
}

// Client

type Client struct {
	_Client

	Flag     *flag.Flag       `inject:""`
	HubInfo  *hubinfo.HubInfo `inject:""`
	Identity *mtls.Identity   `inject:""`
}

func (c *Client) InitOption(option *_Option) {
	option.InprocMode = c.Flag.HubInprocMode
	if option.InprocMode {
		return
	}

	option.Endpoint = c.HubInfo.MQEndpoint()
	if c.Identity.Enabled() && c.HubInfo.UsesEmbeddedNATS() {
		option.TLSConfig = c.Identity.ClientConfig(daemon.HubIdentity.SPIFFEPath())
	}
}

// _Client

type _Client struct {
	app.BaseManagedComponent[*_ClientManager]

	conn           *gonats.Conn
	jetStream      jetstream.JetStream
	ensuredStream  map[string]struct{}
	consumers      map[*_ConsumeContext]struct{}
	messages       map[*_MessagesContext]struct{}
	mutex          sync.Mutex
	recoverMutex   sync.Mutex
	cancelRecovery context.CancelFunc
}

var newNATSConnect = gonats.Connect

func (*_Client) InitOption(*_Option) {}

func (*_Client) mustBeClient() {}

// Lifecycle

func (c *_Client) setConn(conn *gonats.Conn) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	js, err := jetstream.New(conn)
	vpre.CheckNilError(err, "create nats jetstream context failed")
	c.conn = conn
	c.jetStream = js
	c.ensuredStream = map[string]struct{}{}
	if c.consumers == nil {
		c.consumers = map[*_ConsumeContext]struct{}{}
	}
	if c.messages == nil {
		c.messages = map[*_MessagesContext]struct{}{}
	}
}

func (c *_Client) onReconnect(ctx context.Context, _ *gonats.Conn) {
	go c.recoverJetStream(ctx)
}

type recoverableConsumer interface {
	restart(context.Context) error
}

func (c *_Client) recoverJetStream(ctx context.Context) {
	// Take the snapshot alongside subscription creation. A newer reconnect
	// supersedes this recovery, including waits for streams that no longer exist.
	c.recoverMutex.Lock()
	if c.cancelRecovery != nil {
		c.cancelRecovery()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.cancelRecovery = cancel
	c.mutex.Lock()
	clear(c.ensuredStream)
	pending := make([]recoverableConsumer, 0, len(c.consumers)+len(c.messages))
	for consumer := range c.consumers {
		pending = append(pending, consumer)
	}
	for messages := range c.messages {
		pending = append(pending, messages)
	}
	c.mutex.Unlock()
	c.recoverMutex.Unlock()

	// AccountInfo can succeed before Hub has provisioned every stream.
	// Retry only failed consumers, leaving recovered subscriptions running.
	for delay := jetStreamReadyInterval; len(pending) > 0; delay = min(2*delay, reconnectWait) {
		failed := pending[:0]
		for _, consumer := range pending {
			if ctx.Err() != nil {
				return
			}
			if err := consumer.restart(ctx); err != nil {
				natsLogger.Warn("nats consumer recovery failed, retrying", "error", err)
				failed = append(failed, consumer)
			}
		}
		clear(pending[len(failed):])
		pending = failed
		if len(pending) > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}
}

// Publish

func (c *_Client) Publish(streamConfig jetstream.StreamConfig, subject string, data []byte) {
	c.publishJetStream(streamConfig, subject, data)
}

func (c *_Client) publishJetStream(streamConfig jetstream.StreamConfig, subject string, data []byte) {
	js, err := c.ensureJetStream(context.Background(), streamConfig)
	vpre.CheckNilError(err, "read nats jetstream stream failed")
	_, err = js.Publish(context.Background(), subject, data)
	vpre.CheckNilError(err, "publish nats jetstream message failed")
}

// Stream

func (c *_Client) ensureJetStream(ctx context.Context, streamConfig jetstream.StreamConfig) (jetstream.JetStream, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.ensuredStream[streamConfig.Name]; exists {
		return c.jetStream, nil
	}

	_, err := c.jetStream.Stream(ctx, streamConfig.Name)
	if err != nil {
		return nil, err
	}
	c.ensuredStream[streamConfig.Name] = struct{}{}
	return c.jetStream, nil
}
