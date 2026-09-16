package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/app"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

type _TestAppSpec struct {
	app.Application
}

func (*_TestAppSpec) Name() string {
	return "standalone.test"
}

type _BundledTestAppSpec struct {
	app.Application
}

func (*_BundledTestAppSpec) Name() string {
	return "standalone.bundled.test"
}

func TestNewBundledPanicsForNonStandaloneApp(t *testing.T) {
	assert.PanicsWithError(t, "standalone app expected", func() {
		NewBundled(_NonStandaloneApp{})
	})
}

func TestNewBundledPanicsForEmptyApps(t *testing.T) {
	assert.PanicsWithError(t, "standalone app expected", func() {
		NewBundled()
	})
}

func TestNewBundledPanicsForAppWithOption(t *testing.T) {
	assert.PanicsWithError(t, "bundled standalone app must not have option", func() {
		NewBundled(
			NewWithOption[*_TestAppSpec](Option{SQLiteFile: "/tmp/hub.sqlite"}),
		)
	})
}

func TestNewBundledPanicsForBundleWithOption(t *testing.T) {
	assert.PanicsWithError(t, "bundled standalone app must not have option", func() {
		NewBundled(
			NewBundledWithOption(
				Option{SQLiteFile: "/tmp/hub.sqlite"},
				New[*_BundledTestAppSpec](),
			),
		)
	})
}

func TestNewBundledFlattensBundles(t *testing.T) {
	appA := _RecordingApp{name: "app.a"}
	appB := _RecordingApp{name: "app.b"}
	appC := _RecordingApp{name: "app.c"}

	innerBundle := NewBundled(
		&_App{apps: []app.App{appA, appB}},
	)
	outerBundle := NewBundled(
		innerBundle,
		&_App{apps: []app.App{appC}},
	).(*_App)

	assert.Equal(t, []app.App{appA, appB, appC}, outerBundle.apps)
}

func TestStopGracefullyWaitsAppsBeforeStoppingInfra(t *testing.T) {
	events := []string{}
	appA := &_RecordingApp{name: "app.a", events: &events}
	appB := &_RecordingApp{name: "app.b", events: &events}
	standalone := &_App{
		apps:   []app.App{appA, appB},
		link:   &_RecordingApp{name: "link", events: &events},
		portal: &_RecordingApp{name: "portal", events: &events},
		hub:    &_RecordingApp{name: "hub", events: &events},
	}

	standalone.StopGracefully()

	assert.Equal(t, []string{
		"app.b.stop", "app.b.wait",
		"app.a.stop", "app.a.wait",
		"link.stop", "link.wait",
		"portal.stop", "portal.wait",
		"hub.stop", "hub.wait",
	}, events)
}

func TestApplyOptionOverridesFlag(t *testing.T) {
	flag := &hubflag.Flag{
		SeedHubDataFile: "/tmp/cli-hub.yaml",
		DBSQLiteFile:    "/tmp/cli-hub.sqlite",
		DBPostgresURL:   "postgres://cli",
	}

	applyOption(flag, Option{
		SeedHubDataFile: "/tmp/option-hub.yaml",
		SQLiteFile:      "/tmp/option-hub.sqlite",
		PostgresURL:     "postgres://demo:demo@127.0.0.1:5432/hub",
	})

	assert.Equal(t, "/tmp/option-hub.yaml", flag.SeedHubDataFile)
	assert.Equal(t, "/tmp/option-hub.sqlite", flag.DBSQLiteFile)
	assert.Equal(t, "postgres://demo:demo@127.0.0.1:5432/hub", flag.DBPostgresURL)
}

func TestApplyOptionKeepsUnsetFlagValues(t *testing.T) {
	flag := &hubflag.Flag{
		SeedHubDataFile: "/tmp/cli-hub.yaml",
		DBSQLiteFile:    "/tmp/cli-hub.sqlite",
		DBPostgresURL:   "postgres://cli",
	}

	applyOption(flag, Option{})

	assert.Equal(t, "/tmp/cli-hub.yaml", flag.SeedHubDataFile)
	assert.Equal(t, "/tmp/cli-hub.sqlite", flag.DBSQLiteFile)
	assert.Equal(t, "postgres://cli", flag.DBPostgresURL)
}

type _NonStandaloneApp struct{}

func (_NonStandaloneApp) Name() string    { return "non.standalone" }
func (_NonStandaloneApp) Start()          {}
func (_NonStandaloneApp) StopGracefully() {}
func (_NonStandaloneApp) StartAndWait()   {}

type _RecordingApp struct {
	name   string
	events *[]string
}

func (a _RecordingApp) Name() string { return a.name }
func (_RecordingApp) Start()         {}

func (a _RecordingApp) StopGracefully() {
	*a.events = append(*a.events, a.name+".stop", a.name+".wait")
}

func (_RecordingApp) StartAndWait() {}

func TestInlineSeedOption(t *testing.T) {
	flags := new(hubflag.Flag{})
	applyOption(flags, Option{SeedHubData: "{}"})
	flags.Normalize(true)
	assert.Equal(t, "{}", flags.SeedHubData)
	assert.True(t, flags.NoDB)
	assert.PanicsWithError(t, "bundled standalone app must not have option", func() {
		NewBundled(new(_App{option: Option{SeedHubData: "{}"}}))
	})
}

func TestInlineSeedConflictsWithFile(t *testing.T) {
	for _, fromCLI := range []bool{false, true} {
		flags := new(hubflag.Flag{})
		option := Option{SeedHubData: "{}"}
		if fromCLI {
			flags.SeedHubDataFile = "seed.yaml"
		} else {
			option.SeedHubDataFile = "seed.yaml"
		}
		applyOption(flags, option)
		assert.PanicsWithError(t, "SeedHubData and seed-data-file are mutually exclusive", func() {
			flags.Normalize(true)
		})
	}
}

func TestApplySeedTemplateOptions(t *testing.T) {
	flags := new(hubflag.Flag)
	applyOption(flags, Option{SeedHubData: "{}", SeedHubSource: "source", SeedHubVarsFile: "vars.yaml"})
	assert.Equal(t, "source", flags.SeedHubSource)
	assert.Equal(t, "vars.yaml", flags.SeedHubVarsFile)
	assert.False(t, Option{SeedHubSourceFile: "source.yaml"}.isZero())
	assert.False(t, Option{SeedHubVarsFile: "vars.yaml"}.isZero())
}
