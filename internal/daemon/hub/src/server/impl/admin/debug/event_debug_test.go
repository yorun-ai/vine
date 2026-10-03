package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
)

func TestListEventsDeduplicatesAndSortsSchemaVersions(t *testing.T) {
	service, _ := newMessageDebugServices()
	items := service.ListEvents()
	require.Len(t, items, 3)
	assert.Equal(t, []string{"demo.A/v1", "demo.A/v2", "demo.Z/z"}, []string{items[0].EventSkelName + "/" + items[0].SchemaHash, items[1].EventSkelName + "/" + items[1].SchemaHash, items[2].EventSkelName + "/" + items[2].SchemaHash})
	assert.Equal(t, "demo.Av1", items[0].Name)
	assert.Equal(t, "event description", items[0].Description)
	require.Len(t, items[0].Fields, 1)
	assert.Equal(t, "message", items[0].Fields[0].Name)
}

func TestBuildDefaultEmitRequestSelectsSchemaVersion(t *testing.T) {
	service, _ := newMessageDebugServices()
	request := service.BuildDefaultEmitRequest("demo.A", "v1")
	assert.JSONEq(t, `{"message":"hello"}`, string(request.EventJson))
	_, err := meta.NewTrace(request.TraceId, request.SpanId)
	assert.NoError(t, err)
	requireDebugError(t, ex.NotFound, func() { service.BuildDefaultEmitRequest("demo.A", "unknown") })
	requireDebugError(t, ex.NotFound, func() { service.BuildDefaultEmitRequest("missing", "") })
}

func TestEmitEventRejectsInvalidRequestBeforePublishing(t *testing.T) {
	service, _ := newMessageDebugServices()
	for _, item := range []struct {
		name    string
		request skeled.EventDebugEmitRequest
		code    ex.Code
	}{
		{"missing listener", skeled.EventDebugEmitRequest{EventSkelName: "missing", EventJson: skel.JSON(`{}`)}, ex.NotFound},
		{"wrong version", skeled.EventDebugEmitRequest{EventSkelName: "demo.A", SchemaHash: "unknown", EventJson: skel.JSON(`{}`)}, ex.NotFound},
		{"invalid JSON", skeled.EventDebugEmitRequest{EventSkelName: "demo.A", SchemaHash: "v1", EventJson: skel.JSON(`{`)}, ex.InvalidRequest},
		{"invalid trace", skeled.EventDebugEmitRequest{EventSkelName: "demo.A", EventJson: skel.JSON(`{}`), TraceId: new("invalid")}, ex.InvalidRequest},
	} {
		t.Run(item.name, func(t *testing.T) { requireDebugError(t, item.code, func() { service.EmitEvent(item.request) }) })
	}
}
