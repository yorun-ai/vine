package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
)

func TestListEventsDeduplicatesAndSortsDescriptorVersions(t *testing.T) {
	service, _ := newMessageDebugServices()
	items := service.ListEvents()
	require.Len(t, items, 3)
	assert.Equal(t, []string{"demo.A/v1", "demo.A/v2", "demo.Z/z"}, []string{items[0].EventSkelName + "/" + items[0].DescriptorHash, items[1].EventSkelName + "/" + items[1].DescriptorHash, items[2].EventSkelName + "/" + items[2].DescriptorHash})
	assert.Equal(t, "demo.Av1", items[0].Name)
	assert.Equal(t, "event description", items[0].Description)
	require.Len(t, items[0].Fields, 1)
	assert.Equal(t, "message", items[0].Fields[0].Name)
}

func TestBuildDefaultEmitRequestSelectsDescriptorVersion(t *testing.T) {
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
		{"missing listener", skeled.EventDebugEmitRequest{EventSkelName: "missing", EventJson: skeltype.JSON(`{}`)}, ex.NotFound},
		{"wrong version", skeled.EventDebugEmitRequest{EventSkelName: "demo.A", DescriptorHash: "unknown", EventJson: skeltype.JSON(`{}`)}, ex.NotFound},
		{"invalid JSON", skeled.EventDebugEmitRequest{EventSkelName: "demo.A", DescriptorHash: "v1", EventJson: skeltype.JSON(`{`)}, ex.InvalidRequest},
		{"invalid trace", skeled.EventDebugEmitRequest{EventSkelName: "demo.A", EventJson: skeltype.JSON(`{}`), TraceId: new("invalid")}, ex.InvalidRequest},
	} {
		t.Run(item.name, func(t *testing.T) { requireDebugError(t, item.code, func() { service.EmitEvent(item.request) }) })
	}
}
