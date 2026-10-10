package watch

import (
	"context"
	"encoding/json/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

func (c *Client) LoadAndSubscribe(ctx context.Context, key string, handle func(event Event)) (string, bool, Subscription) {
	c.lockLifecycle()
	defer c.unlockLifecycle()
	values, _, err := c.loadStableSnapshot(ctx, []string{key}, "")
	vpre.CheckNilError(err, "load redis key failed")
	value, ok := values[key]
	watcher := &_Watcher{
		client:       c,
		ctx:          ctx,
		key:          key,
		handle:       handle,
		snapshot:     &_WatchKeySnapshot{client: c, key: key, value: value, hasValue: ok},
		subscription: newSubscription(ctx, handle),
	}
	vpre.CheckNilError(c.subscribeWatcher(watcher), "subscribe failed")
	return value, ok, watcher.subscription
}

func (c *Client) LoadListAndSubscribe(ctx context.Context, prefix string, handle func(event Event)) (map[string]string, Subscription) {
	c.lockLifecycle()
	defer c.unlockLifecycle()
	valuesByKey, _, err := c.loadStableSnapshot(ctx, nil, prefix)
	vpre.CheckNilError(err, "load redis list failed")
	watcher := &_Watcher{
		client:       c,
		ctx:          ctx,
		prefix:       prefix,
		handle:       handle,
		snapshot:     &_WatchListSnapshot{client: c, prefix: prefix, valuesByKey: vmap.Clone(valuesByKey)},
		subscription: newSubscription(ctx, handle),
	}
	vpre.CheckNilError(c.subscribeWatcher(watcher), "subscribe failed")
	return valuesByKey, watcher.subscription
}

// LoadKeysAndSubscribe loads a stable key snapshot and watches those exact keys
// over one PubSub connection. Like other subscriptions, it reconciles after
// attachment and reconnect and buffers events until Start. keys must be nonempty
// literal Watch keys, without Redis pattern characters.
func (c *Client) LoadKeysAndSubscribe(ctx context.Context, keys []string, handle func(event Event)) (map[string]string, Subscription, error) {
	if err := c.lockLifecycleContext(ctx); err != nil {
		return nil, nil, err
	}
	defer c.unlockLifecycle()
	keys = append([]string(nil), keys...)
	vpre.Check(len(keys) > 0, "key subscription requires at least one key")
	for _, key := range keys {
		vpre.Check(key != "" && !strings.ContainsAny(key, "*?[]\\"), "key subscription requires literal keys")
	}
	slices.Sort(keys)
	keys = slices.Compact(keys)
	values, _, err := c.loadStableSnapshot(ctx, keys, "")
	if err != nil {
		return nil, nil, err
	}
	watcher := &_Watcher{
		client: c, ctx: ctx, keys: keys, handle: handle,
		snapshot:     &_WatchListSnapshot{client: c, keys: keys, valuesByKey: vmap.Clone(values)},
		subscription: newSubscription(ctx, handle),
	}
	if err := c.subscribeWatcher(watcher); err != nil {
		return nil, nil, err
	}
	return values, watcher.subscription, nil
}

func (c *Client) subscribeWatcher(watcher *_Watcher) error {
	redisClient, _ := c.currentRedisClient()
	c.addWatcher(watcher)
	succeeded := false
	defer func() {
		if !succeeded {
			c.removeWatcher(watcher)
		}
	}()
	if err := watcher.start(redisClient); err != nil {
		return err
	}
	succeeded = true
	return nil
}

// _Watcher owns one subscription to a key or key prefix. It outlives a
// reconnect: restart re-creates the PubSub on the current connection and
// reconciles the snapshot against Hub again, so keys that changed while the
// endpoint was stale are reported as regular events.
type _Watcher struct {
	client   *Client
	ctx      context.Context
	key      string
	prefix   string
	keys     []string
	handle   func(event Event)
	snapshot _Snapshot

	subscription *_Subscription

	mutex  sync.Mutex
	cancel context.CancelFunc
	pubsub *redis.PubSub
	done   chan struct{}
}

// _Snapshot reconciles watched values with the events delivered for them.
type _Snapshot interface {
	handleSubscription(handle func(event Event)) (uint64, error)
	handleEvent(handle func(event Event)) func(event Event)
}

func (w *_Watcher) start(redisClient *redis.Client) error {
	if redisClient == nil || w.ctx.Err() != nil {
		w.client.removeWatcher(w)
		return w.ctx.Err()
	}

	consumeCtx, cancel := context.WithCancel(w.ctx)
	pubsub := w.subscribe(consumeCtx, redisClient)
	succeeded := false
	defer func() {
		if !succeeded {
			cancel()
			_ = pubsub.Close()
		}
	}()
	attachCtx, attachCancel := context.WithTimeout(consumeCtx, snapshotTimeout)
	defer attachCancel()
	stopClose := context.AfterFunc(attachCtx, func() { _ = pubsub.Close() })
	defer stopClose()
	// Redis acknowledges each pattern separately. Only a complete group is a
	// subscription barrier; a single acknowledgement must not reload all keys.
	for {
		message, err := pubsub.Receive(attachCtx)
		if err != nil {
			return err
		}
		if acknowledgement, ok := message.(*redis.Subscription); ok && acknowledgement.Count == w.subscriptionCount() {
			break
		}
	}

	// Reconcile before consuming: the reload covers everything that changed
	// while no subscription was attached, and later events are filtered by the
	// resulting revision.
	reader := &Client{ctx: attachCtx, redisClient: redisClient}
	switch snapshot := w.snapshot.(type) {
	case *_WatchKeySnapshot:
		snapshot.client = reader
	case *_WatchListSnapshot:
		snapshot.client = reader
	}
	handleEvent := w.snapshot.handleEvent(w.enqueue)
	revision, err := w.snapshot.handleSubscription(handleEvent)
	if err != nil {
		return err
	}
	if !stopClose() || attachCtx.Err() != nil {
		return attachCtx.Err()
	}

	reader.ctx = consumeCtx
	w.mutex.Lock()
	w.cancel = cancel
	w.pubsub = pubsub
	w.done = make(chan struct{})
	done := w.done
	w.mutex.Unlock()
	succeeded = true

	go func() {
		defer close(done)
		w.client.consumePatternMessages(consumeCtx, pubsub.ChannelWithSubscriptions(), revision, w.subscriptionCount(), w.snapshot.handleSubscription, handleEvent, pubsub.Close)
	}()
	return nil
}

func (w *_Watcher) subscriptionCount() int { return max(1, len(w.keys)) }

func (w *_Watcher) subscribe(ctx context.Context, redisClient *redis.Client) *redis.PubSub {
	if len(w.keys) > 0 {
		return redisClient.PSubscribe(ctx, w.keys...)
	}
	if w.key != "" {
		return redisClient.Subscribe(ctx, w.key)
	}
	return redisClient.PSubscribe(ctx, formatRedisListPattern(w.prefix))
}

func (w *_Watcher) enqueue(event Event) {
	w.subscription.enqueue(event)
}

func (w *_Watcher) restart(redisClient *redis.Client) {
	w.stop()
	err := w.start(redisClient)
	if w.ctx.Err() == nil {
		vpre.CheckNilError(err, "restart subscription failed")
	}
}

func (w *_Watcher) stop() {
	w.mutex.Lock()
	cancel := w.cancel
	pubsub := w.pubsub
	done := w.done
	w.cancel = nil
	w.pubsub = nil
	w.done = nil
	w.mutex.Unlock()

	if cancel != nil {
		cancel()
	}
	if pubsub != nil {
		_ = pubsub.Close()
	}
	if done != nil {
		<-done
	}
}

type _Subscription struct {
	ctx    context.Context
	handle func(event Event)

	mutex       sync.Mutex
	started     bool
	dispatching bool
	events      []Event
}

func newSubscription(ctx context.Context, handle func(event Event)) *_Subscription {
	return &_Subscription{
		ctx:    ctx,
		handle: handle,
		events: make([]Event, 0),
	}
}

func (s *_Subscription) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.started {
		return
	}
	s.started = true
	s.startDispatcherLocked()
}

func (s *_Subscription) enqueue(event Event) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.ctx.Err() != nil {
		return
	}
	s.events = append(s.events, event)
	s.startDispatcherLocked()
}

func (s *_Subscription) startDispatcherLocked() {
	if !s.started || s.dispatching || len(s.events) == 0 {
		return
	}
	s.dispatching = true
	go s.dispatch()
}

func (s *_Subscription) dispatch() {
	for {
		s.mutex.Lock()
		if s.ctx.Err() != nil || len(s.events) == 0 {
			clear(s.events)
			s.events = nil
			s.dispatching = false
			s.mutex.Unlock()
			return
		}
		event := s.events[0]
		s.events[0] = Event{}
		s.events = s.events[1:]
		s.mutex.Unlock()

		s.handle(event)
	}
}

func (c *Client) consumePatternMessages(
	ctx context.Context,
	messageCh <-chan any,
	revision uint64,
	subscriptionCount int,
	handleSubscription func(handle func(event Event)) (uint64, error),
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
				if (message.Kind == "subscribe" || message.Kind == "psubscribe") && message.Count == subscriptionCount {
					for {
						nextRevision, err := handleSubscription(handleEvent)
						if err == nil {
							revision = nextRevision
							break
						}
						select {
						case <-ctx.Done():
							return
						case <-time.After(100 * time.Millisecond):
						}
					}
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

type _WatchKeySnapshot struct {
	client   *Client
	key      string
	value    string
	hasValue bool
}

func (s *_WatchKeySnapshot) handleSubscription(handle func(event Event)) (uint64, error) {
	_, ctx := s.client.currentRedisClient()
	values, revision, err := s.client.loadStableSnapshot(ctx, []string{s.key}, "")
	if err != nil {
		return 0, err
	}
	value, ok := values[s.key]
	return s.reconcile(value, ok, revision, handle), nil
}

func (s *_WatchKeySnapshot) reconcile(value string, ok bool, revision uint64, handle func(event Event)) uint64 {
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

func (s *_WatchKeySnapshot) handleEvent(handle func(event Event)) func(event Event) {
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

type _WatchListSnapshot struct {
	client      *Client
	prefix      string
	keys        []string
	valuesByKey map[string]string
}

func (s *_WatchListSnapshot) handleSubscription(handle func(event Event)) (uint64, error) {
	_, ctx := s.client.currentRedisClient()
	values, revision, err := s.client.loadStableSnapshot(ctx, s.keys, s.prefix)
	if err != nil {
		return 0, err
	}
	return s.reconcile(values, revision, handle), nil
}

func (s *_WatchListSnapshot) reconcile(valuesByKey map[string]string, revision uint64, handle func(event Event)) uint64 {
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

func (s *_WatchListSnapshot) handleEvent(handle func(event Event)) func(event Event) {
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
