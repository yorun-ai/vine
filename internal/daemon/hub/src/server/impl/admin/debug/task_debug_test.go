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

func TestListTasksAndTriggersDeduplicateAndSort(t *testing.T) {
	_, service := newMessageDebugServices()
	items := service.ListTasks()
	require.Len(t, items, 3)
	assert.Equal(t, []string{"demo.A/v1", "demo.A/v2", "demo.Z/z"}, []string{items[0].TaskSkelName + "/" + items[0].DescriptorHash, items[1].TaskSkelName + "/" + items[1].DescriptorHash, items[2].TaskSkelName + "/" + items[2].DescriptorHash})
	assert.Equal(t, "demo.Av1", items[0].Name)
	assert.Equal(t, "task description", items[0].Description)
	triggers := service.ListTriggers("demo.A", "v1")
	require.Len(t, triggers, 2)
	assert.Equal(t, "A", triggers[0].SkelName)
	assert.Equal(t, "Z", triggers[1].SkelName)
	require.Len(t, triggers[0].Arguments, 1)
	assert.Equal(t, "message", triggers[0].Arguments[0].Name)
}

func TestBuildDefaultLaunchRequestSelectsDescriptorAndTrigger(t *testing.T) {
	_, service := newMessageDebugServices()
	request := service.BuildDefaultLaunchRequest("demo.A", "v1", "A")
	assert.JSONEq(t, `{"message":"hello"}`, string(request.ArgumentsJson))
	_, err := meta.NewTrace(request.TraceId, request.SpanId)
	assert.NoError(t, err)
	requireDebugError(t, ex.NotFound, func() { service.BuildDefaultLaunchRequest("demo.A", "unknown", "A") })
	requireDebugError(t, ex.NotFound, func() { service.BuildDefaultLaunchRequest("demo.A", "v1", "missing") })
}

func TestLaunchTaskRejectsInvalidRequestBeforePublishing(t *testing.T) {
	_, service := newMessageDebugServices()
	for _, item := range []struct {
		name    string
		request skeled.TaskDebugLaunchRequest
		code    ex.Code
	}{
		{"missing runner", skeled.TaskDebugLaunchRequest{TaskSkelName: "missing", ArgumentsJson: skeltype.JSON(`{}`)}, ex.NotFound},
		{"wrong version", skeled.TaskDebugLaunchRequest{TaskSkelName: "demo.A", DescriptorHash: "unknown", ArgumentsJson: skeltype.JSON(`{}`)}, ex.NotFound},
		{"invalid JSON", skeled.TaskDebugLaunchRequest{TaskSkelName: "demo.A", DescriptorHash: "v1", ArgumentsJson: skeltype.JSON(`{`)}, ex.InvalidRequest},
		{"invalid trace", skeled.TaskDebugLaunchRequest{TaskSkelName: "demo.A", ArgumentsJson: skeltype.JSON(`{}`), TraceId: new("invalid")}, ex.InvalidRequest},
	} {
		t.Run(item.name, func(t *testing.T) { requireDebugError(t, item.code, func() { service.LaunchTask(item.request) }) })
	}
}
