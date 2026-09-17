package linked

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/internal/appcli"
	"go.yorun.ai/vine/internal/core/mtls"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func TestStopGracefullyWaitsAppBeforeStoppingLink(t *testing.T) {
	events := []string{}
	application := &_App{
		apps: []app.App{&_RecordingApp{name: "app", events: &events}},
		link: &_RecordingApp{name: "link", events: &events},
	}

	application.StopGracefully()

	assert.Equal(t, []string{"app.stop", "app.wait", "link.stop", "link.wait"}, events)
}

func TestNewBundledRequiresLinkedApps(t *testing.T) {
	assert.PanicsWithError(t, "linked app expected", func() {
		NewBundled(&_RecordingApp{name: "app"})
	})
}

func TestApplyOptionOverridesFlag(t *testing.T) {
	flag := &linkflag.Flag{
		HubEndpoint:   "http://cli-hub.local:7071",
		IngressListen: "127.0.0.1:8080",
		MTLS: mtls.Files{
			CAFile:   "/tmp/cli-ca.pem",
			CertFile: "/tmp/cli-link.pem",
			KeyFile:  "/tmp/cli-link-key.pem",
		},
	}

	applyOption(flag, Option{
		LinkHubEndpoint:   "http://option-hub.local:7071",
		LinkIngressListen: "127.0.0.1:9090",
		LinkMTLSCAFile:    "/tmp/option-ca.pem",
		LinkMTLSCertFile:  "/tmp/option-link.pem",
		LinkMTLSKeyFile:   "/tmp/option-link-key.pem",
	})

	assert.Equal(t, "http://option-hub.local:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:9090", flag.IngressListen)
	assert.Equal(t, "/tmp/option-ca.pem", flag.MTLS.CAFile)
	assert.Equal(t, "/tmp/option-link.pem", flag.MTLS.CertFile)
	assert.Equal(t, "/tmp/option-link-key.pem", flag.MTLS.KeyFile)
}

// An ignored flag keeps parsing so the rest of the command line still applies,
// but neither the flag nor its environment variable reaches the runtime.
func TestIgnoredFlagAndItsEnvironmentAreDropped(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{
		"/tmp/app",
		"--link-mtls-key-file", "/tmp/cli-link-key.pem",
		"--link-hub-endpoint", "http://cli-hub.local:7071",
	}
	t.Setenv(EnvMTLSKeyFile, "/tmp/env-link-key.pem")

	flag := &linkflag.Flag{}
	appcli.Handle(flags(flag, Option{IgnoredFlags: []string{FlagMTLSKeyFile}})...)

	assert.Empty(t, flag.MTLS.KeyFile)
	assert.Equal(t, "http://cli-hub.local:7071", flag.HubEndpoint)
}

func TestUnknownIgnoredFlagPanics(t *testing.T) {
	assert.Panics(t, func() { flags(&linkflag.Flag{}, Option{IgnoredFlags: []string{"link-unknown"}}) })
}

// A renamed flag answers to the new name alone: the declared name and the
// environment variable derived from it replace the declared variable.
func TestRenamedFlagUsesTheDerivedNameAndEnvironment(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	renamed := Option{RenamedFlags: map[string]string{FlagMTLSCAFile: "ca-file"}}

	os.Args = []string{"/tmp/app", "--ca-file", "/tmp/cli-ca.pem"}
	fromFlag := &linkflag.Flag{}
	appcli.Handle(flags(fromFlag, renamed)...)
	assert.Equal(t, "/tmp/cli-ca.pem", fromFlag.MTLS.CAFile)

	os.Args = []string{"/tmp/app"}
	t.Setenv(EnvMTLSCAFile, "/tmp/env-ca.pem")
	declaredEnv := &linkflag.Flag{}
	appcli.Handle(flags(declaredEnv, renamed)...)
	assert.Empty(t, declaredEnv.MTLS.CAFile)

	t.Setenv("VINE_CA_FILE", "/tmp/derived-ca.pem")
	derivedEnv := &linkflag.Flag{}
	appcli.Handle(flags(derivedEnv, renamed)...)
	assert.Equal(t, "/tmp/derived-ca.pem", derivedEnv.MTLS.CAFile)
}

func TestRenamedFlagCannotBeIgnored(t *testing.T) {
	assert.Panics(t, func() {
		flags(&linkflag.Flag{}, Option{
			IgnoredFlags: []string{FlagMTLSCAFile},
			RenamedFlags: map[string]string{FlagMTLSCAFile: "ca-file"},
		})
	})
}

func TestUnknownRenamedFlagPanics(t *testing.T) {
	assert.Panics(t, func() {
		flags(&linkflag.Flag{}, Option{RenamedFlags: map[string]string{"link-unknown": "ca-file"}})
	})
}

func TestApplyOptionKeepsUnsetFlagValues(t *testing.T) {
	flag := &linkflag.Flag{
		HubEndpoint:   "http://cli-hub.local:7071",
		IngressListen: "127.0.0.1:8080",
		MTLS: mtls.Files{
			CAFile:   "/tmp/cli-ca.pem",
			CertFile: "/tmp/cli-link.pem",
			KeyFile:  "/tmp/cli-link-key.pem",
		},
	}

	applyOption(flag, Option{})

	assert.Equal(t, "http://cli-hub.local:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:8080", flag.IngressListen)
	assert.Equal(t, "/tmp/cli-ca.pem", flag.MTLS.CAFile)
	assert.Equal(t, "/tmp/cli-link.pem", flag.MTLS.CertFile)
	assert.Equal(t, "/tmp/cli-link-key.pem", flag.MTLS.KeyFile)
}

func TestFlagsParseHubEndpointAndIngressListen(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{
		"/tmp/vine",
		"--link-hub-endpoint", "http://10.0.0.8:7071",
		"--link-ingress-listen", "127.0.0.1:8080",
		"--link-mtls-ca-file", "/tmp/ca.pem",
		"--link-mtls-cert-file", "/tmp/link.pem",
		"--link-mtls-key-file", "/tmp/link-key.pem",
	}

	flag := &linkflag.Flag{}
	appcli.Handle(flags(flag, Option{})...)

	assert.Equal(t, "http://10.0.0.8:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:8080", flag.IngressListen)
	assert.Equal(t, "/tmp/ca.pem", flag.MTLS.CAFile)
	assert.Equal(t, "/tmp/link.pem", flag.MTLS.CertFile)
	assert.Equal(t, "/tmp/link-key.pem", flag.MTLS.KeyFile)
}

func TestFlagsParseHubEndpointAndIngressListenFromEnv(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"/tmp/vine"}
	t.Setenv(EnvHubEndpoint, "http://10.0.0.9:7071")
	t.Setenv(EnvIngressListen, "127.0.0.1:9090")
	t.Setenv(EnvMTLSCAFile, "/tmp/env-ca.pem")
	t.Setenv(EnvMTLSCertFile, "/tmp/env-link.pem")
	t.Setenv(EnvMTLSKeyFile, "/tmp/env-link-key.pem")

	flag := &linkflag.Flag{}
	appcli.Handle(flags(flag, Option{})...)

	assert.Equal(t, "http://10.0.0.9:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:9090", flag.IngressListen)
	assert.Equal(t, "/tmp/env-ca.pem", flag.MTLS.CAFile)
	assert.Equal(t, "/tmp/env-link.pem", flag.MTLS.CertFile)
	assert.Equal(t, "/tmp/env-link-key.pem", flag.MTLS.KeyFile)
}

type _RecordingApp struct {
	name   string
	events *[]string
}

func (a *_RecordingApp) Name() string { return a.name }
func (*_RecordingApp) Start()         {}

func (a *_RecordingApp) StopGracefully() {
	*a.events = append(*a.events, a.name+".stop", a.name+".wait")
}

func (*_RecordingApp) StartAndWait() {}
