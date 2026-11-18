package watch

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionSnapshotFailureRetriesAndRemainsCancellable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		messages := make(chan any, 1)
		messages <- &redis.Subscription{Kind: "subscribe", Count: 1}
		var calls atomic.Int32
		done := make(chan struct{})
		go func() {
			defer close(done)
			new(Client).consumePatternMessages(ctx, messages, 0, 1, true, func(func(Event)) (uint64, error) {
				if calls.Add(1) == 1 {
					return 0, errors.New("temporary snapshot failure")
				}
				return 2, nil
			}, func(Event) {}, func() error { return nil })
		}()
		synctest.Wait()
		require.Equal(t, int32(1), calls.Load())
		synctest.Sleep(100 * time.Millisecond)
		synctest.Wait()
		require.Equal(t, int32(2), calls.Load())
		cancel()
		<-done
	})
}

func TestCancelledWatcherIsRemovedFromRepairSet(t *testing.T) {
	client := new(Client)
	ctx, cancel := context.WithCancel(context.Background())
	watcher := &_Watcher{client: client, ctx: ctx}
	client.addWatcher(watcher)

	cancel()
	require.Eventually(t, func() bool {
		client.watcherMutex.Lock()
		defer client.watcherMutex.Unlock()
		return len(client.watchers) == 0
	}, time.Second, time.Millisecond)
}

func TestSubscriptionReplaysDeleteAfterSnapshotPublication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		key := "portal:site:demo"
		loadedSnapshot := map[string]string{key: "old"}
		published := map[string]string{}
		handled := make(chan Event, 1)
		subscription := newSubscription(t.Context(), func(event Event) {
			applyTestEvent(published, event)
			handled <- event
		})

		// The delete arrives after the scan but before the caller publishes its
		// loaded snapshot. It must remain buffered until Start.
		subscription.enqueue(Event{Revision: 2, Kind: EventKindDelete, Key: key})
		assert.Empty(t, handled)
		maps.Copy(published, loadedSnapshot)
		subscription.Start()
		synctest.Wait()

		event := <-handled
		assert.Equal(t, EventKindDelete, event.Kind)
		assert.NotContains(t, published, key)
	})
}

func TestSubscriptionReplaysUpsertAfterSnapshotPublication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		key := "portal:site:demo"
		loadedSnapshot := map[string]string{key: "old"}
		published := map[string]string{}
		handled := make(chan Event, 1)
		subscription := newSubscription(t.Context(), func(event Event) {
			applyTestEvent(published, event)
			handled <- event
		})

		// The upsert arrives after the scan but before snapshot publication.
		subscription.enqueue(Event{Revision: 2, Kind: EventKindUpsert, Key: key, Value: "new"})
		assert.Empty(t, handled)
		maps.Copy(published, loadedSnapshot)
		subscription.Start()
		synctest.Wait()

		event := <-handled
		assert.Equal(t, EventKindUpsert, event.Kind)
		assert.Equal(t, "new", published[key])
	})
}

func TestSubscriptionStartIsNonBlockingAndPreservesEventOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var stateMutex sync.Mutex
		handled := make(chan Event, 3)
		subscription := newSubscription(t.Context(), func(event Event) {
			stateMutex.Lock()
			defer stateMutex.Unlock()
			handled <- event
		})
		for revision := uint64(1); revision <= 3; revision++ {
			subscription.enqueue(Event{Revision: revision, Kind: EventKindUpsert})
		}

		stateMutex.Lock()
		started := make(chan struct{})
		go func() {
			subscription.Start()
			close(started)
		}()
		<-started
		assert.Empty(t, handled)
		stateMutex.Unlock()
		synctest.Wait()

		for revision := uint64(1); revision <= 3; revision++ {
			assert.Equal(t, revision, (<-handled).Revision)
		}
	})
}

func applyTestEvent(valuesByKey map[string]string, event Event) {
	if event.Kind == EventKindDelete {
		delete(valuesByKey, event.Key)
		return
	}
	valuesByKey[event.Key] = event.Value
}

type keysetCounter struct {
	gets      atomic.Int64
	pipelines atomic.Int64
}

func (h *keysetCounter) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *keysetCounter) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, commands []redis.Cmder) error {
		h.pipelines.Add(1)
		for _, command := range commands {
			if command.Name() == "get" {
				h.gets.Add(1)
			}
		}
		return next(ctx, commands)
	}
}
func (h *keysetCounter) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, command redis.Cmder) error {
		if command.Name() == "get" {
			h.gets.Add(1)
		}
		return next(ctx, command)
	}
}

func TestKeySetReadsGrowLinearlyAndReconcileAfterReconnect(t *testing.T) {
	server := miniredis.RunT(t)
	server.Set(RevisionKey, "1")
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr(), Protocol: 2, DisableIdentity: true, ContextTimeoutEnabled: true})
	counter := new(keysetCounter)
	rdb.AddHook(counter)
	client := new(Client)
	client.setRedisClient(t.Context(), rdb)
	t.Cleanup(client.Close)
	const count = 1000
	keys := make([]string, count)
	for i := range keys {
		keys[i] = fmt.Sprintf("descriptor:data:demo.T%d", i)
		server.Set(keys[i], "old")
	}
	delivered := make(chan Event, 10)
	_, sub, err := client.LoadKeysAndSubscribe(t.Context(), keys, func(event Event) { delivered <- event })
	require.NoError(t, err)
	sub.Start()
	server.Set(RevisionKey, "2")
	server.Publish(keys[0], fmt.Sprintf(`{"revision":2,"kind":"upsert","key":%q,"value":"sentinel"}`, keys[0]))
	select {
	case <-delivered:
	case <-time.After(5 * time.Second):
		t.Fatal("event stayed behind subscription acknowledgements")
	}
	require.Equal(t, int64(2*(count+2)), counter.gets.Load())
	require.Equal(t, int64(2*((count+snapshotBatchSize-1)/snapshotBatchSize)), counter.pipelines.Load())
	t.Logf("%d keys: %d GETs in %d data pipelines", count, counter.gets.Load(), counter.pipelines.Load())

	before := counter.gets.Load()
	server.Close()
	server.Set(keys[0], "restored")
	server.Set(RevisionKey, "3")
	require.NoError(t, server.Restart())
	select {
	case event := <-delivered:
		require.Equal(t, "restored", event.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("reconnect did not reconcile the key set")
	}
	require.InDelta(t, count+2, counter.gets.Load()-before, 4, "only one full reload is needed per reconnect")
}

func TestKeySetReadFailureAndLifecycleCancellationReturnErrors(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr(), Protocol: 2, DisableIdentity: true})
	client := new(Client)
	client.setRedisClient(t.Context(), rdb)
	require.NoError(t, rdb.Close())
	_, sub, err := client.LoadKeysAndSubscribe(t.Context(), []string{"descriptor:data:demo.User"}, func(Event) {})
	require.Error(t, err)
	require.Nil(t, sub)
	require.Empty(t, client.watcherSnapshot())

	client.lockLifecycle()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err = client.LoadKeysAndSubscribe(ctx, []string{"descriptor:data:demo.User"}, func(Event) {})
	client.unlockLifecycle()
	require.ErrorIs(t, err, context.Canceled)
}

func TestGroupAcknowledgementsReconcileOnlyWhenComplete(t *testing.T) {
	messages := make(chan any, 7)
	for _, count := range []int{1, 2, 3, 1, 2, 3} {
		messages <- &redis.Subscription{Kind: "psubscribe", Count: count}
	}
	close(messages)
	calls := 0
	new(Client).consumePatternMessages(t.Context(), messages, 0, 3, true, func(func(Event)) (uint64, error) {
		calls++
		return 1, nil
	}, func(Event) {}, func() error { return nil })
	require.Equal(t, 2, calls)
}

type failAttachedSnapshotHook struct {
	keysetCounter
	revisions atomic.Int32
}

func (h *failAttachedSnapshotHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, command redis.Cmder) error {
		if command.Name() == "get" && command.Args()[1] == RevisionKey && h.revisions.Add(1) == 3 {
			return errors.New("attached snapshot unavailable")
		}
		return next(ctx, command)
	}
}

func TestFailedKeySetAttachmentReleasesPartialSubscription(t *testing.T) {
	server := miniredis.RunT(t)
	server.Set(RevisionKey, "1")
	server.Set("descriptor:data:demo.User", "user")
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr(), Protocol: 2, DisableIdentity: true})
	rdb.AddHook(new(failAttachedSnapshotHook))
	client := new(Client)
	client.setRedisClient(t.Context(), rdb)
	t.Cleanup(client.Close)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, subscription, err := client.LoadKeysAndSubscribe(ctx, []string{"descriptor:data:demo.User"}, func(Event) {})
	require.ErrorContains(t, err, "attached snapshot unavailable")
	require.Nil(t, subscription)
	require.Empty(t, client.watcherSnapshot())
	require.Eventually(t, func() bool { return server.PubSubNumPat() == 0 }, time.Second, time.Millisecond)
}

func TestPeriodicReconciliationRepairsMissedEventsWithoutReconnect(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		messages := make(chan any, 4)
		cached := map[string]string{"rpc:old": "old-ip", "rpc:updated": "old-ip"}
		snapshot := &_WatchListSnapshot{valuesByKey: maps.Clone(cached)}
		var calls atomic.Int32
		var cacheMutex sync.Mutex
		readCache := func() map[string]string {
			cacheMutex.Lock()
			defer cacheMutex.Unlock()
			return maps.Clone(cached)
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			new(Client).consumePatternMessages(ctx, messages, 1, 1, true, func(handle func(Event)) (uint64, error) {
				calls.Add(1)
				// Hub changed while the subscription stayed connected, but its
				// notifications were missed. Only the periodic snapshot can repair it.
				return snapshot.reconcile(map[string]string{"rpc:new": "new-ip", "rpc:updated": "new-ip"}, 3, handle), nil
			}, snapshot.handleEvent(func(event Event) {
				cacheMutex.Lock()
				defer cacheMutex.Unlock()
				applyTestEvent(cached, event)
			}), func() error { return nil })
		}()
		synctest.Wait()
		require.Zero(t, calls.Load())
		synctest.Sleep(35 * time.Second)
		synctest.Wait()
		require.Equal(t, int32(1), calls.Load())
		require.Equal(t, map[string]string{"rpc:new": "new-ip", "rpc:updated": "new-ip"}, readCache())
		// Buffered events from the snapshot's revision must not resurrect old keys.
		messages <- &redis.Message{Payload: `{"revision":3,"kind":"upsert","key":"rpc:old","value":"old-ip"}`}
		messages <- &redis.Message{Payload: `{"revision":4,"kind":"upsert","key":"rpc:a","value":"a"}`}
		messages <- &redis.Message{Payload: `{"revision":4,"kind":"upsert","key":"rpc:b","value":"b"}`}
		synctest.Wait()
		require.NotContains(t, readCache(), "rpc:old")
		require.Equal(t, "a", readCache()["rpc:a"])
		require.Equal(t, "b", readCache()["rpc:b"], "same-revision incremental batch members must all apply")
		cancel()
		<-done
		synctest.Sleep(time.Minute)
		require.Equal(t, int32(1), calls.Load(), "canceled subscriptions stop reconciling")
	})
}

func TestPeriodicSnapshotFailurePreservesCacheAndContinuesEvents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		messages := make(chan any, 1)
		cached := map[string]string{"rpc:old": "old-ip"}
		snapshot := &_WatchListSnapshot{valuesByKey: maps.Clone(cached)}
		var calls atomic.Int32
		var cacheMutex sync.Mutex
		readCache := func() map[string]string {
			cacheMutex.Lock()
			defer cacheMutex.Unlock()
			return maps.Clone(cached)
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			new(Client).consumePatternMessages(ctx, messages, 1, 1, true, func(handle func(Event)) (uint64, error) {
				if calls.Add(1) == 1 {
					return 0, errors.New("snapshot read failed")
				}
				return snapshot.reconcile(map[string]string{"rpc:new": "new-ip"}, 3, handle), nil
			}, snapshot.handleEvent(func(event Event) {
				cacheMutex.Lock()
				defer cacheMutex.Unlock()
				applyTestEvent(cached, event)
			}), func() error { return nil })
		}()
		synctest.Wait()
		synctest.Sleep(35 * time.Second)
		synctest.Wait()
		require.Equal(t, map[string]string{"rpc:old": "old-ip"}, readCache())
		messages <- &redis.Message{Payload: `{"revision":2,"kind":"upsert","key":"rpc:new","value":"new-ip"}`}
		synctest.Wait()
		require.Equal(t, "new-ip", readCache()["rpc:new"])
		require.Contains(t, readCache(), "rpc:old")
		synctest.Sleep(35 * time.Second)
		synctest.Wait()
		require.Equal(t, int32(2), calls.Load())
		require.Equal(t, map[string]string{"rpc:new": "new-ip"}, readCache())
		cancel()
		<-done
	})
}

func TestSnapshotRevisionFiltersEqualEventsButRetainsNewBatch(t *testing.T) {
	messages := make(chan any, 4)
	for _, payload := range []string{
		`{"revision":5,"kind":"upsert","key":"rpc:old","value":"old-ip"}`,
		`{"revision":6,"kind":"delete","key":"rpc:a"}`,
		`{"revision":6,"kind":"delete","key":"rpc:b"}`,
		`{"revision":4,"kind":"upsert","key":"rpc:old","value":"old-ip"}`,
	} {
		messages <- &redis.Message{Payload: payload}
	}
	close(messages)
	var keys []string
	new(Client).consumePatternMessages(t.Context(), messages, 5, 1, true, func(func(Event)) (uint64, error) {
		t.Fatal("unexpected snapshot reload")
		return 0, nil
	}, func(event Event) { keys = append(keys, event.Key) }, func() error { return nil })
	require.Equal(t, []string{"rpc:a", "rpc:b"}, keys)
}

func TestInprocConsumerDoesNotPeriodicallyReconcile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		messages := make(chan any, 1)
		events := make(chan Event, 1)
		var reloads atomic.Int32
		done := make(chan struct{})
		go func() {
			defer close(done)
			new(Client).consumePatternMessages(ctx, messages, 1, 1, false, func(func(Event)) (uint64, error) {
				reloads.Add(1)
				return 2, nil
			}, func(event Event) { events <- event }, func() error { return nil })
		}()
		synctest.Wait()
		synctest.Sleep(2 * time.Minute)
		synctest.Wait()
		require.Zero(t, reloads.Load())
		messages <- &redis.Message{Payload: `{"revision":2,"kind":"delete","key":"rpc:old"}`}
		require.Equal(t, EventKindDelete, (<-events).Kind)
		// Subscription attachment/reconnection still reloads its snapshot.
		messages <- &redis.Subscription{Kind: "subscribe", Count: 1}
		synctest.Wait()
		require.Equal(t, int32(1), reloads.Load())
		cancel()
		<-done
	})
}
