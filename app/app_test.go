package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type _BundledTestAppSpec struct {
	Application
}

func (*_BundledTestAppSpec) Name() string {
	return "app.bundled.test"
}

type _RecordingApp struct {
	name   string
	events *[]string
}

func (a *_RecordingApp) Name() string {
	return a.name
}

func (a *_RecordingApp) Start() {
	*a.events = append(*a.events, a.name+".start")
}

func (a *_RecordingApp) StopGracefully() {
	*a.events = append(*a.events, a.name+".stop")
}

func (*_RecordingApp) StartAndWait() {}

func TestNewPreservesApplicationName(t *testing.T) {
	assert.Equal(t, "app.bundled.test", New[*_BundledTestAppSpec]().Name())
}

func TestNewBundledRequiresApps(t *testing.T) {
	assert.PanicsWithError(t, "app.New app expected", func() {
		NewBundled()
	})
}

func TestNewBundledRequiresPlainApps(t *testing.T) {
	assert.PanicsWithError(t, "app.New app expected", func() {
		NewBundled(&_RecordingApp{})
	})
}

func TestNewBundledFlattensBundles(t *testing.T) {
	appA := _RecordingApp{name: "app.a"}
	appB := _RecordingApp{name: "app.b"}
	appC := _RecordingApp{name: "app.c"}

	innerBundle := NewBundled(
		&_App{apps: []App{&appA, &appB}},
	)
	outerBundle := NewBundled(
		innerBundle,
		&_App{apps: []App{&appC}},
	).(*_App)

	assert.Equal(t, []App{&appA, &appB, &appC}, outerBundle.apps)
	assert.Empty(t, outerBundle.Name())
}

func TestNewBundledStartsInOrderAndStopsInReverse(t *testing.T) {
	events := []string{}
	appA := &_RecordingApp{name: "app.a", events: &events}
	appB := &_RecordingApp{name: "app.b", events: &events}
	bundle := NewBundled(
		&_App{apps: []App{appA}},
		&_App{apps: []App{appB}},
	)

	bundle.Start()
	bundle.StopGracefully()

	assert.Equal(t, []string{
		"app.a.start",
		"app.b.start",
		"app.b.stop",
		"app.a.stop",
	}, events)
}

func TestApplyOptionOverridesCliOption(t *testing.T) {
	cliOption := &Option{
		LinkEndpoint: "http://cli-link.local:7079",
	}

	applyOption(cliOption, Option{
		LinkEndpoint: "http://option-link.local:7079",
	})

	assert.Equal(t, "http://option-link.local:7079", cliOption.LinkEndpoint)
}

func TestApplyOptionKeepsUnsetCliOption(t *testing.T) {
	cliOption := &Option{
		LinkEndpoint: "http://cli-link.local:7079",
	}

	applyOption(cliOption, Option{})

	assert.Equal(t, "http://cli-link.local:7079", cliOption.LinkEndpoint)
}
