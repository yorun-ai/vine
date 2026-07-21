package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	coreapp "go.yorun.ai/vine/internal/core/app"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"github.com/google/uuid"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/core/skel"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
)

func TestAppImplStartInprocServerAllowsEmptyEndpointWithoutInprocRoutes(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)

	app := newApp(&testInternalAppSpec{
		InternalApplication: InternalApplication{
			Application: Application{AppFlag: &RunFlag{}},
			InternalAttrs: InternalAttributes{
				Info: testRuntimeApp{
					name:       "test.portal",
					version:    "1.2.3",
					instanceID: "00000000-0000-0000-0000-000000000123",
				},
				Linker:         &testLinker{},
				DisableConsole: true,
			},
		},
	}, flags)
	app.initInjector()
	app.initServers()

	app.inprocFlag.HostPath = ""
	app.startInprocServer()
	app.stopInprocServer()
}

func TestAppImplStartRegistersWebberInprocRoutes(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)

	app := newApp(&testUniqueRouteAppSpec{
		Application: Application{AppFlag: &RunFlag{}},
	}, flags)
	app.initInjector()
	app.initServers()
	app.startInprocServer()
	t.Cleanup(func() {
		app.stopInprocServer()
	})

	endpoint := webinproc.Endpoint(coreapp.InprocHostPath(app.info.InstanceId()), "/web/access")
	req := newTestWebRequest("/web/access/demo.user.TestUniqueRouteWeb/ping")
	resp, err := webinproc.RoundTrip(endpoint, req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if string(body) != "unique" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestAppImplStopUnregistersWebberInprocRoutes(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)

	app := newApp(&testUniqueRouteAppSpec{
		Application: Application{AppFlag: &RunFlag{}},
	}, flags)
	app.initInjector()
	app.initServers()
	app.startInprocServer()
	endpoint := webinproc.Endpoint(coreapp.InprocHostPath(app.info.InstanceId()), "/web/access")
	app.stopInprocServer()

	req := httptest.NewRequest(http.MethodGet, "/web/access/demo.user.TestUniqueRouteWeb/ping", nil)
	if _, err := webinproc.RoundTrip(endpoint, req); err == nil {
		t.Fatal("expected route to be unregistered")
	}
}

func TestAppImplStartInprocServerRegistersConsoleInprocRoute(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)

	app := newApp(&testHelperAppSpec{
		Application: Application{AppFlag: &RunFlag{}},
	}, flags)
	app.initInjector()
	app.initServers()
	app.startInprocServer()
	t.Cleanup(func() {
		app.stopInprocServer()
	})

	endpoint := rpcinproc.Endpoint(coreapp.InprocHostPath(app.info.InstanceId()), coreapp.PathConsole)
	client := appskeled.NewConsoleServiceClientER(newTestInprocRpcClient(app, endpoint))
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestAppImplStartInprocServerRegistersEventerInprocRoute(t *testing.T) {
	ensureEventerEventRegistered()
	testEventerOnGroupID = 0

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)

	app := newApp(&testEventerSpec{
		Application: Application{AppFlag: &RunFlag{}},
	}, flags)
	app.initInjector()
	app.initServers()
	app.startInprocServer()
	t.Cleanup(func() {
		app.stopInprocServer()
	})

	endpoint := rpcinproc.Endpoint(coreapp.InprocHostPath(app.info.InstanceId()), coreapp.PathEvent)
	client := appskeled.NewEventServiceClientER(newTestInprocRpcClient(app, endpoint))
	err := client.OnEvent(appskeled.EventOn{
		Metadata: appskeled.EventOnMeta{
			TraceId:       meta.InitialTrace().Id(),
			TraceSpan:     meta.InitialTrace().Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		EventSkelName: "test.eventer.TestEventerEvent",
		EventJson:     `{"groupId":7}`,
	})
	if err != nil {
		t.Fatalf("OnEvent() error = %v", err)
	}
	if testEventerOnGroupID != 7 {
		t.Fatalf("expected event listener to receive group 7, got %d", testEventerOnGroupID)
	}
}

func TestAppImplStartInprocServerRegistersTaskerInprocRoute(t *testing.T) {
	ensureTaskerTaskRegistered()
	testTaskerRunGroupID = 0

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)

	app := newApp(&testTaskerRunnerSpec{
		Application: Application{AppFlag: &RunFlag{}},
	}, flags)
	app.initInjector()
	app.initServers()
	app.startInprocServer()
	t.Cleanup(func() {
		app.stopInprocServer()
	})

	endpoint := rpcinproc.Endpoint(coreapp.InprocHostPath(app.info.InstanceId()), coreapp.PathTask)
	client := appskeled.NewTaskServiceClientER(newTestInprocRpcClient(app, endpoint))
	err := client.RunTask(appskeled.TaskRun{
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
	if err != nil {
		t.Fatalf("RunTask() error = %v", err)
	}
	if testTaskerRunGroupID != 7 {
		t.Fatalf("expected task runner to receive group 7, got %d", testTaskerRunGroupID)
	}
}

func newTestInprocRpcClient(app *_AppImpl, endpoint string) *rpcclient.Client {
	actor := meta.NewAbsentActor()
	return rpcclient.New(rpcclient.Option{
		Context:        meta.NewContext(context.Background(), meta.InitialTrace(), nil, actor),
		ClientApp:      app.info,
		Logger:         logger.NewLogger(logger.GlobalOption()),
		ServerEndpoint: endpoint,
	})
}
