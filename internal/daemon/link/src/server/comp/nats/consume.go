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
	consumeContext := &_ConsumeContext{
		client:       c,
		streamConfig: streamConfig,
		subject:      subject,
		consumerName: sanitizeNATSResourceName(consumerName),
		handle:       handle,
	}
	consumeContext.start()
	c.addConsumer(consumeContext)
	return consumeContext
}

func (c *_Client) consumeJetStream(streamConfig jetstream.StreamConfig, subject string, consumerName string, handle func(msg jetstream.Msg)) jetstream.ConsumeContext {
	c.ensureJetStream(streamConfig)
	consumer, err := c.jetStream.CreateOrUpdateConsumer(context.Background(), streamConfig.Name, jetstream.ConsumerConfig{
		Durable:       consumerName,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    -1,
		FilterSubject: subject,
		MemoryStorage: true,
	})
	vpre.CheckNilError(err, "create nats jetstream pull consumer failed")

	consumeContext, err := consumer.Consume(handle)
	vpre.CheckNilError(err, "consume nats jetstream subject failed")
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

func (c *_Client) restartConsumers() {
	c.mutex.Lock()
	consumers := make([]*_ConsumeContext, 0, len(c.consumers))
	for consumeContext := range c.consumers {
		consumers = append(consumers, consumeContext)
	}
	c.mutex.Unlock()

	for _, consumeContext := range consumers {
		consumeContext.restart()
	}
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

func (c *_ConsumeContext) start() {
	c.consumeContext = c.client.consumeJetStream(c.streamConfig, c.subject, c.consumerName, c.handle)
}

func (c *_ConsumeContext) restart() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.stopped {
		return
	}
	c.consumeContext.Stop()
	c.start()
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
