package scheduler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type _SchedulerRegistryRepo struct {
	core.RegistryRepo

	statuses []*core.AppStatus
}

func (r *_SchedulerRegistryRepo) ListAppStatuses() []*core.AppStatus {
	return r.statuses
}

type _SchedulerSchemaRepo struct {
	core.SchemaRepo

	taskVersions []core.SchemaVersion[*skel.TaskSchema]
}

func (r *_SchedulerSchemaRepo) ListTaskSchemaVersions() []core.SchemaVersion[*skel.TaskSchema] {
	return r.taskVersions
}

type _SchedulerTaskPublisher struct {
	messages []taskspec.NATSMessage
}

func (p *_SchedulerTaskPublisher) PublishTask(message taskspec.NATSMessage) {
	p.messages = append(p.messages, message)
}

func TestSchedulerRefreshSchedulesDeduplicatesAppInstances(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	target := newTestScheduler([]*core.AppStatus{
		newTestScheduledAppStatus("instance-1"),
		newTestScheduledAppStatus("instance-2"),
	}, publisher)

	target.refreshSchedules()

	assert.Len(t, target.jobs, 1)
}

func TestSchedulerRefreshSchedulesAddsMultipleCronSchedulers(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	status := newTestScheduledAppStatus("instance-1")
	status.TaskRunners[0].CronSchedulers = append(status.TaskRunners[0].CronSchedulers, core.TaskRunnerCronScheduler{
		TriggerSkelName: "rebuild",
		CronExpr:        "30 * * * *",
	})
	target := newTestScheduler([]*core.AppStatus{status}, publisher)

	target.refreshSchedules()

	assert.Len(t, target.jobs, 2)
}

func TestSchedulerRefreshSchedulesRemovesMissingRegistrations(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	registryRepo := &_SchedulerRegistryRepo{statuses: []*core.AppStatus{newTestScheduledAppStatus("instance-1")}}
	target := newTestSchedulerWithRegistry(registryRepo, publisher)

	target.refreshSchedules()
	registryRepo.statuses = []*core.AppStatus{}
	target.refreshSchedules()

	assert.Empty(t, target.jobs)
}

func TestSchedulerPublishSchedulePublishesTaskMessage(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	target := newTestScheduler([]*core.AppStatus{newTestScheduledAppStatus("instance-1")}, publisher)

	target.publishSchedule(newTestScheduleConfig())

	require.Len(t, publisher.messages, 1)
	assert.Equal(t, schedulerClientName, publisher.messages[0].Metadata.AppName)
	assert.Equal(t, "demo.booker.RebuildCatalogIndexTask", publisher.messages[0].TaskSkelName)
	assert.Equal(t, "rebuild", publisher.messages[0].TriggerSkelName)
	assert.Equal(t, "{}", publisher.messages[0].ArgumentsJson)
}

func TestSchedulerPublishScheduleSkipsInactiveRunner(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	target := newTestScheduler([]*core.AppStatus{}, publisher)

	target.publishSchedule(newTestScheduleConfig())

	assert.Empty(t, publisher.messages)
}

func TestSchedulerRejectsCronSchedulerTriggerWithArguments(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	target := newTestScheduler([]*core.AppStatus{newTestScheduledAppStatus("instance-1")}, publisher)
	target.SchemaRepo = &_SchedulerSchemaRepo{taskVersions: []core.SchemaVersion[*skel.TaskSchema]{{
		SchemaHash: "task-hash",
		Schema: &skel.TaskSchema{
			SkelName: "demo.booker.RebuildCatalogIndexTask",
			Triggers: []*skel.TriggerSchema{{
				SkelName: "rebuild",
				Arguments: []*skel.MemberSchema{{
					Name: "full",
					Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarBool},
				}},
			}},
		},
	}}}

	assert.Panics(t, func() {
		target.refreshSchedules()
	})
}

func newTestScheduler(statuses []*core.AppStatus, publisher *_SchedulerTaskPublisher) *Scheduler {
	return newTestSchedulerWithRegistry(&_SchedulerRegistryRepo{statuses: statuses}, publisher)
}

func newTestSchedulerWithRegistry(registryRepo *_SchedulerRegistryRepo, publisher *_SchedulerTaskPublisher) *Scheduler {
	target := &Scheduler{
		RegistryRepo: registryRepo,
		SchemaRepo: &_SchedulerSchemaRepo{taskVersions: []core.SchemaVersion[*skel.TaskSchema]{{
			SchemaHash: "task-hash",
			Schema: &skel.TaskSchema{
				SkelName: "demo.booker.RebuildCatalogIndexTask",
				Triggers: []*skel.TriggerSchema{{
					SkelName: "rebuild",
				}},
			},
		}}},
		publisher: publisher,
	}
	target.DIInit()
	return target
}

func newTestScheduledAppStatus(instanceId string) *core.AppStatus {
	return &core.AppStatus{
		Name:       "booker",
		InstanceId: instanceId,
		TaskRunners: []core.TaskRunnerRegistration{{
			TaskSkelName: "demo.booker.RebuildCatalogIndexTask",
			SchemaHash:   "task-hash",
			CronSchedulers: []core.TaskRunnerCronScheduler{{
				TriggerSkelName: "rebuild",
				CronExpr:        "0 * * * *",
			}},
		}},
	}
}

func newTestScheduleConfig() _ScheduleConfig {
	return _ScheduleConfig{
		AppName:         "booker",
		TaskSkelName:    "demo.booker.RebuildCatalogIndexTask",
		SchemaHash:      "task-hash",
		TriggerSkelName: "rebuild",
		CronExpr:        "0 * * * *",
	}
}
