package epmgr

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
)

const testService = "demo.Service"

func singleEndpointManager(t *testing.T) *Manager {
	t.Helper()
	m := newTestManager(t, map[string]string{
		testRpcRegistrationKey(testService, uuid.MustParse("00000000-0000-0000-0000-00000000000a")): testRpcRegistrationValue(testService, uuid.MustParse("00000000-0000-0000-0000-00000000000a"), "http://old-ip"),
	})
	w := m.WatchRpc(testService)
	t.Cleanup(w.Release)
	return m
}

func acquireAttempt(t *testing.T, m *Manager) *Attempt {
	t.Helper()
	registration, attempt, configured := m.NextRpcEndpoint(testService)
	require.True(t, configured)
	require.NotNil(t, registration)
	require.NotNil(t, attempt)
	return attempt
}

func requireIsolated(t *testing.T, m *Manager) {
	t.Helper()
	registration, attempt, configured := m.NextRpcEndpoint(testService)
	require.True(t, configured)
	require.Nil(t, registration)
	require.Nil(t, attempt)
}

func TestIsolationSkipsFailedEndpointWithoutDeletingRegistration(t *testing.T) {
	m := newTestManager(t, map[string]string{
		testRpcRegistrationKey(testService, uuid.MustParse("00000000-0000-0000-0000-00000000006b")): testRpcRegistrationValue(testService, uuid.MustParse("00000000-0000-0000-0000-00000000006b"), "http://one"),
		testRpcRegistrationKey(testService, uuid.MustParse("00000000-0000-0000-0000-00000000006d")): testRpcRegistrationValue(testService, uuid.MustParse("00000000-0000-0000-0000-00000000006d"), "http://two"),
	})
	w := m.WatchRpc(testService)
	t.Cleanup(w.Release)
	failed, attempt, _ := m.NextRpcEndpoint(testService)
	attempt.Finish(&net.OpError{Op: "dial", Err: syscall.ECONNREFUSED})
	for range 10 {
		next, attempt, configured := m.NextRpcEndpoint(testService)
		require.True(t, configured)
		require.NotNil(t, next)
		require.NotEqual(t, failed.Endpoint, next.Endpoint)
		attempt.Finish(nil)
	}
	require.Len(t, m.routesByPrefix[watched.FormatRpcServiceRegistrationPrefix(testService)].registrationsByKey, 2)
}

func TestCooldownAllowsOneProbeAndIgnoresLateCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := singleEndpointManager(t)
		failed := acquireAttempt(t, m)
		late := acquireAttempt(t, m)
		failed.Finish(syscall.ECONNREFUSED)
		late.Finish(nil)
		requireIsolated(t, m)
		synctest.Sleep(initialCooldown)
		probes := make(chan *Attempt, 64)
		var wait sync.WaitGroup
		for range cap(probes) {
			wait.Go(func() {
				_, attempt, _ := m.NextRpcEndpoint(testService)
				if attempt != nil {
					probes <- attempt
				}
			})
		}
		wait.Wait()
		require.Len(t, probes, 1)
		probe := <-probes
		probe.Finish(io.ErrUnexpectedEOF)
		synctest.Sleep(initialCooldown)
		requireIsolated(t, m)
		synctest.Sleep(initialCooldown)
		recovery := acquireAttempt(t, m)
		body := recovery.TrackBody(io.NopCloser(strings.NewReader("business error response")))
		_, err := io.ReadAll(body)
		require.NoError(t, err)
		require.NoError(t, body.Close())
		acquireAttempt(t, m).Finish(nil)
	})
}

func TestTransportFailureThresholdAndRequestErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		err       error
		threshold int
	}{
		{name: "dial timeout", err: &net.OpError{Op: "dial", Err: context.DeadlineExceeded}, threshold: 1},
		{name: "truncated response", err: io.ErrUnexpectedEOF, threshold: 3},
		{name: "reset connection", err: syscall.ECONNRESET, threshold: 3},
		{name: "canceled call", err: context.Canceled},
		{name: "overall deadline", err: context.DeadlineExceeded},
		{name: "invalid request", err: errors.New("invalid forward request")},
		{name: "wrapped invalid request", err: &url.Error{Op: "Post", Err: errors.New("invalid forward request")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := singleEndpointManager(t)
			if tc.threshold == 0 {
				for range 4 {
					acquireAttempt(t, m).Finish(tc.err)
				}
				acquireAttempt(t, m).Finish(nil)
				return
			}
			for range tc.threshold {
				acquireAttempt(t, m).Finish(tc.err)
			}
			requireIsolated(t, m)
		})
	}
}

func TestSuccessfulTransportResetsConsecutiveFailures(t *testing.T) {
	m := singleEndpointManager(t)
	for range 2 {
		acquireAttempt(t, m).Finish(io.ErrUnexpectedEOF)
	}
	acquireAttempt(t, m).Finish(nil)
	for range 2 {
		acquireAttempt(t, m).Finish(io.ErrUnexpectedEOF)
	}
	acquireAttempt(t, m).Finish(nil)
}

func TestReplacementAndRemovalRetireOldAttempt(t *testing.T) {
	m := singleEndpointManager(t)
	prefix := watched.FormatRpcServiceRegistrationPrefix(testService)
	key := testRpcRegistrationKey(testService, uuid.MustParse("00000000-0000-0000-0000-00000000000a"))
	old := acquireAttempt(t, m)
	replacement := testRpcRegistrationValue(testService, uuid.MustParse("00000000-0000-0000-0000-00000000000a"), "http://new-ip")
	m.handleRegistrationEvent(prefix, hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: key, Value: replacement})
	old.Finish(syscall.ECONNREFUSED)
	registration, next, _ := m.NextRpcEndpoint(testService)
	require.Equal(t, "http://new-ip", registration.Endpoint)
	next.Finish(syscall.ECONNREFUSED)
	m.handleRegistrationEvent(prefix, hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: key, Value: replacement})
	requireIsolated(t, m)
	m.handleRegistrationEvent(prefix, hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: key})
	m.handleRegistrationEvent(prefix, hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: key, Value: replacement})
	acquireAttempt(t, m).Finish(nil)
}

func TestProbeAbandonmentAndCooldownCap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := singleEndpointManager(t)
		acquireAttempt(t, m).Finish(syscall.ECONNREFUSED)
		synctest.Sleep(initialCooldown)
		probe := acquireAttempt(t, m)
		probe.Finish(context.Canceled)
		requireIsolated(t, m)
		synctest.Sleep(initialCooldown)
		for _, cooldown := range []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second, 30 * time.Second} {
			acquireAttempt(t, m).Finish(syscall.ECONNREFUSED)
			synctest.Sleep(cooldown - time.Nanosecond)
			requireIsolated(t, m)
			synctest.Sleep(time.Nanosecond)
		}
		acquireAttempt(t, m).Finish(nil)
	})
}
