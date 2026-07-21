package nats

import (
	"context"
	"sync"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
)

type _Option struct {
	InprocMode bool
	Endpoint   string
}

type _ClientSpec interface {
	InitOption(option *_Option)

	mustBeClient()
}

// Client

type Client struct {
	_Client

	Flag    *flag.Flag       `inject:""`
	HubInfo *hubinfo.HubInfo `inject:""`
}

func (c *Client) InitOption(option *_Option) {
	option.InprocMode = c.Flag.HubInprocMode
	if option.InprocMode {
		return
	}

	option.Endpoint = c.HubInfo.MQEndpoint()
}

// _Client

type _Client struct {
	app.BaseFrameworkComponent[*_ClientMinder]

	conn          *gonats.Conn
	jetStream     jetstream.JetStream
	ensuredStream map[string]struct{}
	consumers     map[*_ConsumeContext]struct{}
	mutex         sync.Mutex
	recoverMutex  sync.Mutex
}

var newNATSConnect = gonats.Connect

func (*_Client) InitOption(*_Option) {}

func (*_Client) mustBeClient() {}

// Lifecycle

func (c *_Client) setConn(conn *gonats.Conn) {
	c.conn = conn
	js, err := jetstream.New(conn)
	vpre.CheckNilError(err, "create nats jetstream context failed")
	c.jetStream = js
	c.ensuredStream = map[string]struct{}{}
	c.consumers = map[*_ConsumeContext]struct{}{}
}

func (c *_Client) onReconnect(ctx context.Context, _ *gonats.Conn) {
	go c.recoverJetStream(ctx)
}

func (c *_Client) recoverJetStream(ctx context.Context) {
	c.recoverMutex.Lock()
	defer c.recoverMutex.Unlock()

	for {
		err := c.waitJetStreamReady(ctx, jetStreamReadyTimeout, jetStreamReadyInterval)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return
		}
		logger.Warn("nats jetstream not ready after reconnect, retrying", "error", err)
	}

	c.clearEnsuredStreams()
	c.restartConsumers()
}

func (c *_Client) clearEnsuredStreams() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.ensuredStream = map[string]struct{}{}
}

// Publish

func (c *_Client) Publish(streamConfig jetstream.StreamConfig, subject string, data []byte) {
	c.publishJetStream(streamConfig, subject, data)
}

func (c *_Client) publishJetStream(streamConfig jetstream.StreamConfig, subject string, data []byte) {
	c.ensureJetStream(streamConfig)
	_, err := c.jetStream.Publish(context.Background(), subject, data)
	vpre.CheckNilError(err, "publish nats jetstream message failed")
}

// Stream

func (c *_Client) ensureJetStream(streamConfig jetstream.StreamConfig) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.ensuredStream[streamConfig.Name]; exists {
		return
	}

	_, err := c.jetStream.Stream(context.Background(), streamConfig.Name)
	if err == nil {
		c.ensuredStream[streamConfig.Name] = struct{}{}
		return
	}
	if err != jetstream.ErrStreamNotFound {
		vpre.CheckNilError(err, "read nats jetstream info failed")
	}

	_, err = c.jetStream.CreateOrUpdateStream(context.Background(), streamConfig)
	if err != nil {
		_, infoErr := c.jetStream.Stream(context.Background(), streamConfig.Name)
		vpre.CheckNilError(infoErr, "create nats jetstream stream failed")
	}
	c.ensuredStream[streamConfig.Name] = struct{}{}
}
