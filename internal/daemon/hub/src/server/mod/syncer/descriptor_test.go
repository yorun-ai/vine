package syncer

import (
	"testing"

	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func TestWebDescriptorsPublicationLifecycle(t *testing.T) {
	server := watchserver.NewServerForTest()
	t.Cleanup(server.AfterAppStop)
	target := testSyncer(server)
	views := []core.DomainDescriptorView{{Webs: []core.DescriptorVersion[*skeldesc.Web]{
		{Descriptor: new(skeldesc.Web{SkelName: "demo.Web", Hash: "first", AuthMode: skeldesc.AuthModeRequired}), Main: true},
		{Descriptor: new(skeldesc.Web{SkelName: "demo.OldWeb", Hash: "old", AuthMode: skeldesc.AuthModeRequired}), Main: false},
	}}}
	target.SyncDescriptors(views)
	key := watched.FormatDescriptorWebKey("demo.Web")
	value, ok := server.Get(key)
	require.True(t, ok)
	require.Equal(t, skeldesc.AuthModeRequired, vcode.MustUnmarshalJsonS[*skeldesc.Web](value).AuthMode)
	_, ok = server.Get(watched.FormatDescriptorWebKey("demo.OldWeb"))
	require.False(t, ok)
	revision := testWatchRevision(t, server)
	target.SyncDescriptors(views)
	require.Equal(t, revision, testWatchRevision(t, server))
	views[0].Webs[0].Descriptor.Hash = "second"
	views[0].Webs[0].Descriptor.AuthMode = skeldesc.AuthModeOff
	target.SyncDescriptors(views)
	value, ok = server.Get(key)
	require.True(t, ok)
	require.Equal(t, skeldesc.AuthModeOff, vcode.MustUnmarshalJsonS[*skeldesc.Web](value).AuthMode)
	target.SyncDescriptors(nil)
	_, ok = server.Get(key)
	require.False(t, ok)
	target.WriteDescriptors(views)
	target.SyncDescriptors(nil)
	_, ok = server.Get(key)
	require.True(t, ok, "WriteDescriptors must not join the application diff/delete lifecycle")
}
