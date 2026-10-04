package syncer

import (
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"testing"
)

func TestWebSchemasPublicationLifecycle(t *testing.T) {
	server := watchserver.NewServerForTest()
	t.Cleanup(server.AfterAppStop)
	target := testSyncer(server)
	views := []core.DomainSchemaView{{Webs: []core.SchemaVersion[*skel.WebSchema]{
		{Schema: new(skel.WebSchema{SkelName: "demo.Web", Hash: "first", AuthMode: skel.AuthModeRequired}), Main: true},
		{Schema: new(skel.WebSchema{SkelName: "demo.OldWeb", Hash: "old"}), Main: false},
	}}}
	target.SyncSchemas(views)
	key := watched.FormatSchemaWebKey("demo.Web")
	value, ok := server.Get(key)
	require.True(t, ok)
	require.Equal(t, skel.AuthModeRequired, vcode.MustUnmarshalJsonS[*skel.WebSchema](value).AuthMode)
	_, ok = server.Get(watched.FormatSchemaWebKey("demo.OldWeb"))
	require.False(t, ok)
	revision := testWatchRevision(t, server)
	target.SyncSchemas(views)
	require.Equal(t, revision, testWatchRevision(t, server))
	views[0].Webs[0].Schema.Hash = "second"
	views[0].Webs[0].Schema.AuthMode = skel.AuthModeOff
	target.SyncSchemas(views)
	value, ok = server.Get(key)
	require.True(t, ok)
	require.Equal(t, skel.AuthModeOff, vcode.MustUnmarshalJsonS[*skel.WebSchema](value).AuthMode)
	target.SyncSchemas(nil)
	_, ok = server.Get(key)
	require.False(t, ok)
	target.WriteSchemas(views)
	target.SyncSchemas(nil)
	_, ok = server.Get(key)
	require.True(t, ok, "WriteSchemas must not join the application diff/delete lifecycle")
}
