package desc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

func TestAdmissionObserverReceivesInitialStateAndStopsOnCancellation(t *testing.T) {
	target := newTestManager(t, map[string]string{watched.FormatDescriptorActorKey("demo.User"): vcode.MustMarshalJsonS(skeldesc.Actor{SkelName: "demo.User", Hash: "initial"})})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var snapshots []AdmissionSnapshot
	target.ObserveAdmission(ctx, func(snapshot AdmissionSnapshot) { snapshots = append(snapshots, snapshot) })
	require.Len(t, snapshots, 1)
	target.handleActorEvent(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorActorKey("demo.User"), Value: vcode.MustMarshalJsonS(skeldesc.Actor{SkelName: "demo.User", Hash: "updated"})})
	require.Len(t, snapshots, 2)
	require.Equal(t, "initial", snapshots[0].Actors["demo.User"].Hash)
	require.Equal(t, "updated", snapshots[1].Actors["demo.User"].Hash)
	cancel()
	target.handleActorEvent(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: watched.FormatDescriptorActorKey("demo.User")})
	require.Len(t, snapshots, 2)
	_, ok := target.Actor("demo.User")
	require.False(t, ok)
}

func TestAdmissionCallbacksRunOutsideTheCacheLock(t *testing.T) {
	manager := newTestManager(t, nil)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		manager.ObserveAdmission(t.Context(), func(AdmissionSnapshot) {
			manager.Actor("demo.User")
			close(entered)
			<-release
		})
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("observer cannot read the cache")
	}
	read := make(chan struct{})
	go func() { manager.Snapshot(); close(read) }()
	select {
	case <-read:
	case <-time.After(time.Second):
		t.Fatal("observer blocked cache reads")
	}
	close(release)
	<-done
}
