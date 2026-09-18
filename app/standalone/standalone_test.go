package standalone

import (
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/internal/appcli"
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

// _AdminListenTestAppSpec is separate: a spec type admits one application per process.
type _AdminListenTestAppSpec struct {
	app.Application
}

func (*_AdminListenTestAppSpec) Name() string {
	return "standalone.adminlisten.test"
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
			NewWithOption[*_TestAppSpec](Option{HubDBSQLiteFile: "/tmp/hub.sqlite"}),
		)
	})
}

func TestNewBundledPanicsForBundleWithOption(t *testing.T) {
	assert.PanicsWithError(t, "bundled standalone app must not have option", func() {
		NewBundled(
			NewBundledWithOption(
				Option{HubDBSQLiteFile: "/tmp/hub.sqlite"},
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
		HubSeedDataFile:  "/tmp/option-hub.yaml",
		HubDBSQLiteFile:  "/tmp/option-hub.sqlite",
		HubDBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
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
	applyOption(flags, Option{HubSeedData: "{}"})
	flags.Normalize(true)
	assert.Equal(t, "{}", flags.SeedHubData)
	assert.True(t, flags.NoDB)
	assert.PanicsWithError(t, "bundled standalone app must not have option", func() {
		NewBundled(new(_App{option: Option{HubSeedData: "{}"}}))
	})
}

func TestInlineSeedConflictsWithFile(t *testing.T) {
	for _, fromCLI := range []bool{false, true} {
		flags := new(hubflag.Flag{})
		option := Option{HubSeedData: "{}"}
		if fromCLI {
			flags.SeedHubDataFile = "seed.yaml"
		} else {
			option.HubSeedDataFile = "seed.yaml"
		}
		applyOption(flags, option)
		assert.PanicsWithError(t, "SeedHubData and the seed data file are mutually exclusive", func() {
			flags.Normalize(true)
		})
	}
}

func TestApplySeedTemplateOptions(t *testing.T) {
	flags := new(hubflag.Flag)
	applyOption(flags, Option{HubSeedData: "{}", HubSeedSource: "source", HubSeedVarsFile: "vars.yaml"})
	assert.Equal(t, "source", flags.SeedHubSource)
	assert.Equal(t, "vars.yaml", flags.SeedHubVarsFile)
	assert.False(t, Option{HubSeedSourceFile: "source.yaml"}.isZero())
	assert.False(t, Option{HubSeedVarsFile: "vars.yaml"}.isZero())
}

func TestFlagsParseInProcessHubParameters(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{
		"/tmp/app",
		"--hub-no-db",
		"--hub-db-sqlite-file", "/tmp/hub.sqlite",
		"--hub-db-postgres-url", "postgres://demo",
		"--hub-seed-data-file", "/tmp/seed.yaml",
		"--hub-seed-source-file", "/tmp/source.yaml",
		"--hub-seed-vars-file", "/tmp/vars.yaml",
		"--hub-admin-listen", "127.0.0.1:7099",
	}

	flag := &hubflag.Flag{}
	appcli.Handle(flags(flag, Option{})...)

	assert.True(t, flag.NoDB)
	assert.Equal(t, "/tmp/hub.sqlite", flag.DBSQLiteFile)
	assert.Equal(t, "postgres://demo", flag.DBPostgresURL)
	assert.Equal(t, "/tmp/seed.yaml", flag.SeedHubDataFile)
	assert.Equal(t, "/tmp/source.yaml", flag.SeedHubSourceFile)
	assert.Equal(t, "/tmp/vars.yaml", flag.SeedHubVarsFile)
	assert.Equal(t, "127.0.0.1:7099", flag.AdminListen)
}

func TestAdminListenOptionOverridesFlag(t *testing.T) {
	flag := &hubflag.Flag{AdminListen: "127.0.0.1:7098"}
	applyOption(flag, Option{HubAdminListen: "127.0.0.1:7099"})

	assert.Equal(t, "127.0.0.1:7099", flag.AdminListen)
	assert.False(t, Option{HubAdminListen: "127.0.0.1:7099"}.isZero())
}

// An ignored flag keeps parsing so the rest of the command line still applies,
// but neither the flag nor its environment variable reaches the runtime.
func TestIgnoredFlagAndItsEnvironmentAreDropped(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{
		"/tmp/app",
		"--hub-admin-listen", "127.0.0.1:7099",
		"--hub-seed-data-file", "/tmp/seed.yaml",
	}
	t.Setenv(EnvHubAdminListen, "127.0.0.1:7098")

	flag := &hubflag.Flag{}
	appcli.Handle(flags(flag, Option{IgnoredFlags: []string{FlagHubAdminListen}})...)

	assert.Empty(t, flag.AdminListen)
	assert.Equal(t, "/tmp/seed.yaml", flag.SeedHubDataFile)
}

func TestUnknownIgnoredFlagPanics(t *testing.T) {
	assert.Panics(t, func() { flags(&hubflag.Flag{}, Option{IgnoredFlags: []string{"hub-unknown"}}) })
}

// A renamed flag answers to the new name alone: the declared name and the
// environment variable derived from it replace the declared variable.
func TestRenamedFlagUsesTheDerivedNameAndEnvironment(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	renamed := Option{RenamedFlags: map[string]string{FlagHubAdminListen: "admin-listen"}}

	os.Args = []string{"/tmp/app", "--admin-listen", "127.0.0.1:7099"}
	fromFlag := &hubflag.Flag{}
	appcli.Handle(flags(fromFlag, renamed)...)
	assert.Equal(t, "127.0.0.1:7099", fromFlag.AdminListen)

	os.Args = []string{"/tmp/app"}
	t.Setenv(EnvHubAdminListen, "127.0.0.1:7098")
	declaredEnv := &hubflag.Flag{}
	appcli.Handle(flags(declaredEnv, renamed)...)
	assert.Empty(t, declaredEnv.AdminListen)

	t.Setenv("VINE_ADMIN_LISTEN", "127.0.0.1:7097")
	derivedEnv := &hubflag.Flag{}
	appcli.Handle(flags(derivedEnv, renamed)...)
	assert.Equal(t, "127.0.0.1:7097", derivedEnv.AdminListen)
}

func TestRenamedFlagCannotBeIgnored(t *testing.T) {
	assert.Panics(t, func() {
		flags(&hubflag.Flag{}, Option{
			IgnoredFlags: []string{FlagHubAdminListen},
			RenamedFlags: map[string]string{FlagHubAdminListen: "admin-listen"},
		})
	})
}

func TestUnknownRenamedFlagPanics(t *testing.T) {
	assert.Panics(t, func() {
		flags(&hubflag.Flag{}, Option{RenamedFlags: map[string]string{"hub-unknown": "admin-listen"}})
	})
}

func TestStandaloneServesDashboardOnAdminListen(t *testing.T) {
	address := freeListenAddress(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, os.WriteFile(seedPath, []byte("{}\n"), 0600))

	application := NewWithOption[*_AdminListenTestAppSpec](Option{
		HubSeedDataFile: seedPath,
		HubAdminListen:  address,
	})
	application.Start()
	t.Cleanup(application.StopGracefully)

	response, err := http.Get("http://" + address + "/")
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	// Source-only and packaged binaries expose the same admin listener.
	// The admin package separately pins resource detection for both layouts.
	if response.StatusCode == http.StatusNotFound {
		require.Contains(t, string(body), "script/dev-hub-dashboard.sh")
	} else {
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Contains(t, string(body), `<div id="app"></div>`)
	}
}

func freeListenAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return address
}
