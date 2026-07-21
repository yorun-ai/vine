package redis

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

func (c *Client) LoadAndSubscribe(ctx context.Context, key string, handle func(event Event)) (string, bool) {
	pubsub := c.redisClient.Subscribe(ctx, key)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		vpre.Panic(err)
	}

	value, ok, revision := c.loadStableValue(key)
	snapshot := _RedisKeySnapshot{
		client:   c,
		key:      key,
		value:    value,
		hasValue: ok,
	}
	go c.consumePatternMessages(ctx, pubsub.ChannelWithSubscriptions(), revision, snapshot.handleSubscription, snapshot.handleEvent(handle), pubsub.Close)
	return value, ok
}

func (c *Client) LoadListAndSubscribe(ctx context.Context, prefix string, handle func(event Event)) map[string]string {
	pattern := formatRedisListPattern(prefix)
	pubsub := c.redisClient.PSubscribe(ctx, pattern)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		vpre.Panic(err)
	}

	valuesByKey, revision := c.loadStableScanKeyValues(prefix)
	snapshot := _RedisListSnapshot{
		client:      c,
		prefix:      prefix,
		valuesByKey: vmap.Clone(valuesByKey),
	}
	go c.consumePatternMessages(ctx, pubsub.ChannelWithSubscriptions(), revision, snapshot.handleSubscription, snapshot.handleEvent(handle), pubsub.Close)
	return valuesByKey
}

func (c *Client) consumePatternMessages(
	ctx context.Context,
	messageCh <-chan interface{},
	revision uint64,
	handleSubscription func(handle func(event Event)) uint64,
	handleEvent func(event Event),
	closeFn func() error,
) {
	defer func() {
		_ = closeFn()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-messageCh:
			if !ok {
				return
			}
			switch message := message.(type) {
			case *redis.Subscription:
				if message.Kind == "subscribe" || message.Kind == "psubscribe" {
					revision = handleSubscription(handleEvent)
				}
			case *redis.Message:
				var event Event
				if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
					continue
				}
				if event.Revision < revision {
					continue
				}
				revision = event.Revision
				handleEvent(event)
			}
		}
	}
}

type _RedisKeySnapshot struct {
	client   *Client
	key      string
	value    string
	hasValue bool
}

func (s *_RedisKeySnapshot) handleSubscription(handle func(event Event)) uint64 {
	value, ok, revision := s.client.loadStableValue(s.key)
	return s.reconcile(value, ok, revision, handle)
}

func (s *_RedisKeySnapshot) reconcile(value string, ok bool, revision uint64, handle func(event Event)) uint64 {
	if ok {
		if s.hasValue && s.value == value {
			return revision
		}
		s.value = value
		s.hasValue = true
		handle(Event{
			Revision: revision,
			Kind:     EventKindUpsert,
			Key:      s.key,
			Value:    value,
		})
		return revision
	}

	if !s.hasValue {
		return revision
	}
	s.value = ""
	s.hasValue = false
	handle(Event{
		Revision: revision,
		Kind:     EventKindDelete,
		Key:      s.key,
	})
	return revision
}

func (s *_RedisKeySnapshot) handleEvent(handle func(event Event)) func(event Event) {
	return func(event Event) {
		switch event.Kind {
		case EventKindDelete:
			s.value = ""
			s.hasValue = false
		default:
			s.value = event.Value
			s.hasValue = true
		}
		handle(event)
	}
}

type _RedisListSnapshot struct {
	client      *Client
	prefix      string
	valuesByKey map[string]string
}

func (s *_RedisListSnapshot) handleSubscription(handle func(event Event)) uint64 {
	valuesByKey, revision := s.client.loadStableScanKeyValues(s.prefix)
	return s.reconcile(valuesByKey, revision, handle)
}

func (s *_RedisListSnapshot) reconcile(valuesByKey map[string]string, revision uint64, handle func(event Event)) uint64 {
	for key, value := range valuesByKey {
		oldValue, exists := s.valuesByKey[key]
		if exists && oldValue == value {
			continue
		}
		s.valuesByKey[key] = value
		handle(Event{
			Revision: revision,
			Kind:     EventKindUpsert,
			Key:      key,
			Value:    value,
		})
	}
	for key := range s.valuesByKey {
		if _, ok := valuesByKey[key]; ok {
			continue
		}
		delete(s.valuesByKey, key)
		handle(Event{
			Revision: revision,
			Kind:     EventKindDelete,
			Key:      key,
		})
	}
	return revision
}

func (s *_RedisListSnapshot) handleEvent(handle func(event Event)) func(event Event) {
	return func(event Event) {
		switch event.Kind {
		case EventKindDelete:
			delete(s.valuesByKey, event.Key)
		default:
			s.valuesByKey[event.Key] = event.Value
		}
		handle(event)
	}
}
