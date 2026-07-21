package ctr

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/internal/core/di"
)

type ctrTestTarget struct {
	di.SingletonScoped
}

func (t *ctrTestTarget) Combine(count int, name string) string {
	return name + ":" + string(rune('0'+count))
}

func (t *ctrTestTarget) Sum(left int, right int) int {
	return left + right
}

type ctrPanicTarget struct {
	di.SingletonScoped
}

func (t *ctrPanicTarget) Boom() {
	panic("boom")
}

type ctrTestRecorder struct {
	events *[]string
}

func (r *ctrTestRecorder) Record(event string) {
	*r.events = append(*r.events, event)
}

type ctrTraceFilter struct {
	di.ExecutionScoped

	Recorder *ctrTestRecorder `inject:""`
}

func (f *ctrTraceFilter) Filter(next FilterNext) {
	f.Recorder.Record("trace:before")
	next()
	f.Recorder.Record("trace:after")
}

type ctrContextFilter struct {
	di.ExecutionScoped

	Context  *Context         `inject:""`
	Recorder *ctrTestRecorder `inject:""`
}

func (f *ctrContextFilter) Filter(next FilterNext) {
	f.Recorder.Record("context:" + f.Context.TargetMethodName())
	next()
}

type ctrSeededMessage struct {
	di.ExecutionScoped

	Value string
}

type ctrSeededFilter struct {
	di.ExecutionScoped

	Message  *ctrSeededMessage `inject:""`
	Recorder *ctrTestRecorder  `inject:""`
}

func (f *ctrSeededFilter) Filter(next FilterNext) {
	f.Recorder.Record("seed:" + f.Message.Value)
	next()
}

type ctrDisposeFilter struct {
	di.ExecutionScoped

	Recorder *ctrTestRecorder `inject:""`
}

func (f *ctrDisposeFilter) DIDispose() {
	f.Recorder.Record("dispose:filter")
}

func (f *ctrDisposeFilter) Filter(next FilterNext) {
	f.Recorder.Record("dispose:before")
	next()
}

type ctrFallbackExecutionDep struct {
	Recorder *ctrTestRecorder `inject:""`
}

func (d *ctrFallbackExecutionDep) DIDispose() {
	d.Recorder.Record("dispose:fallback")
}

type ctrFallbackExecutionFilter struct {
	di.ExecutionScoped

	Dependency *ctrFallbackExecutionDep `inject:""`
	Recorder   *ctrTestRecorder         `inject:""`
}

func (f *ctrFallbackExecutionFilter) Filter(next FilterNext) {
	f.Recorder.Record("fallback:before")
	next()
	f.Recorder.Record("fallback:after")
}

func newCTRTestContainer(events *[]string, filterTypes []reflect.Type, bindAppliers ...di.BindApplier) Container {
	baseBindAppliers := []di.BindApplier{
		func(b *di.Binder) {
			b.BindInstance(&ctrTestRecorder{events: events})
		},
	}
	baseBindAppliers = append(baseBindAppliers, bindAppliers...)

	return NewContainer(Option{
		BindAppliers: baseBindAppliers,
		FilterTypes:  filterTypes,
	})
}

func TestNewContainerRegistersFiltersAndContext(t *testing.T) {
	events := []string{}
	container := newCTRTestContainer(
		&events,
		[]reflect.Type{reflect.TypeFor[*ctrTraceFilter]()},
		func(b *di.Binder) {
			b.Bind(di.T[*ctrTestTarget]())
		},
	)

	execution := container.NewExecution(reflect.TypeOf(&ctrTestTarget{}), getMethodByName(reflect.TypeOf(&ctrTestTarget{}), "Sum"))
	execution.Execute([]any{1, 2})

	assert.Equal(t, []string{"trace:before", "trace:after"}, events)
	assert.Equal(t, []any{3}, execution.Results())
}

func TestNewContainerPanicsWhenFilterTypeIsInvalid(t *testing.T) {
	assert.PanicsWithError(t, "filter type int must be pointer to struct", func() {
		NewContainer(Option{
			FilterTypes: []reflect.Type{reflect.TypeFor[int]()},
		})
	})
}

func TestNewContainerResolvesUnsetBindingsWithExecutionFallback(t *testing.T) {
	events := []string{}
	container := newCTRTestContainer(
		&events,
		[]reflect.Type{reflect.TypeFor[*ctrFallbackExecutionFilter]()},
		func(b *di.Binder) {
			b.Bind(di.T[*ctrFallbackExecutionDep]())
			b.Bind(di.T[*ctrTestTarget]())
		},
	)

	execution := container.NewExecution(reflect.TypeOf(&ctrTestTarget{}), getMethodByName(reflect.TypeOf(&ctrTestTarget{}), "Sum"))
	execution.Execute([]any{3, 4})

	assert.Equal(t, []string{"fallback:before", "fallback:after", "dispose:fallback"}, events)
	assert.Equal(t, []any{7}, execution.Results())
}
