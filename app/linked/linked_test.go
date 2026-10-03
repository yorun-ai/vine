package linked

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/app"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/mtls"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/src/server/app"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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

	t.Setenv("VINE_CA_FILE", "/tmp/unused-ca.pem")
	prefixedEnv := &linkflag.Flag{}
	appcli.Handle(flags(prefixedEnv, renamed)...)
	assert.Empty(t, prefixedEnv.MTLS.CAFile)

	t.Setenv("CA_FILE", "/tmp/derived-ca.pem")
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

type linkedLifecycleSpec struct {
	app.Application
	app.ServicerEnabled
}

func (*linkedLifecycleSpec) Name() string                           { return "linked.test" }
func (*linkedLifecycleSpec) InitModules(add app.TypeAdder)          { add(app.T[*linkedLifecycleModule]()) }
func (*linkedLifecycleSpec) ServicerInitHandlers(add app.TypeAdder) { add(app.T[*linkedPingServer]()) }

type linkedLifecycleModule struct{ app.BaseModule }

var linkedLifecycleEvents []string

func (*linkedLifecycleModule) BeforeAppStart() error {
	linkedLifecycleEvents = append(linkedLifecycleEvents, "start")
	return nil
}
func (*linkedLifecycleModule) AfterAppStop() {
	linkedLifecycleEvents = append(linkedLifecycleEvents, "stop")
}

type linkedPingAPI interface{ Ping() string }
type linkedPingAPIER interface{ Ping() (string, ex.Error) }
type linkedPingWrapper struct{ server linkedPingAPI }

func (w *linkedPingWrapper) Ping() (string, ex.Error) { return w.server.Ping(), nil }

type linkedPingDefault struct{}
type linkedPingDefaultER struct{}
type linkedPingServer struct{ linkedPingDefault }

func (*linkedPingServer) Ping() string { return "pong" }

type linkedWatchClient struct {
	watch.Client
	endpoint string
}

func (c *linkedWatchClient) InitOption(option *watch.Option) {
	option.Endpoint = c.endpoint
	option.Username = watch.LinkUsername
}

func TestLinkedRuntimeLifecycle(t *testing.T) {
	if os.Getenv("VINE_TEST_LINKED_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLinkedRuntimeLifecycle$", "-test.count=1")
		command.Env = append(os.Environ(), "VINE_TEST_LINKED_CHILD=1")
		output, err := command.CombinedOutput()
		require.NoError(t, err, "%s", output)
		return
	}
	// App specs and registries are process singletons. The child owns this runtime.
	os.Args = []string{"linked-lifecycle-test"}
	controlAddr, watchAddr, adminAddr := linkedTestAddresses(t)
	hub := internalapp.NewInternal[*hubapp.HubApp](internalapp.With(&hubflag.Flag{
		ControlListen: controlAddr, WatchListen: watchAddr, AdminListen: adminAddr,
		DBSQLiteFile: filepath.Join(t.TempDir(), "hub.sqlite"),
	}))
	hub.Start()
	t.Cleanup(hub.StopGracefully)
	service := &rpcspec.ServiceSpec{Type: rpcspec.ServiceSpecTypeServer, Name: "LinkedPing", SkelName: "test.LinkedPing", Hash: "test",
		ServerType: reflect.TypeFor[linkedPingAPI](), ERServerType: reflect.TypeFor[linkedPingAPIER](),
		WrapperERServerCtor: func(server linkedPingAPI) linkedPingAPIER { return &linkedPingWrapper{server: server} },
		DefaultServerType:   reflect.TypeFor[*linkedPingDefault](), DefaultERServerType: reflect.TypeFor[*linkedPingDefaultER](),
		Methods: []*rpcspec.MethodSpec{{Name: "Ping", SkelName: "Ping", ResultType: reflect.TypeFor[string]()}},
	}
	rpcspec.Register(service)
	application := NewWithOption[*linkedLifecycleSpec](Option{LinkHubEndpoint: "http://" + controlAddr, LinkIngressListen: "127.0.0.1:0"})
	application.Start()
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			application.StopGracefully()
		}
	})
	assert.Equal(t, []string{"start"}, linkedLifecycleEvents)
	client := &linkedWatchClient{endpoint: "redis://" + watchAddr}
	manager := &watch.ClientManager{Context: t.Context()}
	t.Cleanup(manager.AfterAppStop)
	manager.InitComponent(client)
	values, subscription := client.LoadListAndSubscribe(t.Context(), watched.FormatRpcServiceRegistrationPrefix("test.LinkedPing"), func(watch.Event) {})
	subscription.Start()
	require.Len(t, values, 1, "linked application must publish service discovery to the external Hub")
	var registrationKey string
	for key := range values {
		registrationKey = key
	}
	registration := vcode.MustUnmarshalJsonS[watched.RpcServiceRegistration](values[registrationKey])
	caller := meta.MustNewApp(registration.AppName, registration.AppVersion, registration.AppInstanceId)
	linker := link.NewLinker(caller, true, "")
	rpc := rpcclient.New(rpcclient.Option{Context: meta.NewContext(t.Context(), meta.InitialTrace(), nil, meta.NewAbsentActor()), ClientApp: caller, Logger: logger.New("linked:test"), ReturnIfSystemError: true, ServerEndpoint: linker.RpcProxyEndpoint()})
	result, err := rpc.Invoke(service.Methods[0].Info(), nil, rpcclient.WithTimeout(time.Second))
	require.Nil(t, err)
	assert.Equal(t, "pong", result)
	application.StopGracefully()
	stopped = true
	assert.Equal(t, []string{"start", "stop"}, linkedLifecycleEvents)
	_, registered := client.Load(registrationKey)
	assert.False(t, registered, "shutdown must unregister the app before stopping Link")
	_, err = rpc.Invoke(service.Methods[0].Info(), nil, rpcclient.WithTimeout(time.Second))
	assert.NotNil(t, err, "Link must stop accepting invocations after shutdown")
}

func linkedTestAddresses(t *testing.T) (string, string, string) {
	t.Helper()
	var listeners []net.Listener
	var addresses []string
	defer func() {
		for _, listener := range listeners {
			_ = listener.Close()
		}
	}()
	for range 3 {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		listeners = append(listeners, listener)
		addresses = append(addresses, listener.Addr().String())
	}
	return addresses[0], addresses[1], addresses[2]
}
