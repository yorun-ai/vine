package app

import (
	"context"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ctr"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/skel"
	taskcore "go.yorun.ai/vine/internal/core/task"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
)

type testTaskerSpec struct {
	Application
	TaskerEnabled
}

func (*testTaskerSpec) Name() string {
	return "test.tasker"
}

func TestNewTaskerStoresAppAndSpec(t *testing.T) {
	ensureTaskerTaskRegistered()

	app := newTestAppImpl()
	spec := &testTaskerRunnerSpec{}

	tasker := newTasker(spec, app.info, app.bindAppDeps)

	assert.Equal(t, app.info, tasker.appInfo)
	assert.Same(t, spec, tasker.spec)
}

type testTaskerFilter struct{}

func (*testTaskerFilter) Filter(next ctr.FilterNext) {
	next()
}

type testTaskerRunnerLauncher interface {
	mustBeTestTaskerRunnerLauncher()
}

type defaultTestTaskerRunnerLauncher struct{}

func (*defaultTestTaskerRunnerLauncher) mustBeTestTaskerRunnerLauncher() {}

type testTaskerRunner interface {
	ForGroup(groupId int)
	EveryHour()
	mustBeTestTaskerRunner()
}

type defaultTestTaskerRunner struct{}

func (*defaultTestTaskerRunner) ForGroup(groupId int)    {}
func (*defaultTestTaskerRunner) EveryHour()              {}
func (*defaultTestTaskerRunner) mustBeTestTaskerRunner() {}

type testTaskerRunnerER interface {
	ForGroup(groupId int) ex.Error
	EveryHour() ex.Error
	mustBeTestTaskerRunnerER()
}

type _WrapperTestTaskerRunnerER struct {
	defaultTestTaskerRunner
	runnerImpl testTaskerRunner
}

func newWrapperTestTaskerRunnerER(runnerImpl testTaskerRunner) testTaskerRunnerER {
	return &_WrapperTestTaskerRunnerER{runnerImpl: runnerImpl}
}

func (r *_WrapperTestTaskerRunnerER) runner() testTaskerRunner {
	if r.runnerImpl == nil {
		return &r.defaultTestTaskerRunner
	}
	return r.runnerImpl
}

func (r *_WrapperTestTaskerRunnerER) ForGroup(groupId int) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	r.runner().ForGroup(groupId)
	return
}

func (r *_WrapperTestTaskerRunnerER) EveryHour() (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	r.runner().EveryHour()
	return
}

func (*_WrapperTestTaskerRunnerER) mustBeTestTaskerRunnerER() {}

type defaultTestTaskerRunnerER struct {
	_WrapperTestTaskerRunnerER
}

type testTaskerRunnerImpl struct {
	defaultTestTaskerRunner
}

var testTaskerRunGroupID int

func (*testTaskerRunnerImpl) ForGroup(groupId int) {
	testTaskerRunGroupID = groupId
}

type testTaskerRunArguments struct {
	GroupId int `json:"groupId"`
}

func ensureTaskerTaskRegistered() {
	RegisterTaskerTaskOnce()
}

var registerTaskerTaskOnce = func() func() {
	var done bool
	return func() {
		if done {
			return
		}
		done = true
		taskspec.Register(&taskspec.TaskSpec{
			Name:                "TaskerTestTask",
			SkelName:            "test.tasker.TaskerTestTask",
			RunnerType:          reflect.TypeOf((*testTaskerRunner)(nil)).Elem(),
			DefaultRunnerType:   reflect.TypeOf(&defaultTestTaskerRunner{}),
			ERRunnerType:        reflect.TypeOf((*testTaskerRunnerER)(nil)).Elem(),
			WrapperERRunnerCtor: newWrapperTestTaskerRunnerER,
			DefaultERRunnerType: reflect.TypeOf(&defaultTestTaskerRunnerER{}),
			LauncherType:        reflect.TypeOf((*testTaskerRunnerLauncher)(nil)).Elem(),
			LauncherCtor:        func(*taskcore.Launcher) testTaskerRunnerLauncher { return &defaultTestTaskerRunnerLauncher{} },
			Triggers: []*taskspec.TriggerSpec{{
				Name:             "ForGroup",
				SkelName:         "forGroup",
				RunnerMethodName: "ForGroup",
				ArgumentsType:    reflect.TypeOf(testTaskerRunArguments{}),
			}, {
				Name:             "EveryHour",
				SkelName:         "everyHour",
				RunnerMethodName: "EveryHour",
			}},
		})
	}
}()

func RegisterTaskerTaskOnce() {
	registerTaskerTaskOnce()
}

type testTaskerRunnerSpec struct {
	Application
	TaskerEnabled
}

func (*testTaskerRunnerSpec) Name() string {
	return "test.tasker.runner"
}

func (*testTaskerRunnerSpec) TaskerBind(*di.Binder) {}

func (*testTaskerRunnerSpec) TaskerInitRunners(addRunner RunnerTypeAdder) {
	addRunner(
		T[*testTaskerRunnerImpl](),
		WithRunnerTimeout(time.Second),
		WithRunnerConcurrency(2),
		WithRunnerNoRetry(),
	)
}

func (*testTaskerRunnerSpec) TaskerInitFilters(addFilter TypeAdder) {
	addFilter(T[*testTaskerFilter]())
}

func TestNewTaskerInitBuildsServers(t *testing.T) {
	ensureTaskerTaskRegistered()

	app := newTestAppImpl()
	spec := &testTaskerRunnerSpec{}

	tasker := newTasker(spec, app.info, app.bindAppDeps)

	assert.NotNil(t, tasker.taskServer)
	assert.NotNil(t, tasker.rpcServer)
	assert.Equal(t, []reflect.Type{T[*testTaskerRunnerImpl]()}, tasker.runnerTypes())
	assert.Equal(t, []reflect.Type{T[*testTaskerFilter]()}, tasker.filterTypes())
	assert.Len(t, tasker.runners, 1)
	assert.Equal(t, time.Second, tasker.runners[0].options.Timeout)
	assert.Equal(t, 2, tasker.runners[0].options.Concurrency)
	assert.True(t, tasker.runners[0].options.NoRetry)
}

func TestNewTaskerUsesDefaultRunnerOptions(t *testing.T) {
	ensureTaskerTaskRegistered()

	app := newTestAppImpl()
	spec := &testFullSpec{}

	tasker := newTasker(spec, app.info, app.bindAppDeps)

	assert.Len(t, tasker.runners, 1)
	assert.Equal(t, 30*time.Second, tasker.runners[0].options.Timeout)
	assert.Equal(t, 10, tasker.runners[0].options.Concurrency)
	assert.False(t, tasker.runners[0].options.NoRetry)
}

type testTaskerRunnerCronSchedulerSpec struct {
	Application
	TaskerEnabled
}

func (*testTaskerRunnerCronSchedulerSpec) Name() string {
	return "test.tasker.runner.scheduler"
}

func (*testTaskerRunnerCronSchedulerSpec) TaskerInitRunners(addRunner RunnerTypeAdder) {
	addRunner(
		T[*testTaskerRunnerImpl](),
		WithRunnerCronScheduler("everyHour", "0 * * * *"),
		WithRunnerCronScheduler("everyHour", "30 * * * *"),
	)
}

func TestTaskerTaskRunnerRegistrationsIncludesCronScheduler(t *testing.T) {
	ensureTaskerTaskRegistered()

	app := newTestAppImpl()
	tasker := newTasker(&testTaskerRunnerCronSchedulerSpec{}, app.info, app.bindAppDeps)

	registrations := tasker.taskRunnerRegistrations()

	require.Len(t, registrations, 1)
	require.Len(t, registrations[0].CronSchedulers, 2)
	assert.Equal(t, "everyHour", registrations[0].CronSchedulers[0].TriggerSkelName)
	assert.Equal(t, "0 * * * *", registrations[0].CronSchedulers[0].CronExpr)
	assert.Equal(t, "everyHour", registrations[0].CronSchedulers[1].TriggerSkelName)
	assert.Equal(t, "30 * * * *", registrations[0].CronSchedulers[1].CronExpr)
}

type testTaskerRunnerArgumentCronSchedulerSpec struct {
	Application
	TaskerEnabled
}

func (*testTaskerRunnerArgumentCronSchedulerSpec) Name() string {
	return "test.tasker.runner.argument.scheduler"
}

func (*testTaskerRunnerArgumentCronSchedulerSpec) TaskerInitRunners(addRunner RunnerTypeAdder) {
	addRunner(
		T[*testTaskerRunnerImpl](),
		WithRunnerCronScheduler("forGroup", "0 * * * *"),
	)
}

func TestTaskerTaskRunnerRegistrationsPanicsWhenCronSchedulerTriggerHasArguments(t *testing.T) {
	ensureTaskerTaskRegistered()

	app := newTestAppImpl()
	tasker := newTasker(&testTaskerRunnerArgumentCronSchedulerSpec{}, app.info, app.bindAppDeps)

	assert.Panics(t, func() {
		tasker.taskRunnerRegistrations()
	})
}

func TestWithRunnerCronSchedulerPanicsWhenCronExprInvalid(t *testing.T) {
	assert.Panics(t, func() {
		WithRunnerCronScheduler("everyHour", "bad cron")
	})
}

func TestAppTaskServiceServerRunTaskForwardsToTaskServer(t *testing.T) {
	ensureTaskerTaskRegistered()
	testTaskerRunGroupID = 0

	app := newTestAppImpl()
	tasker := newTasker(&testTaskerRunnerSpec{}, app.info, app.bindAppDeps)
	actor := meta.NewAbsentActor()
	service := &_AppTaskServiceServerImpl{
		Context:    rpcspec.NewContext(context.Background(), meta.InitialTrace(), nil, nil, actor),
		TaskServer: tasker.taskServer,
	}

	service.RunTask(appskeled.TaskRun{
		Metadata: appskeled.TaskRunMeta{
			TraceId:       meta.InitialTrace().Id(),
			TraceSpan:     meta.InitialTrace().Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		TaskSkelName:    "test.tasker.TaskerTestTask",
		TriggerSkelName: "forGroup",
		ArgumentsJson:   `{"groupId":7}`,
	})

	assert.Equal(t, 7, testTaskerRunGroupID)
}
