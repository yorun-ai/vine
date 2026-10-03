package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type messageDebugRepo struct {
	core.RegistryRepo
	core.SchemaRepo
	statuses []*core.AppStatus
	events   []core.SchemaVersion[*skel.EventSchema]
	tasks    []core.SchemaVersion[*skel.TaskSchema]
}

func (r *messageDebugRepo) ListAppStatuses() []*core.AppStatus { return r.statuses }
func (r *messageDebugRepo) ListEventSchemaVersions() []core.SchemaVersion[*skel.EventSchema] {
	return r.events
}
func (r *messageDebugRepo) ListTaskSchemaVersions() []core.SchemaVersion[*skel.TaskSchema] {
	return r.tasks
}

func newMessageDebugServices() (*EventDebugApiServiceServerImpl, *TaskDebugApiServiceServerImpl) {
	repo := &messageDebugRepo{}
	for _, item := range [][2]string{{"demo.Z", "z"}, {"demo.A", "v2"}, {"demo.A", "v1"}} {
		name, hash := item[0], item[1]
		repo.events = append(repo.events, core.SchemaVersion[*skel.EventSchema]{SchemaHash: hash, Schema: &skel.EventSchema{SkelName: name, Name: name + hash, Description: "event description", Members: []*skel.MemberSchema{{Name: "message", Example: `"hello"`}}}})
		repo.tasks = append(repo.tasks, core.SchemaVersion[*skel.TaskSchema]{SchemaHash: hash, Schema: &skel.TaskSchema{SkelName: name, Name: name + hash, Description: "task description", Triggers: []*skel.TriggerSchema{{SkelName: "Z"}, {SkelName: "A", Arguments: []*skel.MemberSchema{{Name: "message", Example: `"hello"`}}}}}})
		repo.statuses = append(repo.statuses, &core.AppStatus{EventListeners: []core.EventListenerRegistration{{EventSkelName: name, SchemaHash: hash}}, TaskRunners: []core.TaskRunnerRegistration{{TaskSkelName: name, SchemaHash: hash}}})
	}
	repo.statuses = append(repo.statuses, repo.statuses[0])
	currentApp := meta.MustNewApp("vine.hub", "1.2.3", "123e4567-e89b-12d3-a456-426614174099")
	return &EventDebugApiServiceServerImpl{RegistryRepo: repo, SchemaRepo: repo, CurrentApp: currentApp}, &TaskDebugApiServiceServerImpl{RegistryRepo: repo, SchemaRepo: repo, CurrentApp: currentApp}
}

func requireDebugError(t *testing.T, code ex.Code, invoke func()) {
	t.Helper()
	var caught any
	func() { defer func() { caught = recover() }(); invoke() }()
	err, ok := caught.(ex.Error)
	require.True(t, ok, "expected Vine error, got %#v", caught)
	assert.Equal(t, code, err.Code())
}
