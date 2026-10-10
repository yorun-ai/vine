package desc

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
)

func dataRef(name string) *skeldesc.Type {
	return new(skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: name})
}

func apiDescriptor(name string, result *skeldesc.Type) *skeldesc.Service {
	return new(skeldesc.Service{Api: true, SkelName: name, Methods: []*skeldesc.Method{{Name: "read", ResultType: result}}})
}

func testTypeValues() map[string]string {
	return map[string]string{
		watched.FormatDescriptorServiceKey("demo.API"):      vcode.MustMarshalJsonS(apiDescriptor("demo.API", new(skeldesc.Type{Kind: skeldesc.TypeKindList, Element: new(skeldesc.Type{Kind: skeldesc.TypeKindMap, Key: new(skeldesc.Type{Kind: skeldesc.TypeKindEnum, SkelName: "demo.Status"}), Value: new(skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: "demo.Wrapper", TypeArguments: []*skeldesc.Type{dataRef("demo.User")}})})}))),
		watched.FormatDescriptorServiceKey("demo.Shared"):   vcode.MustMarshalJsonS(apiDescriptor("demo.Shared", dataRef("demo.User"))),
		watched.FormatDescriptorServiceKey("demo.Internal"): vcode.MustMarshalJsonS(skeldesc.Service{SkelName: "demo.Internal", Methods: []*skeldesc.Method{{ResultType: dataRef("demo.Unused")}}}),
		watched.FormatDescriptorDataKey("demo.Wrapper"):     vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Wrapper", TypeParameters: []string{"T"}, Members: []*skeldesc.Member{{Name: "item", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindTypeParameter, Name: "T"})}}}),
		watched.FormatDescriptorDataKey("demo.User"):        vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.User", Members: []*skeldesc.Member{{Name: "address", Type: dataRef("demo.Address")}}}),
		watched.FormatDescriptorDataKey("demo.Address"):     vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Address", Members: []*skeldesc.Member{{Name: "owner", Type: dataRef("demo.User")}}}),
		watched.FormatDescriptorDataKey("demo.Unused"):      vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Unused"}),
		watched.FormatDescriptorEnumKey("demo.Status"):      vcode.MustMarshalJsonS(skeldesc.Enum{SkelName: "demo.Status"}),
	}
}

func publishDescriptor(t *testing.T, a *Manager, key string, value any) {
	t.Helper()
	a.Watch.(*watchtest.Client).Publish(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: key, Value: vcode.MustMarshalJsonS(value)})
}

func TestAPIDependencyCacheTracksReachabilityAndSharedCycles(t *testing.T) {
	a := newTestManager(t, testTypeValues())
	initial := a.Snapshot()
	require.Len(t, initial.Services, 2)
	require.Len(t, initial.Data, 3)
	require.Len(t, initial.Enums, 1)
	require.NotContains(t, initial.Data, "demo.Unused")
	require.Empty(t, initial.MissingKeys)
	require.Len(t, a.typeSubscription.keys, 4, "all dependency keys share a subscription")

	// A Data change adds a dependency without any Service changing.
	publishDescriptor(t, a, watched.FormatDescriptorDataKey("demo.User"), skeldesc.Data{SkelName: "demo.User", Hash: "updated", Members: []*skeldesc.Member{{Name: "role", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindEnum, SkelName: "demo.Role"})}}})
	require.Eventually(t, func() bool {
		s := a.Snapshot()
		return s.Data["demo.User"].Hash == "updated" && len(s.MissingKeys) == 1
	}, time.Second, time.Millisecond)
	require.NotContains(t, a.Snapshot().Data, "demo.Address")
	require.Contains(t, initial.Data, "demo.Address", "previous snapshot remains stable")
	publishDescriptor(t, a, watched.FormatDescriptorEnumKey("demo.Role"), skeldesc.Enum{SkelName: "demo.Role", Items: []*skeldesc.EnumItem{{Name: "ADMIN"}}})
	require.Eventually(t, func() bool { return a.Snapshot().Enums["demo.Role"] != nil }, time.Second, time.Millisecond)

	// Adding a Service dependency reads an already-published value.
	a.Watch.(*watchtest.Client).SetValue(watched.FormatDescriptorDataKey("demo.Other"), vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Other"}))
	publishDescriptor(t, a, watched.FormatDescriptorServiceKey("demo.API"), apiDescriptor("demo.API", dataRef("demo.Other")))
	require.Eventually(t, func() bool { return a.Snapshot().Data["demo.Other"] != nil }, time.Second, time.Millisecond)
	require.Contains(t, a.Snapshot().Data, "demo.User", "shared root keeps its dependencies")
	require.NotContains(t, a.Snapshot().Data, "demo.Wrapper")
	require.NotContains(t, a.Snapshot().Enums, "demo.Status")

	a.Watch.(*watchtest.Client).Publish(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: watched.FormatDescriptorServiceKey("demo.API")})
	require.Eventually(t, func() bool { return len(a.Snapshot().Services) == 1 }, time.Second, time.Millisecond)
	require.NotContains(t, a.Snapshot().Data, "demo.Other")
	require.Contains(t, a.Snapshot().Enums, "demo.Role")
	a.Watch.(*watchtest.Client).Publish(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: watched.FormatDescriptorServiceKey("demo.Shared")})
	require.Eventually(t, func() bool { return len(a.Snapshot().Services) == 0 }, time.Second, time.Millisecond)
	require.Empty(t, a.Snapshot().Data)
	require.Empty(t, a.Snapshot().Enums)
}

func TestDependencyDeletionReappearanceAndStaleSubscription(t *testing.T) {
	a := newTestManager(t, map[string]string{
		watched.FormatDescriptorServiceKey("demo.API"): vcode.MustMarshalJsonS(apiDescriptor("demo.API", dataRef("demo.User"))),
	})
	require.Equal(t, []string{watched.FormatDescriptorDataKey("demo.User")}, a.Snapshot().MissingKeys)
	publishDescriptor(t, a, watched.FormatDescriptorDataKey("demo.User"), skeldesc.Data{SkelName: "demo.User", Hash: "first"})
	require.Eventually(t, func() bool { return a.Snapshot().Data["demo.User"] != nil }, time.Second, time.Millisecond)
	a.Watch.(*watchtest.Client).Publish(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: watched.FormatDescriptorDataKey("demo.User")})
	require.Eventually(t, func() bool { return len(a.Snapshot().MissingKeys) == 1 }, time.Second, time.Millisecond)
	publishDescriptor(t, a, watched.FormatDescriptorDataKey("demo.User"), skeldesc.Data{SkelName: "demo.User", Hash: "restored"})
	require.Eventually(t, func() bool {
		s := a.Snapshot()
		return s.Data["demo.User"] != nil && s.Data["demo.User"].Hash == "restored"
	}, time.Second, time.Millisecond)

	a.mutex.Lock()
	old := a.typeSubscription
	a.mutex.Unlock()
	publishDescriptor(t, a, watched.FormatDescriptorServiceKey("demo.API"), apiDescriptor("demo.API", dataRef("demo.Other")))
	require.Eventually(t, func() bool {
		return len(a.Snapshot().MissingKeys) == 1 && a.Snapshot().MissingKeys[0] == watched.FormatDescriptorDataKey("demo.Other")
	}, time.Second, time.Millisecond)
	require.Error(t, old.ctx.Err())
	a.handleTypeEvent(old, hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorDataKey("demo.User"), Value: vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.User", Hash: "stale"})})
	require.NotContains(t, a.Snapshot().Data, "demo.User")
	// Turning an API into a backend service removes its retained graph.
	publishDescriptor(t, a, watched.FormatDescriptorServiceKey("demo.API"), skeldesc.Service{SkelName: "demo.API"})
	require.Eventually(t, func() bool { return len(a.Snapshot().Services) == 0 }, time.Second, time.Millisecond)
	require.Empty(t, a.Snapshot().MissingKeys)
}

type attachmentChangingWatch struct {
	hubwatch.ClientOps
	client  *watchtest.Client
	changed bool
}

func (w *attachmentChangingWatch) LoadKeysAndSubscribe(ctx context.Context, keys []string, handle func(hubwatch.Event)) (map[string]string, hubwatch.Subscription, error) {
	if !w.changed {
		w.changed = true
		w.client.SetValue(watched.FormatDescriptorDataKey("demo.User"), vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.User", Members: []*skeldesc.Member{{Name: "address", Type: dataRef("demo.Address")}}}))
	}
	return w.ClientOps.LoadKeysAndSubscribe(ctx, keys, handle)
}

func TestDependencySnapshotReconcilesChangesDuringAttachment(t *testing.T) {
	client := watchtest.New(t, map[string]string{
		watched.FormatDescriptorServiceKey("demo.API"):  vcode.MustMarshalJsonS(apiDescriptor("demo.API", dataRef("demo.User"))),
		watched.FormatDescriptorDataKey("demo.User"):    vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.User"}),
		watched.FormatDescriptorDataKey("demo.Address"): vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Address"}),
	})
	watch := &attachmentChangingWatch{ClientOps: client, client: client}
	a := &Manager{Context: t.Context(), Watch: watch}
	a.DIInit()
	t.Cleanup(a.AfterAppStop)
	require.Len(t, a.Snapshot().Data, 2, "attached snapshot expands the dependency graph")
	require.Len(t, a.typeSubscription.keys, 2)
	a.AfterAppStop()
	require.Empty(t, a.Snapshot().Data)
}

func TestDependencyTraversalIncludesInputDataAndEnumMembers(t *testing.T) {
	service := apiDescriptor("demo.API", nil)
	service.Methods[0].Arguments = []*skeldesc.Member{{Name: "input", Type: dataRef("demo.Settings")}}
	a := newTestManager(t, map[string]string{
		watched.FormatDescriptorServiceKey("demo.API"):     vcode.MustMarshalJsonS(service),
		watched.FormatDescriptorDataKey("demo.Settings"):   vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Settings", Members: []*skeldesc.Member{{Name: "status", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindEnum, SkelName: "demo.Status"})}}}),
		watched.FormatDescriptorEnumKey("demo.Status"):     vcode.MustMarshalJsonS(skeldesc.Enum{SkelName: "demo.Status"}),
		watched.FormatDescriptorConfigKey("demo.Settings"): "ignored config descriptor",
		watched.FormatDescriptorEventKey("demo.Created"):   "ignored event descriptor",
	})
	require.Len(t, a.Snapshot().Data, 1)
	require.Len(t, a.Snapshot().Enums, 1)
	require.Equal(t, []string{watched.FormatDescriptorDataKey("demo.Settings"), watched.FormatDescriptorEnumKey("demo.Status")}, a.typeSubscription.keys)
}

func TestVersionedAPIDependenciesStayIndependent(t *testing.T) {
	values := map[string]string{
		watched.FormatDescriptorDataKey("demo.Shared"): vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Shared"}),
	}
	for _, version := range []string{"", "V1", "V2", "V10"} {
		service, data, enum := "demo.OrderApiService"+version, "demo.Order"+version, "demo.Status"+version
		values[watched.FormatDescriptorServiceKey(service)] = vcode.MustMarshalJsonS(apiDescriptor(service, dataRef(data)))
		values[watched.FormatDescriptorDataKey(data)] = vcode.MustMarshalJsonS(skeldesc.Data{SkelName: data, Members: []*skeldesc.Member{
			{Name: "shared", Type: dataRef("demo.Shared")},
			{Name: "status", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindEnum, SkelName: enum})},
		}})
		values[watched.FormatDescriptorEnumKey(enum)] = vcode.MustMarshalJsonS(skeldesc.Enum{SkelName: enum, Hash: "initial"})
	}
	m := newTestManager(t, values)
	initial := m.Snapshot()
	require.Len(t, initial.Services, 4)
	require.Len(t, initial.Data, 5)
	require.Len(t, initial.Enums, 4)
	for name := range initial.Services {
		service, ok := m.Service(name)
		require.True(t, ok)
		require.Equal(t, name, service.SkelName)
	}
	publishDescriptor(t, m, watched.FormatDescriptorEnumKey("demo.StatusV1"), skeldesc.Enum{SkelName: "demo.StatusV1", Hash: "updated"})
	require.Eventually(t, func() bool { return m.Snapshot().Enums["demo.StatusV1"].Hash == "updated" }, time.Second, time.Millisecond)
	require.Equal(t, "initial", m.Snapshot().Enums["demo.StatusV10"].Hash)
	m.Watch.(*watchtest.Client).Publish(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: watched.FormatDescriptorServiceKey("demo.OrderApiServiceV1")})
	require.Eventually(t, func() bool {
		s := m.Snapshot()
		return !s.Syncing && len(s.Services) == 3
	}, time.Second, time.Millisecond)
	snapshot := m.Snapshot()
	require.NotContains(t, snapshot.Data, "demo.OrderV1")
	require.NotContains(t, snapshot.Enums, "demo.StatusV1")
	require.Contains(t, snapshot.Data, "demo.Shared")
	for _, version := range []string{"", "V2", "V10"} {
		require.Contains(t, snapshot.Services, "demo.OrderApiService"+version)
		require.Contains(t, snapshot.Data, "demo.Order"+version)
		require.Contains(t, snapshot.Enums, "demo.Status"+version)
	}
	require.Empty(t, snapshot.MissingKeys)
	require.Len(t, initial.Services, 4, "previous snapshots remain usable")
	require.Equal(t, "initial", initial.Enums["demo.StatusV1"].Hash)
}

type controlledTypeWatch struct {
	hubwatch.ClientOps
	before   func(context.Context, []string) error
	attempts atomic.Int32
}

func (w *controlledTypeWatch) LoadKeysAndSubscribe(ctx context.Context, keys []string, handle func(hubwatch.Event)) (map[string]string, hubwatch.Subscription, error) {
	w.attempts.Add(1)
	if err := w.before(ctx, keys); err != nil {
		return nil, nil, err
	}
	return w.ClientOps.LoadKeysAndSubscribe(ctx, keys, handle)
}

func newControlledManager(t *testing.T, before func(context.Context, []string) error) (*Manager, *controlledTypeWatch) {
	t.Helper()
	client := watchtest.New(t, map[string]string{
		watched.FormatDescriptorServiceKey("demo.API"): vcode.MustMarshalJsonS(apiDescriptor("demo.API", dataRef("demo.A"))),
		watched.FormatDescriptorDataKey("demo.A"):      vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.A"}),
		watched.FormatDescriptorDataKey("demo.B"):      vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.B"}),
		watched.FormatDescriptorDataKey("demo.C"):      vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.C"}),
	})
	watch := &controlledTypeWatch{ClientOps: client, before: before}
	manager := &Manager{Context: t.Context(), Watch: watch}
	manager.DIInit()
	t.Cleanup(manager.AfterAppStop)
	return manager, watch
}

func changeService(manager *Manager, result string) {
	manager.handleServiceEvent(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorServiceKey("demo.API"), Value: vcode.MustMarshalJsonS(apiDescriptor("demo.API", dataRef(result)))})
}

func TestDependencyRefreshKeepsReadsAvailableAndRejectsSupersededWork(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var candidateContext context.Context
	manager, _ := newControlledManager(t, func(ctx context.Context, keys []string) error {
		if !slices.Contains(keys, watched.FormatDescriptorDataKey("demo.B")) {
			return nil
		}
		candidateContext = ctx
		entered <- struct{}{}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			return nil
		}
	})
	initial := manager.Snapshot()
	changeService(manager, "demo.B")
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("refresh did not start")
	}
	readsDone := make(chan struct{})
	go func() {
		defer close(readsDone)
		service, ok := manager.Service("demo.API")
		assert.True(t, ok)
		assert.Equal(t, "demo.B", service.Methods[0].ResultType.SkelName)
		snapshot := manager.Snapshot()
		assert.True(t, snapshot.Syncing)
		assert.Contains(t, snapshot.Data, "demo.A")
		assert.Equal(t, "demo.A", snapshot.Services["demo.API"].Methods[0].ResultType.SkelName)
		changeService(manager, "demo.C")
	}()
	select {
	case <-readsDone:
	case <-time.After(time.Second):
		t.Fatal("refresh blocked admission reads or service updates")
	}
	close(release)
	require.Eventually(t, func() bool {
		snapshot := manager.Snapshot()
		return !snapshot.Syncing && snapshot.Data["demo.C"] != nil
	}, time.Second, time.Millisecond)
	require.Error(t, candidateContext.Err(), "superseded subscription is released")
	require.NotContains(t, manager.Snapshot().Data, "demo.B")
	require.Contains(t, initial.Data, "demo.A", "previous snapshots stay immutable")
}

func TestDependencyRefreshRetriesWithoutLosingPublishedCache(t *testing.T) {
	var fail atomic.Bool
	failed := make(chan struct{}, 1)
	manager, _ := newControlledManager(t, func(ctx context.Context, keys []string) error {
		if fail.Load() {
			select {
			case failed <- struct{}{}:
			default:
			}
			return errors.New("Hub unavailable")
		}
		return nil
	})
	manager.mutex.RLock()
	previous := manager.typeSubscription
	manager.mutex.RUnlock()
	fail.Store(true)
	changeService(manager, "demo.B")
	select {
	case <-failed:
	case <-time.After(time.Second):
		t.Fatal("refresh did not fail")
	}
	require.NoError(t, previous.ctx.Err(), "old subscription survives a failed attachment")
	require.True(t, manager.Snapshot().Syncing)
	require.Contains(t, manager.Snapshot().Data, "demo.A")
	// No further descriptor event is necessary to recover.
	fail.Store(false)
	require.Eventually(t, func() bool {
		snapshot := manager.Snapshot()
		return !snapshot.Syncing && snapshot.Data["demo.B"] != nil
	}, 2*time.Second, time.Millisecond)
	require.Eventually(t, func() bool { return previous.ctx.Err() != nil }, time.Second, time.Millisecond)
}

func TestServiceRevocationAndStopDoNotWaitForTypeIO(t *testing.T) {
	entered := make(chan struct{}, 1)
	manager, watch := newControlledManager(t, func(ctx context.Context, keys []string) error {
		if !slices.Contains(keys, watched.FormatDescriptorDataKey("demo.B")) {
			return nil
		}
		entered <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	})
	changeService(manager, "demo.B")
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("refresh did not start")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		manager.handleServiceEvent(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: watched.FormatDescriptorServiceKey("demo.API")})
		_, ok := manager.Service("demo.API")
		assert.False(t, ok, "revocation must take effect while type IO is stalled")
		manager.AfterAppStop()
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stopping did not cancel the in-flight refresh")
	}
	require.Equal(t, int32(2), watch.attempts.Load())
	require.Empty(t, manager.Snapshot().Data)
}

func TestStoppedDependencyWorkerDoesNotRetry(t *testing.T) {
	var fail atomic.Bool
	failed := make(chan struct{}, 1)
	manager, watch := newControlledManager(t, func(context.Context, []string) error {
		if fail.Load() {
			select {
			case failed <- struct{}{}:
			default:
			}
			return errors.New("Hub unavailable")
		}
		return nil
	})
	fail.Store(true)
	changeService(manager, "demo.B")
	select {
	case <-failed:
	case <-time.After(time.Second):
		t.Fatal("refresh did not start")
	}
	manager.AfterAppStop()
	attempts := watch.attempts.Load()
	select {
	case <-manager.syncDone:
	default:
		t.Fatal("worker did not exit")
	}
	require.Never(t, func() bool { return watch.attempts.Load() != attempts }, 250*time.Millisecond, 10*time.Millisecond)
}
