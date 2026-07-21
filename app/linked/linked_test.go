package linked

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/internal/appcli"
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
	}

	applyOption(flag, Option{
		HubEndpoint:   "http://option-hub.local:7071",
		IngressListen: "127.0.0.1:9090",
	})

	assert.Equal(t, "http://option-hub.local:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:9090", flag.IngressListen)
}

func TestApplyOptionKeepsUnsetFlagValues(t *testing.T) {
	flag := &linkflag.Flag{
		HubEndpoint:   "http://cli-hub.local:7071",
		IngressListen: "127.0.0.1:8080",
	}

	applyOption(flag, Option{})

	assert.Equal(t, "http://cli-hub.local:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:8080", flag.IngressListen)
}

func TestFlagsParseHubEndpointAndIngressListen(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{
		"/tmp/vine",
		"--hub-endpoint", "http://10.0.0.8:7071",
		"--ingress-listen", "127.0.0.1:8080",
	}

	flag := &linkflag.Flag{}
	appcli.Handle(flags(flag)...)

	assert.Equal(t, "http://10.0.0.8:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:8080", flag.IngressListen)
}

func TestFlagsParseHubEndpointAndIngressListenFromEnv(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"/tmp/vine"}
	t.Setenv(envHubEndpoint, "http://10.0.0.9:7071")
	t.Setenv(envIngressListen, "127.0.0.1:9090")

	flag := &linkflag.Flag{}
	appcli.Handle(flags(flag)...)

	assert.Equal(t, "http://10.0.0.9:7071", flag.HubEndpoint)
	assert.Equal(t, "127.0.0.1:9090", flag.IngressListen)
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
