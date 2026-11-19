package nats

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	gonats "github.com/nats-io/nats.go"
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
	messages.mutex.Lock()
	messages.stopOwner = context.AfterFunc(ctx, messages.Stop)
	messages.mutex.Unlock()
	return messages
}

type _MessagesContext struct {
	client       *_Client
	context      context.Context
	streamConfig jetstream.StreamConfig
	subject      string
	consumerName string
	mutex        sync.Mutex
	consumer     jetstream.Consumer
	iterator     *_MessageIterator
	changed      chan struct{}
	stopOwner    func() bool
}

func (c *_MessagesContext) Next(ctx context.Context) (jetstream.Msg, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := c.context.Err(); err != nil {
			return nil, err
		}

		iter, changed, err := c.currentIterator()
		if changed == nil {
			return nil, err
		}
		if err == nil {
			var msg jetstream.Msg
			msg, err = iter.next(ctx)
			if err == nil {
				return msg, nil
			}
			if ctx.Err() != nil {
				// Canceling Next alone leaves its pull on the server. Release it
				// when the reserving instance drains, even if other runners remain.
				c.mutex.Lock()
				if c.iterator == iter {
					c.resetIteratorLocked()
				}
				c.mutex.Unlock()
				return nil, ctx.Err()
			}
		}
		if !errors.Is(err, jetstream.ErrMsgIteratorClosed) && !errors.Is(err, gonats.ErrConnectionClosed) {
			natsLogger.Warn("message read failed", "subject", c.subject, "error", err)
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

func (c *_MessagesContext) currentIterator() (*_MessageIterator, <-chan struct{}, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.consumer == nil {
		return nil, nil, jetstream.ErrMsgIteratorClosed
	}
	if c.iterator == nil {
		messages, err := c.consumer.Messages(jetstream.PullMaxMessages(1))
		if err != nil {
			return nil, c.changed, err
		}
		c.iterator = new(_MessageIterator{MessagesContext: messages})
	}
	return c.iterator, c.changed, nil
}

type _MessageIterator struct {
	jetstream.MessagesContext
	delivered atomic.Bool
}

func (iter *_MessageIterator) next(ctx context.Context) (jetstream.Msg, error) {
	// After a delivery, Next can skip a pull while the previous send is still
	// marked in progress. Clearing that flag does not wake the reader.
	// If the 50 ms wait expires, re-enter Next once to recheck the pull,
	// then block normally so an idle consumer does not poll.
	// See https://github.com/nats-io/nats.go/issues/2240
	if iter.delivered.Load() {
		probeCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		msg, err := iter.Next(jetstream.NextContext(probeCtx))
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return msg, err
		}
	}
	msg, err := iter.Next(jetstream.NextContext(ctx))
	if err == nil {
		iter.delivered.Store(true)
	}
	return msg, err
}

// The next reserved slot opens a fresh iterator on the same durable consumer.
func (c *_MessagesContext) resetIteratorLocked() {
	if c.iterator != nil {
		c.iterator.Stop()
		c.iterator = nil
	}
	close(c.changed)
	c.changed = nil
	if c.consumer != nil {
		c.changed = make(chan struct{})
	}
}

func (c *_MessagesContext) Stop() {
	c.mutex.Lock()
	if c.changed == nil {
		c.mutex.Unlock()
		return
	}
	c.consumer = nil
	c.stopOwner()
	c.resetIteratorLocked()
	c.mutex.Unlock()
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
	if err != nil && c.context.Err() != nil {
		return // The owner's cancellation also stops this reader.
	}
	vpre.CheckNilError(err, "create nats jetstream pull consumer failed")
	c.consumer = consumer
}

func (c *_MessagesContext) restart() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.consumer == nil {
		return
	}
	c.resetIteratorLocked()
	c.start()
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
