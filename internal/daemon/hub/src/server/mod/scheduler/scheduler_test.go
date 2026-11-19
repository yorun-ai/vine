package scheduler

import (
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/meta"
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

type _SchedulerDescriptorRepo struct {
	core.DescriptorRepo

	taskVersions []core.DescriptorVersion[*skeldesc.Task]
}

func (r *_SchedulerDescriptorRepo) ListTaskDescriptorVersions() []core.DescriptorVersion[*skeldesc.Task] {
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

func TestSchedulerPublishesOnceAcrossRollingUpgrade(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	old := newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	next := newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000002"))
	next.TaskRunners[0].DescriptorHash = "new-task-hash"
	registry := &_SchedulerRegistryRepo{statuses: []*core.AppStatus{old}}
	target := newTestSchedulerWithRegistry(registry, publisher)
	descriptors := target.DescriptorRepo.(*_SchedulerDescriptorRepo)
	descriptors.taskVersions = append(descriptors.taskVersions, core.DescriptorVersion[*skeldesc.Task]{
		DescriptorHash: next.TaskRunners[0].DescriptorHash,
		Descriptor: &skeldesc.Task{
			SkelName: next.TaskRunners[0].TaskSkelName,
			Triggers: []*skeldesc.TaskTrigger{{SkelName: "rebuild"}, {SkelName: "preview"}},
		},
	})
	assertTick := func(want int) {
		t.Helper()
		before := len(publisher.messages)
		for _, entry := range target.cron.Entries() {
			entry.Job.Run()
		}
		assert.Equal(t, want, len(publisher.messages)-before)
	}
	require.NoError(t, target.refreshSchedules())
	assertTick(1)

	registry.statuses = []*core.AppStatus{old, next}
	require.NoError(t, target.refreshSchedules())
	assertTick(1)

	// The existing timer can fire between unregister and the next refresh.
	registry.statuses = []*core.AppStatus{next}
	assertTick(1)
	registry.statuses = nil
	assertTick(0)
}

func TestSchedulerRefreshSchedulesAddsMultipleCronSchedulers(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	status := newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
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
	registryRepo := &_SchedulerRegistryRepo{statuses: []*core.AppStatus{newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000001"))}}
	target := newTestSchedulerWithRegistry(registryRepo, publisher)

	require.NoError(t, target.refreshSchedules())
	registryRepo.statuses = []*core.AppStatus{}
	require.NoError(t, target.refreshSchedules())

	assert.Empty(t, target.jobs)
}

func TestSchedulerPublishSchedulePublishesTaskMessage(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	target := newTestScheduler([]*core.AppStatus{newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000001"))}, publisher)

	target.publishSchedule(newTestScheduleConfig())

	require.Len(t, publisher.messages, 1)
	assert.Equal(t, target.CurrentApp.Name(), publisher.messages[0].Metadata.AppName)
	assert.Equal(t, target.CurrentApp.Version(), publisher.messages[0].Metadata.AppVersion)
	assert.Equal(t, target.CurrentApp.InstanceId(), publisher.messages[0].Metadata.AppInstanceId)
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
	target := newTestScheduler([]*core.AppStatus{newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000001"))}, publisher)

	assert.NotPanics(t, func() {
		target.publishSchedule(newTestScheduleConfig())
	})
	assert.Len(t, publisher.messages, 1)
}

func TestSchedulerRejectsCronSchedulerTriggerWithArguments(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	old := newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	next := newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000002"))
	next.TaskRunners[0].DescriptorHash = "incompatible-task-hash"
	registry := &_SchedulerRegistryRepo{statuses: []*core.AppStatus{old}}
	target := newTestSchedulerWithRegistry(registry, publisher)
	require.NoError(t, target.refreshSchedules())
	descriptors := target.DescriptorRepo.(*_SchedulerDescriptorRepo)
	descriptors.taskVersions = append(descriptors.taskVersions, core.DescriptorVersion[*skeldesc.Task]{
		DescriptorHash: next.TaskRunners[0].DescriptorHash,
		Descriptor: &skeldesc.Task{
			SkelName: next.TaskRunners[0].TaskSkelName,
			Triggers: []*skeldesc.TaskTrigger{{
				SkelName: "rebuild",
				Arguments: []*skeldesc.Member{{
					Name: "full",
					Type: &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarBoolean},
				}},
			}},
		},
	})

	registry.statuses = []*core.AppStatus{old, next}
	require.NoError(t, target.refreshSchedules())
	entries := target.cron.Entries()
	require.Len(t, entries, 1)
	registry.statuses = []*core.AppStatus{next, old}
	entries[0].Job.Run()
	require.Len(t, publisher.messages, 1)
	registry.statuses = []*core.AppStatus{next}
	entries[0].Job.Run()
	assert.Len(t, publisher.messages, 1)
}

func TestSchedulerReconcilesHealthySchedulesAlongsideInvalidRegistrations(t *testing.T) {
	publisher := &_SchedulerTaskPublisher{}
	stale := newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	registryRepo := &_SchedulerRegistryRepo{statuses: []*core.AppStatus{stale}}
	target := newTestSchedulerWithRegistry(registryRepo, publisher)
	require.NoError(t, target.refreshSchedules())
	previous := target.cron.Entries()[0]

	stale.TaskRunners[0].DescriptorHash = ""
	healthy := newTestScheduledAppStatus(uuid.MustParse("00000000-0000-0000-0000-000000000002"))
	healthy.Name = "another.app"
	runner := &healthy.TaskRunners[0]
	runner.CronSchedulers = append(runner.CronSchedulers,
		core.TaskRunnerCronScheduler{TriggerSkelName: "rebuild", CronExpr: "not-a-cron"},
		core.TaskRunnerCronScheduler{TriggerSkelName: "rebuild", CronExpr: "TZ=UTC"},
	)
	registryRepo.statuses = append(registryRepo.statuses, healthy)
	require.NoError(t, target.refreshSchedules())
	entries := target.cron.Entries()
	require.Len(t, entries, 1)
	require.NotEqual(t, previous.ID, entries[0].ID)
	previous.Job.Run()
	require.Empty(t, publisher.messages)
	entries[0].Job.Run()
	require.Len(t, publisher.messages, 1)

	registryRepo.statuses = []*core.AppStatus{stale}
	require.NoError(t, target.refreshSchedules())
	require.Empty(t, target.cron.Entries())
}

func TestSchedulerRefreshRecoveryContainsUnexpectedPanic(t *testing.T) {
	target := &Scheduler{
		RegistryRepo:   &_PanickingSchedulerRegistryRepo{},
		DescriptorRepo: &_SchedulerDescriptorRepo{},
		publisher:      &_SchedulerTaskPublisher{},
	}
	target.DIInit()

	assert.NotPanics(t, target.refreshSchedulesSafely)
}

func TestSchedulerCronRecoversJobPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		target := newTestScheduler(nil, &_SchedulerTaskPublisher{})
		started := make(chan struct{})
		var once sync.Once
		_, err := target.cron.AddFunc("@every 1ms", func() {
			once.Do(func() { close(started) })
			panic("job failed")
		})
		require.NoError(t, err)
		target.cron.Start()
		synctest.Sleep(time.Second)
		requireSignal(t, started, "cron job did not start")
		<-target.cron.Stop().Done()
	})
}

func TestSchedulerBeforeAppStopWaitsForRefreshLoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registryRepo := &_BlockingSchedulerRegistryRepo{
			started: make(chan struct{}),
			release: make(chan struct{}),
		}
		target := &Scheduler{
			RegistryRepo:    registryRepo,
			DescriptorRepo:  &_SchedulerDescriptorRepo{},
			publisher:       &_SchedulerTaskPublisher{},
			refreshInterval: time.Millisecond,
		}
		target.DIInit()
		target.AfterAppStart()
		synctest.Sleep(time.Millisecond)
		requireSignal(t, registryRepo.started, "refresh loop did not start")

		stopped := make(chan struct{})
		go func() {
			target.BeforeAppStop()
			close(stopped)
		}()
		synctest.Wait()
		requireNoSignal(t, stopped, "scheduler stopped before refresh completed")
		close(registryRepo.release)
		synctest.Wait()
		requireSignal(t, stopped, "scheduler did not stop after refresh completed")
	})
}

func TestSchedulerBeforeAppStopWaitsForRunningCronJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
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
		synctest.Sleep(time.Second)
		requireSignal(t, started, "cron job did not start")

		stopped := make(chan struct{})
		go func() {
			target.BeforeAppStop()
			close(stopped)
		}()
		synctest.Wait()
		requireNoSignal(t, stopped, "scheduler stopped before cron job completed")
		close(release)
		synctest.Wait()
		requireSignal(t, stopped, "scheduler did not stop after cron job completed")
	})
}

func newTestScheduler(statuses []*core.AppStatus, publisher *_SchedulerTaskPublisher) *Scheduler {
	return newTestSchedulerWithRegistry(&_SchedulerRegistryRepo{statuses: statuses}, publisher)
}

func newTestSchedulerWithRegistry(registryRepo *_SchedulerRegistryRepo, publisher *_SchedulerTaskPublisher) *Scheduler {
	target := &Scheduler{
		CurrentApp:   meta.MustNewApp("vine.hub", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174099")),
		RegistryRepo: registryRepo,
		DescriptorRepo: &_SchedulerDescriptorRepo{
			taskVersions: []core.DescriptorVersion[*skeldesc.Task]{{
				DescriptorHash: "task-hash",
				Descriptor: &skeldesc.Task{
					SkelName: "demo.booker.RebuildCatalogIndexTask",
					Triggers: []*skeldesc.TaskTrigger{{
						SkelName: "rebuild",
					}},
				},
			}},
		},
		publisher: publisher,
	}
	target.DIInit()
	return target
}

func newTestScheduledAppStatus(instanceId uuid.UUID) *core.AppStatus {
	return &core.AppStatus{
		Name:       "booker",
		InstanceId: instanceId,
		TaskRunners: []core.TaskRunnerRegistration{{
			TaskSkelName:   "demo.booker.RebuildCatalogIndexTask",
			DescriptorHash: "task-hash",
			CronSchedulers: []core.TaskRunnerCronScheduler{{
				TriggerSkelName: "rebuild",
				CronExpr:        "0 * * * *",
			}},
		}},
	}
}

func newTestScheduleConfig() _ScheduleConfig {
	return _ScheduleConfig{
		AppName:      "booker",
		TaskSkelName: "demo.booker.RebuildCatalogIndexTask",
		TaskRunnerCronScheduler: core.TaskRunnerCronScheduler{
			TriggerSkelName: "rebuild",
			CronExpr:        "0 * * * *",
		},
	}
}

func requireSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	default:
		t.Fatal(message)
	}
}

func requireNoSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
		t.Fatal(message)
	default:
	}
}
