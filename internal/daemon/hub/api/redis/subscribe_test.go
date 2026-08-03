package redis

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionReplaysDeleteAfterSnapshotPublication(t *testing.T) {
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
	for snapshotKey, value := range loadedSnapshot {
		published[snapshotKey] = value
	}
	subscription.Start()

	event := requireTestEvent(t, handled)
	assert.Equal(t, EventKindDelete, event.Kind)
	assert.NotContains(t, published, key)
}

func TestSubscriptionReplaysUpsertAfterSnapshotPublication(t *testing.T) {
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
	for snapshotKey, value := range loadedSnapshot {
		published[snapshotKey] = value
	}
	subscription.Start()

	event := requireTestEvent(t, handled)
	assert.Equal(t, EventKindUpsert, event.Kind)
	assert.Equal(t, "new", published[key])
}

func TestSubscriptionStartIsNonBlockingAndPreservesEventOrder(t *testing.T) {
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
	requireTestSignal(t, started)
	assert.Empty(t, handled)
	stateMutex.Unlock()

	for revision := uint64(1); revision <= 3; revision++ {
		assert.Equal(t, revision, requireTestEvent(t, handled).Revision)
	}
}

func applyTestEvent(valuesByKey map[string]string, event Event) {
	if event.Kind == EventKindDelete {
		delete(valuesByKey, event.Key)
		return
	}
	valuesByKey[event.Key] = event.Value
}

func requireTestEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		require.FailNow(t, "event delivery timeout")
		return Event{}
	}
}

func requireTestSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		require.FailNow(t, "subscription start timeout")
	}
}
