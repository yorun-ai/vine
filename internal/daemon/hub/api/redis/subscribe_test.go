package redis

import (
	"maps"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
)

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
