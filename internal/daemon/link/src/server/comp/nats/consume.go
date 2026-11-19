package nats

import (
	"context"
	"strings"
	"sync"

	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/util/vpre"
)

type ConsumeContext interface {
	Stop()
}

func (c *_Client) Consume(streamConfig jetstream.StreamConfig, subject string, consumerName string, handle func(msg jetstream.Msg)) ConsumeContext {
	c.recoverMutex.Lock()
	defer c.recoverMutex.Unlock()
	consumeContext := &_ConsumeContext{
		client:       c,
		streamConfig: streamConfig,
		subject:      subject,
		consumerName: sanitizeNATSResourceName(consumerName),
		handle:       handle,
	}
	vpre.CheckNilError(consumeContext.restart(context.Background()), "start nats jetstream consumer failed")
	c.addConsumer(consumeContext)
	return consumeContext
}

func (c *_Client) addConsumer(consumeContext *_ConsumeContext) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.consumers[consumeContext] = struct{}{}
}

func (c *_Client) removeConsumer(consumeContext *_ConsumeContext) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.consumers, consumeContext)
}

type _ConsumeContext struct {
	client       *_Client
	streamConfig jetstream.StreamConfig
	subject      string
	consumerName string
	handle       func(msg jetstream.Msg)

	mutex          sync.Mutex
	consumeContext jetstream.ConsumeContext
	stopped        bool
}

func (c *_ConsumeContext) restart(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.stopped {
		return nil
	}
	js, err := c.client.ensureJetStream(ctx, c.streamConfig)
	if err != nil {
		return err
	}
	consumer, err := js.CreateOrUpdateConsumer(ctx, c.streamConfig.Name, jetstream.ConsumerConfig{
		Durable:       c.consumerName,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    -1,
		FilterSubject: c.subject,
	})
	if err != nil {
		return err
	}
	if c.consumeContext != nil {
		c.consumeContext.Stop()
	}
	next, err := consumer.Consume(c.handle)
	if err == nil {
		c.consumeContext = next
	}
	return err
}

func (c *_ConsumeContext) Stop() {
	c.mutex.Lock()
	if c.stopped {
		c.mutex.Unlock()
		return
	}
	c.stopped = true
	consumeContext := c.consumeContext
	c.consumeContext = nil
	c.mutex.Unlock()

	consumeContext.Stop()
	c.client.removeConsumer(c)
}

func sanitizeNATSResourceName(name string) string {
	var text strings.Builder
	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z':
			text.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			text.WriteRune(char)
		case char >= '0' && char <= '9':
			text.WriteRune(char)
		case char == '-' || char == '_':
			text.WriteRune(char)
		default:
			text.WriteByte('_')
		}
	}
	if text.Len() == 0 {
		return "consumer"
	}
	return text.String()
}
