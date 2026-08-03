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

type _TestAppSpec struct {
	app.Application
}

func (*_TestAppSpec) Name() string {
	return "linked.test"
}

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
		HubEndpoint:   "http://option-hub.local:7071",
		IngressListen: "127.0.0.1:9090",
		MTLSCAFile:    "/tmp/option-ca.pem",
		MTLSCertFile:  "/tmp/option-link.pem",
		MTLSKeyFile:   "/tmp/option-link-key.pem",
	})

	assert.Equal(t, "http://option-hub.local:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:9090", flag.IngressListen)
	assert.Equal(t, "/tmp/option-ca.pem", flag.MTLS.CAFile)
	assert.Equal(t, "/tmp/option-link.pem", flag.MTLS.CertFile)
	assert.Equal(t, "/tmp/option-link-key.pem", flag.MTLS.KeyFile)
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
		"--hub-endpoint", "http://10.0.0.8:7071",
		"--ingress-listen", "127.0.0.1:8080",
		"--mtls-ca-file", "/tmp/ca.pem",
		"--mtls-cert-file", "/tmp/link.pem",
		"--mtls-key-file", "/tmp/link-key.pem",
	}

	flag := &linkflag.Flag{}
	appcli.Handle(flags(flag)...)

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
	t.Setenv(envHubEndpoint, "http://10.0.0.9:7071")
	t.Setenv(envIngressListen, "127.0.0.1:9090")
	t.Setenv(envMTLSCAFile, "/tmp/env-ca.pem")
	t.Setenv(envMTLSCertFile, "/tmp/env-link.pem")
	t.Setenv(envMTLSKeyFile, "/tmp/env-link-key.pem")

	flag := &linkflag.Flag{}
	appcli.Handle(flags(flag)...)

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
