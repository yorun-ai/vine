package watchtest

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
)

func TestSubscriptionBuffersFiltersAndCancels(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := New(t, map[string]string{"config:one": "old", "other:one": "ignored"})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var events []watch.Event
		values, subscription := client.LoadListAndSubscribe(ctx, "config", func(event watch.Event) { events = append(events, event) })
		assert.Equal(t, map[string]string{"config:one": "old"}, values)
		client.Publish(watch.Event{Kind: watch.EventKindUpsert, Key: "config:one", Value: "new"})
		client.Publish(watch.Event{Kind: watch.EventKindUpsert, Key: "other:one", Value: "ignored"})
		synctest.Wait()
		assert.Empty(t, events)
		subscription.Start()
		subscription.Start()
		synctest.Wait()
		assert.Equal(t, []watch.Event{{Kind: watch.EventKindUpsert, Key: "config:one", Value: "new"}}, events)
		cancel()
		synctest.Wait()
		client.Publish(watch.Event{Kind: watch.EventKindDelete, Key: "config:one"})
		synctest.Wait()
		assert.Len(t, events, 1)
		_, exists := client.Load("config:one")
		assert.False(t, exists)
	})
}
