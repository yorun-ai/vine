package scheduler

import (
	"errors"
	"sync"
	"testing"
	"time"

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

type _PanickingSchedulerRegistryRepo struct {
	core.RegistryRepo
}

func (*_PanickingSchedulerRegistryRepo) ListAppStatuses() []*core.AppStatus {
	panic("registry unavailable")
}

type _BlockingSchedulerRegistryRepo struct {
	core.RegistryRepo

	mutex   sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *_BlockingSchedulerRegistryRepo) ListAppStatuses() []*core.AppStatus {
	r.mutex.Lock()
	r.calls++
	call := r.calls
	r.mutex.Unlock()
	if call > 1 {
		r.once.Do(func() { close(r.started) })
		<-r.release
	}
	return nil
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
	mutex    sync.Mutex
	messages []taskspec.NATSMessage
	err      error
}

func (p *_SchedulerTaskPublisher) PublishTask(message taskspec.NATSMessage) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.messages = append(p.messages, message)
	return p.err
}

func TestSchedulerRefreshSchedulesDeduplicatesAppInstances(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	target := newTestScheduler([]*core.AppStatus{
		newTestScheduledAppStatus("instance-1"),
		newTestScheduledAppStatus("instance-2"),
	}, publisher)

	require.NoError(t, target.refreshSchedules())

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

	require.NoError(t, target.refreshSchedules())

	assert.Len(t, target.jobs, 2)
}

func TestSchedulerRefreshSchedulesRemovesMissingRegistrations(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	registryRepo := &_SchedulerRegistryRepo{statuses: []*core.AppStatus{newTestScheduledAppStatus("instance-1")}}
	target := newTestSchedulerWithRegistry(registryRepo, publisher)

	require.NoError(t, target.refreshSchedules())
	registryRepo.statuses = []*core.AppStatus{}
	require.NoError(t, target.refreshSchedules())

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

func TestSchedulerPublishScheduleHandlesPublisherError(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{err: errors.New("nats unavailable")}
	target := newTestScheduler([]*core.AppStatus{newTestScheduledAppStatus("instance-1")}, publisher)

	assert.NotPanics(t, func() {
		target.publishSchedule(newTestScheduleConfig())
	})
	assert.Len(t, publisher.messages, 1)
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

	err := target.refreshSchedules()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have no arguments")
	assert.Empty(t, target.jobs)
}

func TestSchedulerRejectsInvalidCronWithoutChangingExistingJobs(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	registryRepo := &_SchedulerRegistryRepo{statuses: []*core.AppStatus{newTestScheduledAppStatus("instance-1")}}
	target := newTestSchedulerWithRegistry(registryRepo, publisher)
	require.NoError(t, target.refreshSchedules())
	require.Len(t, target.jobs, 1)

	registryRepo.statuses[0].TaskRunners[0].CronSchedulers[0].CronExpr = "not-a-cron"
	err := target.refreshSchedules()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cron expression")
	assert.Len(t, target.jobs, 1)
}

func TestSchedulerRefreshRecoveryContainsUnexpectedPanic(t *testing.T) {
	target := &Scheduler{
		RegistryRepo: &_PanickingSchedulerRegistryRepo{},
		SchemaRepo:   &_SchedulerSchemaRepo{},
		publisher:    &_SchedulerTaskPublisher{},
	}
	target.DIInit()

	assert.NotPanics(t, target.refreshSchedulesSafely)
}

func TestSchedulerCronRecoversJobPanic(t *testing.T) {
	target := newTestScheduler(nil, &_SchedulerTaskPublisher{})
	started := make(chan struct{})
	var once sync.Once
	_, err := target.cron.AddFunc("@every 1ms", func() {
		once.Do(func() { close(started) })
		panic("job failed")
	})
	require.NoError(t, err)
	target.cron.Start()
	requireSchedulerSignal(t, started, "cron job did not start")
	<-target.cron.Stop().Done()
}

func TestSchedulerBeforeAppStopWaitsForRefreshLoop(t *testing.T) {
	registryRepo := &_BlockingSchedulerRegistryRepo{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	target := &Scheduler{
		RegistryRepo:    registryRepo,
		SchemaRepo:      &_SchedulerSchemaRepo{},
		publisher:       &_SchedulerTaskPublisher{},
		refreshInterval: time.Millisecond,
	}
	target.DIInit()
	target.AfterAppStart()
	requireSchedulerSignal(t, registryRepo.started, "refresh loop did not start")

	stopped := make(chan struct{})
	go func() {
		target.BeforeAppStop()
		close(stopped)
	}()
	requireSchedulerBlocked(t, stopped, "scheduler stopped before refresh completed")
	close(registryRepo.release)
	requireSchedulerSignal(t, stopped, "scheduler did not stop after refresh completed")
}

func TestSchedulerBeforeAppStopWaitsForRunningCronJob(t *testing.T) {
	target := newTestScheduler(nil, &_SchedulerTaskPublisher{})
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	_, err := target.cron.AddFunc("@every 1ms", func() {
		once.Do(func() { close(started) })
		<-release
	})
	require.NoError(t, err)
	target.cron.Start()
	requireSchedulerSignal(t, started, "cron job did not start")

	stopped := make(chan struct{})
	go func() {
		target.BeforeAppStop()
		close(stopped)
	}()
	requireSchedulerBlocked(t, stopped, "scheduler stopped before cron job completed")
	close(release)
	requireSchedulerSignal(t, stopped, "scheduler did not stop after cron job completed")
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

func requireSchedulerSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}

func requireSchedulerBlocked(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
		t.Fatal(message)
	case <-time.After(20 * time.Millisecond):
	}
}
