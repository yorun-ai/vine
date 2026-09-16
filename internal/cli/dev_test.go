package cli

import (
	"context"
	"go.yorun.ai/vine/internal/core/ex"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yorun.ai/vine/internal/app"
	coreapp "go.yorun.ai/vine/internal/core/app"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/core/skel"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/util/vcode"
)

func TestRunDev(t *testing.T) {
	originalStart := startDevRuntime
	t.Cleanup(func() { startDevRuntime = originalStart })

	var got _DevOption
	startDevRuntime = func(option _DevOption) {
		got = option
	}

	result := run([]string{
		"dev",
		"--link-api-listen", "127.0.0.1:8079",
		"--db-sqlite-file", "/tmp/vine-dev.sqlite",
		"--seed-data-file", "/tmp/vine-dev.yaml",
	})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if got.LinkAPIListen != "127.0.0.1:8079" {
		t.Fatalf("unexpected link API listen: %q", got.LinkAPIListen)
	}
	if got.DBSQLiteFile != "/tmp/vine-dev.sqlite" {
		t.Fatalf("unexpected SQLite file: %q", got.DBSQLiteFile)
	}
	if got.SeedHubDataFile != "/tmp/vine-dev.yaml" {
		t.Fatalf("unexpected seed YAML file: %q", got.SeedHubDataFile)
	}
}

func TestRunDevUsesDefaults(t *testing.T) {
	originalStart := startDevRuntime
	t.Cleanup(func() { startDevRuntime = originalStart })

	var got _DevOption
	startDevRuntime = func(option _DevOption) {
		got = option
	}

	result := run([]string{"dev"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if got.LinkAPIListen != "127.0.0.1:7079" {
		t.Fatalf("unexpected link API listen default: %q", got.LinkAPIListen)
	}
	if got.DBSQLiteFile != "" || got.DBPostgresURL != "" {
		t.Fatal("expected runtime to select temporary storage")
	}
}

func TestPrepareDevHubFlagDefaultsToNoDB(t *testing.T) {
	flag, cleanup := prepareDevHubFlag(_DevOption{SeedHubDataFile: "seed.yaml"})
	defer cleanup()
	flag.Normalize(true)
	if !flag.NoDB || flag.DBSQLiteFile != "" {
		t.Fatal("expected no-db without a SQLite file")
	}
}

func TestDevRuntimeLifecycleOrder(t *testing.T) {
	var events []string
	runtime := &_DevRuntime{
		hub:    &_DevRecordingApp{name: "hub", events: &events},
		portal: &_DevRecordingApp{name: "portal", events: &events},
		link:   &_DevRecordingApp{name: "link", events: &events},
	}

	runtime.Start()
	runtime.StopGracefully()

	want := []string{
		"hub.start", "portal.start", "link.start",
		"link.stop", "portal.stop", "hub.stop",
	}
	if len(events) != len(want) {
		t.Fatalf("unexpected lifecycle events: %v", events)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("unexpected lifecycle events: %v", events)
		}
	}
}

func TestDevRuntimeAcceptsNetworkAppRegistration(t *testing.T) {
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	if err := os.WriteFile(seedPath, []byte("appConfigs:\n  - name: demo.Config\n    value: '{\"enabled\":true}'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	linkListen := freeDevTestListenAddress(t)
	runtime := newDevRuntime(_DevOption{
		SeedHubDataFile: seedPath,
		LinkAPIListen:   linkListen,
	})
	runtime.Start()
	t.Cleanup(func() {
		runtime.StopGracefully()
		runtime.cleanup()
	})

	externalApp := meta.MustNewApp("dev.external", "1.0.0", "550e8400-e29b-41d4-a716-446655440000")
	linkRPCClient := newDevTestRPCClient("http://"+linkListen+coreapp.PathRpcInvoke, externalApp)
	bootClient := linkskeled.NewBootServiceClientER(linkRPCClient)
	bootInfo, bootErr := bootClient.GetInfo()
	if bootErr != nil {
		t.Fatalf("get Link boot info: %v", bootErr)
	}
	if bootInfo.SkipDomainSchemas {
		t.Fatal("network apps must upload domain schemas to an inproc Hub")
	}

	domainSchema := &skel.DomainSchema{
		Domain: "devprobe",
		Hash:   "devprobe-domain",
		Full:   true,
		Data: []*skel.DataSchema{{
			Name:           "Probe",
			SkelName:       "devprobe.Probe",
			Hash:           "devprobe-data",
			TypeParameters: []string{},
			Members:        []*skel.MemberSchema{},
		}},
	}
	registryClient := linkskeled.NewRegistryServiceClientER(linkRPCClient)
	registerErr := registryClient.Register(linkskeled.AppRegistration{
		ServiceHandlers: []linkskeled.ServiceHandlerRegistration{},
		WebHandlers:     []linkskeled.WebHandlerRegistration{},
		EventListeners:  []linkskeled.EventListenerRegistration{},
		TaskRunners:     []linkskeled.TaskRunnerRegistration{},
		DomainSchemas:   []skel.JSON{skel.JSON(vcode.MustMarshalJsonS(domainSchema))},
	})
	if registerErr != nil {
		t.Fatalf("register external app: %v", registerErr)
	}
	t.Cleanup(func() {
		if err := registryClient.Unregister(); err != nil {
			t.Errorf("unregister external app: %v", err)
		}
	})

	hubRPCClient := newDevTestRPCClient(
		inproc.Endpoint(hubapp.HubAdminInprocHostPath, coreapp.PathRpcInvoke),
		externalApp,
	)
	readOnlyMethod, _ := spec.GetMethodInfo("vine.hub.admin.MaintenanceApiService", "configReadOnly")
	readOnly, accessErr := hubRPCClient.InvokeAs[bool](readOnlyMethod, nil)
	if accessErr != nil || !readOnly {
		t.Fatalf("expected initialized read-only Hub: %v %v", readOnly, accessErr)
	}
	configMethod, _ := spec.GetMethodInfo("vine.hub.admin.AppConfigApiService", "list")
	configs, configErr := hubRPCClient.InvokeAs[[]hubskeled.AppConfigListItem](configMethod, nil)
	if configErr != nil || len(configs) != 1 {
		t.Fatalf("expected seeded config: %v %v", configs, configErr)
	}
	removeMethod, _ := spec.GetMethodInfo("vine.hub.admin.AppConfigApiService", "remove")
	arguments := reflect.New(removeMethod.ArgumentsType())
	arguments.Elem().FieldByName("Id").SetInt(int64(configs[0].Id))
	_, removeErr := hubRPCClient.InvokeAs[bool](removeMethod, arguments.Interface())
	if removeErr == nil || removeErr.Code() != ex.PermissionDenied {
		t.Fatalf("expected read-only rejection, got %v", removeErr)
	}
	method, ok := spec.GetMethodInfo("vine.hub.admin.SkeletonApiService", "listData")
	if !ok {
		t.Fatal("missing SkeletonApiService.listData")
	}
	controlRPCClient := newDevTestRPCClient(
		inproc.Endpoint(hubapp.HubControlInprocHostPath, coreapp.PathRpcInvoke),
		externalApp,
	)
	if _, err := controlRPCClient.InvokeAs[[]hubskeled.SkeletonData](method, nil); err == nil {
		t.Fatal("expected Hub Control API to reject admin service")
	}
	items, listErr := hubRPCClient.InvokeAs[[]hubskeled.SkeletonData](method, nil)
	if listErr != nil {
		t.Fatalf("list Hub data schemas: %v", listErr)
	}
	for _, item := range items {
		if item.SkelName == "devprobe.Probe" {
			return
		}
	}
	t.Fatal("expected external app domain schema to reach the inproc Hub")
}

func freeDevTestListenAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve listen address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release listen address: %v", err)
	}
	return address
}

func newDevTestRPCClient(endpoint string, clientApp meta.App) *client.Client {
	return client.New(client.Option{
		Context:             meta.NewContext(context.Background(), meta.InitialTrace(), nil, meta.NewAbsentActor()),
		ClientApp:           clientApp,
		Logger:              logger.New("test:dev"),
		ReturnIfSystemError: true,
		ServerEndpoint:      endpoint,
	})
}

type _DevRecordingApp struct {
	name   string
	events *[]string
}

var _ app.App = (*_DevRecordingApp)(nil)

func (a *_DevRecordingApp) Name() string {
	return a.name
}

func (a *_DevRecordingApp) Start() {
	*a.events = append(*a.events, a.name+".start")
}

func (a *_DevRecordingApp) StopGracefully() {
	*a.events = append(*a.events, a.name+".stop")
}

func (*_DevRecordingApp) StartAndWait() {}
