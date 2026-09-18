package nats

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/util/vpre"
)

// MessageAckWait is the acknowledgement deadline for managed message iterators.
// Long-running handlers must send InProgress before this interval expires.
const MessageAckWait = 30 * time.Second

// MessagesContext reads messages on demand and survives managed connection replacement.
type MessagesContext interface {
	Next(ctx context.Context) (jetstream.Msg, error)
	Stop()
}

// Messages creates a managed pull iterator with at most one message per pull.
// Callers must reserve processing capacity before calling Next.
func (c *_Client) Messages(ctx context.Context, streamConfig jetstream.StreamConfig, subject string, consumerName string) MessagesContext {
	c.recoverMutex.Lock()
	defer c.recoverMutex.Unlock()
	messages := new(_MessagesContext{
		client:       c,
		context:      ctx,
		streamConfig: streamConfig,
		subject:      subject,
		consumerName: sanitizeNATSResourceName(consumerName),
		changed:      make(chan struct{}),
	})
	messages.start()
	c.mutex.Lock()
	c.messages[messages] = struct{}{}
	c.mutex.Unlock()
	return messages
}

type _MessagesContext struct {
	client       *_Client
	context      context.Context
	streamConfig jetstream.StreamConfig
	subject      string
	consumerName string
	mutex        sync.Mutex
	messages     jetstream.MessagesContext
	changed      chan struct{}
	stopped      bool
}

func (c *_MessagesContext) Next(ctx context.Context) (jetstream.Msg, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c.mutex.Lock()
		messages, changed, stopped := c.messages, c.changed, c.stopped
		c.mutex.Unlock()
		if stopped {
			return nil, jetstream.ErrMsgIteratorClosed
		}
		msg, err := messages.Next(jetstream.NextContext(ctx))
		if !errors.Is(err, jetstream.ErrMsgIteratorClosed) {
			return msg, err
		}
		// A closed old connection/iterator waits for recovery, rather than spinning
		// or terminating the dispatcher before the replacement is installed.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

func (c *_MessagesContext) Stop() {
	c.mutex.Lock()
	if c.stopped {
		c.mutex.Unlock()
		return
	}
	c.stopped = true
	messages := c.messages
	c.messages = nil
	close(c.changed)
	c.mutex.Unlock()
	messages.Stop()
	c.client.mutex.Lock()
	delete(c.client.messages, c)
	c.client.mutex.Unlock()
}

func (c *_MessagesContext) start() {
	js := c.client.ensureJetStream(c.streamConfig)
	consumer, err := js.CreateOrUpdateConsumer(c.context, c.streamConfig.Name, jetstream.ConsumerConfig{
		Durable:       c.consumerName,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       MessageAckWait,
		MaxDeliver:    -1,
		FilterSubject: c.subject,
	})
	vpre.CheckNilError(err, "create nats jetstream pull consumer failed")
	c.messages, err = consumer.Messages(jetstream.PullMaxMessages(1))
	vpre.CheckNilError(err, "create nats jetstream message iterator failed")
}

func (c *_MessagesContext) restart() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.stopped {
		return
	}
	c.messages.Stop()
	c.start()
	close(c.changed)
	c.changed = make(chan struct{})
}

func (c *_Client) restartMessages() {
	c.mutex.Lock()
	messages := make([]*_MessagesContext, 0, len(c.messages))
	for messageContext := range c.messages {
		messages = append(messages, messageContext)
	}
	c.mutex.Unlock()
	for _, messageContext := range messages {
		messageContext.restart()
	}
}
