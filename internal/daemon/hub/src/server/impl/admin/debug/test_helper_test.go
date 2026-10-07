package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type messageDebugRepo struct {
	core.RegistryRepo
	core.DescriptorRepo
	statuses []*core.AppStatus
	events   []core.DescriptorVersion[*skeldesc.Event]
	tasks    []core.DescriptorVersion[*skeldesc.Task]
}

func (r *messageDebugRepo) ListAppStatuses() []*core.AppStatus {
	return r.statuses
}
func (r *messageDebugRepo) ListEventDescriptorVersions() []core.DescriptorVersion[*skeldesc.Event] {
	return r.events
}
func (r *messageDebugRepo) ListTaskDescriptorVersions() []core.DescriptorVersion[*skeldesc.Task] {
	return r.tasks
}

func newMessageDebugServices() (*EventDebugApiServiceServerImpl, *TaskDebugApiServiceServerImpl) {
	repo := &messageDebugRepo{}
	for _, item := range [][2]string{{"demo.Z", "z"}, {"demo.A", "v2"}, {"demo.A", "v1"}} {
		name, hash := item[0], item[1]
		repo.events = append(repo.events, core.DescriptorVersion[*skeldesc.Event]{DescriptorHash: hash, Descriptor: &skeldesc.Event{SkelName: name, Name: name + hash, Description: "event description", Members: []*skeldesc.Member{{Name: "message", Example: `"hello"`}}}})
		repo.tasks = append(repo.tasks, core.DescriptorVersion[*skeldesc.Task]{DescriptorHash: hash, Descriptor: &skeldesc.Task{SkelName: name, Name: name + hash, Description: "task description", Triggers: []*skeldesc.TaskTrigger{{SkelName: "Z"}, {SkelName: "A", Arguments: []*skeldesc.Member{{Name: "message", Example: `"hello"`}}}}}})
		repo.statuses = append(repo.statuses, &core.AppStatus{EventListeners: []core.EventListenerRegistration{{EventSkelName: name, DescriptorHash: hash}}, TaskRunners: []core.TaskRunnerRegistration{{TaskSkelName: name, DescriptorHash: hash}}})
	}
	repo.statuses = append(repo.statuses, repo.statuses[0])
	currentApp := meta.MustNewApp("vine.hub", "1.2.3", "123e4567-e89b-12d3-a456-426614174099")
	return &EventDebugApiServiceServerImpl{RegistryRepo: repo, DescriptorRepo: repo, CurrentApp: currentApp}, &TaskDebugApiServiceServerImpl{RegistryRepo: repo, DescriptorRepo: repo, CurrentApp: currentApp}
}

func requireDebugError(t *testing.T, code ex.Code, invoke func()) {
	t.Helper()
	var caught any
	func() { defer func() { caught = recover() }(); invoke() }()
	err, ok := caught.(ex.Error)
	require.True(t, ok, "expected Vine error, got %#v", caught)
	assert.Equal(t, code, err.Code())
}
