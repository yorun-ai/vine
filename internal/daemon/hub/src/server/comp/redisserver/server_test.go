package redisserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/app"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func TestNewServerPasswordUsesRandom256BitValue(t *testing.T) {
	first := newServerPassword()
	second := newServerPassword()

	assert.Len(t, first, 43)
	assert.Len(t, second, 43)
	assert.NotEqual(t, first, second)
}

func newNetworkTestServer(t *testing.T) (*Server, *redis.Client) {
	t.Helper()

	server := &Server{
		Context: context.Background(),
		Option: &flag.Flag{
			RedisListen: "127.0.0.1:0",
		},
		InprocFlag: &app.InternalInprocFlag{},
	}
	server.DIInit()
	t.Cleanup(server.AfterAppStop)

	client := redis.NewClient(&redis.Options{
		Addr:            RedisListenAddrForTest(t, server),
		Protocol:        2,
		DisableIdentity: true,
		Username:        "hubserver",
		Password:        server.serverPassword,
	})
	t.Cleanup(func() {
		_ = client.Close()
	})
	return server, client
}

func newInprocTestServer(t *testing.T) (*Server, *redis.Client) {
	t.Helper()

	server := &Server{
		Context: context.Background(),
		Option:  &flag.Flag{},
		InprocFlag: &app.InternalInprocFlag{
			Enabled: true,
		},
	}
	server.DIInit()
	t.Cleanup(server.AfterAppStop)

	client := newInprocTestClient(server)
	t.Cleanup(func() {
		_ = client.Close()
	})
	return server, client
}

func newInprocTestClient(server *Server) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:            hubredis.RedisInprocEndpoint,
		Dialer:          hubredis.DialInproc,
		Protocol:        2,
		DisableIdentity: true,
		Username:        "hubserver",
		Password:        server.serverPassword,
	})
}

func newInprocReadOnlyClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:            hubredis.RedisInprocEndpoint,
		Dialer:          hubredis.DialInproc,
		Protocol:        2,
		DisableIdentity: true,
	})
}

func TestRedisInprocClientCommands(t *testing.T) {
	server, client := newInprocTestServer(t)
	ctx := context.Background()

	assert.Equal(t, server, hubredis.InprocServer())

	pong, err := client.Ping(ctx).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", pong)

	require.NoError(t, client.Set(ctx, "config:feature-a", "a", 0).Err())
	require.NoError(t, client.Set(ctx, "config:feature-b", "b", 0).Err())
	require.NoError(t, client.Set(ctx, "config:feature-c", "c", 0).Err())

	value, err := client.Get(ctx, "config:feature-a").Result()
	require.NoError(t, err)
	assert.Equal(t, "a", value)

	var cursor uint64
	keys := make([]string, 0)
	for {
		var batch []string
		batch, cursor, err = client.Scan(ctx, cursor, "config:*", 2).Result()
		require.NoError(t, err)
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}
	assert.Equal(t, []string{"config:feature-a", "config:feature-b", "config:feature-c"}, keys)
}

func TestRedisInprocClientHandshakeAndMutationCommands(t *testing.T) {
	_, client := newInprocTestServer(t)
	ctx := context.Background()

	hello, err := client.Do(ctx, "HELLO").Slice()
	require.NoError(t, err)
	assert.Contains(t, hello, "server")
	assert.Contains(t, hello, "redis")
	assert.Contains(t, hello, "proto")
	assert.Contains(t, hello, int64(2))

	echo, err := client.Do(ctx, "PING", "inproc").Text()
	require.NoError(t, err)
	assert.Equal(t, "inproc", echo)

	value, err := client.Incr(ctx, "counter").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), value)

	require.NoError(t, client.Set(ctx, "counter", "bad", 0).Err())
	_, err = client.Incr(ctx, "counter").Result()
	assert.Error(t, err)

	require.NoError(t, client.Set(ctx, "ephemeral", "value", 0).Err())
	ttl, err := client.TTL(ctx, "ephemeral").Result()
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl)

	updated, err := client.Expire(ctx, "ephemeral", time.Second).Result()
	require.NoError(t, err)
	assert.True(t, updated)

	ttl, err = client.TTL(ctx, "ephemeral").Result()
	require.NoError(t, err)
	assert.True(t, ttl >= 0)
	assert.True(t, ttl <= time.Second)

	deleted, err := client.Del(ctx, "ephemeral", "missing").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}

func TestRedisInprocClientACL(t *testing.T) {
	server, serverClient := newInprocTestServer(t)
	ctx := context.Background()
	require.NoError(t, serverClient.Set(ctx, "config:feature-a", "a", 0).Err())

	client := newInprocReadOnlyClient()
	t.Cleanup(func() {
		_ = client.Close()
	})

	value, err := client.Get(ctx, "config:feature-a").Result()
	require.NoError(t, err)
	assert.Equal(t, "a", value)

	_, err = client.Del(ctx, "config:feature-a").Result()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOPERM")

	assert.Equal(t, server, hubredis.InprocServer())
}

func TestRedisInprocClientPubSub(t *testing.T) {
	_, client := newInprocTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pubsub := client.PSubscribe(ctx, "config:*")
	defer pubsub.Close()
	subscription, err := pubsub.Receive(ctx)
	require.NoError(t, err)
	requireSubscription(t, subscription, "psubscribe", "config:*", 1)

	count, err := client.Publish(ctx, "config:feature-a", "payload").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	message, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "config:*", message.Pattern)
	assert.Equal(t, "config:feature-a", message.Channel)
	assert.Equal(t, "payload", message.Payload)
}

func TestRedisInprocClientSubscribeUnsubscribe(t *testing.T) {
	_, client := newInprocTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pubsub := client.Subscribe(ctx, "config:feature-a")
	defer pubsub.Close()
	subscription, err := pubsub.Receive(ctx)
	require.NoError(t, err)
	requireSubscription(t, subscription, "subscribe", "config:feature-a", 1)

	count, err := client.Publish(ctx, "config:feature-a", "first").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	message, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "config:feature-a", message.Channel)
	assert.Equal(t, "first", message.Payload)

	require.NoError(t, pubsub.Unsubscribe(ctx, "config:feature-a"))
	subscription, err = pubsub.Receive(ctx)
	require.NoError(t, err)
	requireSubscription(t, subscription, "unsubscribe", "config:feature-a", 0)

	count, err = client.Publish(ctx, "config:feature-a", "second").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	_, err = pubsub.ReceiveTimeout(ctx, 10*time.Millisecond)
	assert.True(t, isPubSubTimeout(err), "expected timeout, got %v", err)
}

func TestRedisInprocDialFailsAfterStop(t *testing.T) {
	server, client := newInprocTestServer(t)
	ctx := context.Background()

	require.NoError(t, client.Ping(ctx).Err())
	require.NoError(t, client.Close())
	server.AfterAppStop()

	_, err := hubredis.DialInproc(ctx, "tcp", hubredis.RedisInprocEndpoint)

	assert.EqualError(t, err, "redis inproc server is not initialized")
	assert.Nil(t, hubredis.InprocServer())
}

func TestRedisInprocDialFailsWhenServerMissing(t *testing.T) {
	oldServer := hubredis.InprocServer()
	hubredis.SetInprocServer(nil)
	t.Cleanup(func() {
		hubredis.SetInprocServer(oldServer)
	})

	_, err := hubredis.DialInproc(context.Background(), "tcp", hubredis.RedisInprocEndpoint)

	assert.EqualError(t, err, "redis inproc server is not initialized")
}

func TestRedisInprocDialFailsForRemoteServer(t *testing.T) {
	server := NewServerForTest()

	_, err := server.DialInproc(context.Background())

	assert.EqualError(t, err, "redis server is not inproc")
}

func TestRedisServerACL(t *testing.T) {
	server, serverClient := newNetworkTestServer(t)
	ctx := context.Background()
	require.NoError(t, serverClient.Set(ctx, "config:feature-a", "a", 0).Err())

	client := redis.NewClient(&redis.Options{
		Addr:            RedisListenAddrForTest(t, server),
		Protocol:        2,
		DisableIdentity: true,
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	value, err := client.Get(ctx, "config:feature-a").Result()
	require.NoError(t, err)
	assert.Equal(t, "a", value)

	for _, run := range []struct {
		name string
		run  func() error
	}{
		{"set", func() error { return client.Set(ctx, "config:feature-b", "b", 0).Err() }},
		{"incr", func() error { return client.Incr(ctx, "counter").Err() }},
		{"del", func() error { return client.Del(ctx, "config:feature-a").Err() }},
		{"publish", func() error { return client.Publish(ctx, "config:feature-a", "payload").Err() }},
		{"expire", func() error { return client.Expire(ctx, "config:feature-a", time.Second).Err() }},
	} {
		t.Run("read-only client rejects "+run.name, func(t *testing.T) {
			err := run.run()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "NOPERM")
		})
	}

	badPassword := redis.NewClient(&redis.Options{
		Addr:            RedisListenAddrForTest(t, server),
		Protocol:        2,
		DisableIdentity: true,
		Username:        "hubserver",
		Password:        "wrong",
	})
	t.Cleanup(func() {
		_ = badPassword.Close()
	})
	_, err = badPassword.Get(ctx, "config:feature-a").Result()
	assert.Error(t, err)
}

func TestRedisServerCommands(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx := context.Background()

	require.NoError(t, client.Set(ctx, "config:feature-a", "a", 0).Err())
	require.NoError(t, client.Set(ctx, "config:feature-b", "b", 0).Err())
	require.NoError(t, client.Set(ctx, "app:demo:status:1", "status", 0).Err())

	value, err := client.Get(ctx, "config:feature-a").Result()
	require.NoError(t, err)
	assert.Equal(t, "a", value)

	keys, cursor, err := client.Scan(ctx, 0, "config:*", 1000).Result()
	require.NoError(t, err)
	assert.Zero(t, cursor)
	assert.Equal(t, []string{"config:feature-a", "config:feature-b"}, keys)

	ttl, err := client.TTL(ctx, "config:feature-a").Result()
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl)

	updated, err := client.Expire(ctx, "config:feature-a", 2*time.Second).Result()
	require.NoError(t, err)
	assert.True(t, updated)

	ttl, err = client.TTL(ctx, "config:feature-a").Result()
	require.NoError(t, err)
	assert.True(t, ttl > 0)
	assert.True(t, ttl <= 2*time.Second)

	count, err := client.Del(ctx, "config:feature-b", "missing").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRedisServerHandshakeAndPing(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx := context.Background()

	hello, err := client.Do(ctx, "HELLO").Slice()
	require.NoError(t, err)
	assert.Contains(t, hello, "server")
	assert.Contains(t, hello, "redis")
	assert.Contains(t, hello, "proto")
	assert.Contains(t, hello, int64(2))

	pong, err := client.Ping(ctx).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", pong)

	echo, err := client.Do(ctx, "PING", "hello").Text()
	require.NoError(t, err)
	assert.Equal(t, "hello", echo)
}

func TestRedisServerExpiryRemovesKeyFromReadsAndScan(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx := context.Background()
	now := time.Now()
	SetTimeNowForTest(t, func() time.Time { return now })

	require.NoError(t, client.Set(ctx, "config:short", "gone", 0).Err())
	require.NoError(t, client.Set(ctx, "config:long", "kept", 0).Err())
	updated, err := client.Expire(ctx, "config:short", time.Second).Result()
	require.NoError(t, err)
	require.True(t, updated)

	now = now.Add(2 * time.Second)

	_, err = client.Get(ctx, "config:short").Result()
	assert.ErrorIs(t, err, redis.Nil)

	keys, cursor, err := client.Scan(ctx, 0, "config:*", 1000).Result()
	require.NoError(t, err)
	assert.Zero(t, cursor)
	assert.Equal(t, []string{"config:long"}, keys)

	deleted, err := client.Del(ctx, "config:short", "config:long").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}

func TestRedisServerScanPaginatesWithCursor(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx := context.Background()

	require.NoError(t, client.Set(ctx, "config:feature-a", "a", 0).Err())
	require.NoError(t, client.Set(ctx, "config:feature-b", "b", 0).Err())
	require.NoError(t, client.Set(ctx, "config:feature-c", "c", 0).Err())
	require.NoError(t, client.Set(ctx, "config:feature-d", "d", 0).Err())

	var cursor uint64
	var keys []string
	for {
		var batch []string
		var err error
		batch, cursor, err = client.Scan(ctx, cursor, "config:*", 2).Result()
		require.NoError(t, err)
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	assert.Equal(t, []string{
		"config:feature-a",
		"config:feature-b",
		"config:feature-c",
		"config:feature-d",
	}, keys)
}

func TestRedisServerIncr(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx := context.Background()

	value, err := client.Incr(ctx, "counter").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), value)

	value, err = client.Incr(ctx, "counter").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(2), value)

	require.NoError(t, client.Set(ctx, "counter", "bad", 0).Err())
	_, err = client.Incr(ctx, "counter").Result()
	assert.Error(t, err)
}

func TestRedisServerPatternPubSub(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pubsub := client.PSubscribe(ctx, "config:*")
	defer pubsub.Close()
	_, err := pubsub.Receive(ctx)
	require.NoError(t, err)

	count, err := client.Publish(ctx, "config:feature-a", "payload").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	message, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "config:*", message.Pattern)
	assert.Equal(t, "config:feature-a", message.Channel)
	assert.Equal(t, "payload", message.Payload)
}

func TestRedisServerSubscribeUnsubscribeStopsMessages(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pubsub := client.Subscribe(ctx, "config:feature-a")
	defer pubsub.Close()
	subscription, err := pubsub.Receive(ctx)
	require.NoError(t, err)
	requireSubscription(t, subscription, "subscribe", "config:feature-a", 1)

	count, err := client.Publish(ctx, "config:feature-a", "first").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	message, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "config:feature-a", message.Channel)
	assert.Equal(t, "first", message.Payload)

	require.NoError(t, pubsub.Unsubscribe(ctx, "config:feature-a"))
	subscription, err = pubsub.Receive(ctx)
	require.NoError(t, err)
	requireSubscription(t, subscription, "unsubscribe", "config:feature-a", 0)

	count, err = client.Publish(ctx, "config:feature-a", "second").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	_, err = pubsub.ReceiveTimeout(ctx, 10*time.Millisecond)
	assert.True(t, isPubSubTimeout(err), "expected timeout, got %v", err)
}

func TestRedisServerPUnsubscribeStopsPatternMessages(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pubsub := client.PSubscribe(ctx, "config:*")
	defer pubsub.Close()
	subscription, err := pubsub.Receive(ctx)
	require.NoError(t, err)
	requireSubscription(t, subscription, "psubscribe", "config:*", 1)

	count, err := client.Publish(ctx, "config:feature-a", "first").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	message, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "config:*", message.Pattern)
	assert.Equal(t, "config:feature-a", message.Channel)
	assert.Equal(t, "first", message.Payload)

	require.NoError(t, pubsub.PUnsubscribe(ctx, "config:*"))
	subscription, err = pubsub.Receive(ctx)
	require.NoError(t, err)
	requireSubscription(t, subscription, "punsubscribe", "config:*", 0)

	count, err = client.Publish(ctx, "config:feature-a", "second").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	_, err = pubsub.ReceiveTimeout(ctx, 10*time.Millisecond)
	assert.True(t, isPubSubTimeout(err), "expected timeout, got %v", err)
}

func TestRedisSetAndNotifyUpdatesRevisionAndPublishes(t *testing.T) {
	server, client := newNetworkTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pubsub := client.Subscribe(ctx, "config:feature-a")
	defer pubsub.Close()
	_, err := pubsub.Receive(ctx)
	require.NoError(t, err)

	baseRevision := testRevision(t, server)
	server.SetAndNotify("config:feature-a", "on")

	value, ok := server.Get("config:feature-a")
	assert.True(t, ok)
	assert.Equal(t, "on", value)
	assert.Equal(t, strconv.FormatUint(baseRevision+1, 10), mustGet(t, server, hubredis.RevisionKey))

	message, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)

	var got hubredis.Event
	require.NoError(t, json.Unmarshal([]byte(message.Payload), &got))
	assert.Equal(t, hubredis.Event{
		Revision: baseRevision + 1,
		Kind:     hubredis.EventKindUpsert,
		Key:      "config:feature-a",
		Value:    "on",
	}, got)
}

func TestRedisDeleteAndNotifyIgnoresMissingKey(t *testing.T) {
	server := NewServerForTest()
	baseRevision := testRevision(t, server)

	server.DeleteAndNotify("config:missing")

	assert.Equal(t, strconv.FormatUint(baseRevision, 10), mustGet(t, server, hubredis.RevisionKey))
}

func TestRedisDeleteAndNotifyUpdatesRevisionAndPublishes(t *testing.T) {
	server, client := newNetworkTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	server.SetAndNotify("config:feature-a", "on")
	pubsub := client.Subscribe(ctx, "config:feature-a")
	defer pubsub.Close()
	_, err := pubsub.Receive(ctx)
	require.NoError(t, err)

	baseRevision := testRevision(t, server)
	server.DeleteAndNotify("config:feature-a")

	_, ok := server.Get("config:feature-a")
	assert.False(t, ok)
	assert.Equal(t, strconv.FormatUint(baseRevision+1, 10), mustGet(t, server, hubredis.RevisionKey))

	message, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)

	var got hubredis.Event
	require.NoError(t, json.Unmarshal([]byte(message.Payload), &got))
	assert.Equal(t, hubredis.Event{
		Revision: baseRevision + 1,
		Kind:     hubredis.EventKindDelete,
		Key:      "config:feature-a",
	}, got)
}

func TestRedisNotifyBatchUsesSingleRevision(t *testing.T) {
	server, client := newNetworkTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	server.SetAndNotify("config:feature-old", "old")
	pubsub := client.PSubscribe(ctx, "config:*")
	defer pubsub.Close()
	_, err := pubsub.Receive(ctx)
	require.NoError(t, err)

	baseRevision := testRevision(t, server)
	batch := server.NotifyBatch()
	batch.Set("config:feature-a", "on")
	batch.Set("config:feature-b", "off")
	batch.Delete("config:feature-old")
	batch.Notify()

	assert.Equal(t, strconv.FormatUint(baseRevision+1, 10), mustGet(t, server, hubredis.RevisionKey))
	_, ok := server.Get("config:feature-old")
	assert.False(t, ok)

	events := make([]hubredis.Event, 0, 3)
	for range 3 {
		message, err := pubsub.ReceiveMessage(ctx)
		require.NoError(t, err)

		var event hubredis.Event
		require.NoError(t, json.Unmarshal([]byte(message.Payload), &event))
		events = append(events, event)
	}

	assert.Equal(t, []hubredis.Event{
		{
			Revision: baseRevision + 1,
			Kind:     hubredis.EventKindUpsert,
			Key:      "config:feature-a",
			Value:    "on",
		},
		{
			Revision: baseRevision + 1,
			Kind:     hubredis.EventKindUpsert,
			Key:      "config:feature-b",
			Value:    "off",
		},
		{
			Revision: baseRevision + 1,
			Kind:     hubredis.EventKindDelete,
			Key:      "config:feature-old",
		},
	}, events)
}

func TestRedisNotifyBatchRejectsReuseAfterNotify(t *testing.T) {
	server := NewServerForTest()
	batch := server.NotifyBatch()
	batch.Notify()

	assert.Panics(t, func() {
		batch.Set("config:feature-a", "on")
	})
	assert.Panics(t, func() {
		batch.Delete("config:feature-a")
	})
	assert.Panics(t, func() {
		batch.Notify()
	})
}

func TestRedisKeepEphemeralUpdatesTTL(t *testing.T) {
	server := NewServerForTest()
	ttl := 180 * time.Second
	server.SetEphemeralAndNotify("config:feature-a", "on", ttl)
	assert.True(t, server.Expire("config:feature-a", 1))

	server.KeepEphemeral("config:feature-a", ttl)

	got := server.TTL("config:feature-a")
	assert.True(t, got > 1)
	assert.True(t, got <= int(ttl/time.Second))
}

func TestRedisServerRejectsInvalidCommands(t *testing.T) {
	_, client := newNetworkTestServer(t)
	ctx := context.Background()

	assert.Error(t, client.Do(ctx, "MGET", "a", "b").Err())
	assert.Error(t, client.Do(ctx, "SET", "only-key").Err())
	assert.Error(t, client.Do(ctx, "SCAN", "1").Err())
	assert.Error(t, client.Do(ctx, "EXPIRE", "key", "bad").Err())
}

func TestRedisServerScanReturnsMatchingKeysOnce(t *testing.T) {
	server := NewServerForTest()

	server.Set("config:feature-a", "a")
	server.Set("config:feature-b", "b")
	server.Set("app:demo:status:1", "status")

	keys := server.Scan("config:*")

	assert.Equal(t, []string{"config:feature-a", "config:feature-b"}, keys)
}

func testRevision(t *testing.T, server *Server) uint64 {
	t.Helper()

	value := mustGet(t, server, hubredis.RevisionKey)
	revision, err := strconv.ParseUint(value, 10, 64)
	require.NoError(t, err)
	return revision
}

func mustGet(t *testing.T, server *Server, key string) string {
	t.Helper()

	value, ok := server.Get(key)
	require.True(t, ok)
	return value
}

func requireSubscription(t *testing.T, value any, kind string, channel string, count int) {
	t.Helper()

	subscription, ok := value.(*redis.Subscription)
	require.True(t, ok, "expected redis.Subscription, got %T", value)
	assert.Equal(t, kind, subscription.Kind)
	assert.Equal(t, channel, subscription.Channel)
	assert.Equal(t, count, subscription.Count)
}

func isPubSubTimeout(err error) bool {
	var netErr net.Error
	return errors.Is(err, redis.Nil) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) && netErr.Timeout()
}
