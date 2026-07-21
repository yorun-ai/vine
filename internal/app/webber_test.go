package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	coreapp "go.yorun.ai/vine/internal/core/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/runtime"
	web "go.yorun.ai/vine/internal/core/web/spec"
)

type testWebSpec struct {
	Application
	WebberEnabled
}

func (*testWebSpec) Name() string {
	return "test.web"
}

func TestWebberMetaContextUsesApplicationInfo(t *testing.T) {
	parent := context.Background()
	ctx := newMetaContext(parent)

	assert.Equal(t, parent, ctx.(*meta.ContextImpl).Context)
	assert.Nil(t, ctx.Initiator())
	assert.Equal(t, meta.ActorTypeAbsent, ctx.Actor().Type())
	assert.NotNil(t, ctx.Trace())
}

type testWebberContextRecorder struct {
	MetaCtx meta.Context
	GinCtx  *gin.Context
}

type testWebberContextHandler struct {
	defaultTestWebberContextWebServer
	MetaCtx  meta.Context               `inject:""`
	GinCtx   *gin.Context               `inject:""`
	Recorder *testWebberContextRecorder `inject:""`
}

func (h *testWebberContextHandler) Routes(r *web.Router) {
	r.GET("/ctx", h.Inspect)
}

func (h *testWebberContextHandler) Inspect() {
	h.Recorder.MetaCtx = h.MetaCtx
	h.Recorder.GinCtx = h.GinCtx
	h.GinCtx.Status(http.StatusNoContent)
}

type testWebberContextSpec struct {
	Application
	WebberEnabled
}

type testWebberContextWebServer interface {
	web.Handler

	mustBeTestWebberContextWebServer()
}

type defaultTestWebberContextWebServer struct {
}

func (*defaultTestWebberContextWebServer) Routes(*web.Router) {
	panic("method routes is not implemented")
}

func (*defaultTestWebberContextWebServer) mustBeTestWebberContextWebServer() {}

func (*testWebberContextSpec) Name() string {
	return "test.webber.context"
}

func (*testWebberContextSpec) WebberInitHandlers(addHandler TypeAdder) {
	addHandler(reflect.TypeFor[*testWebberContextHandler]())
}

type testUniqueWebberRouteHandler struct {
	defaultTestUniqueRouteWebServer
	GinCtx *gin.Context `inject:""`
}

func (h *testUniqueWebberRouteHandler) Routes(r *web.Router) {
	r.GET("/ping", h.Ping)
}

func (h *testUniqueWebberRouteHandler) Ping() {
	h.GinCtx.String(http.StatusOK, "unique")
}

type testUniqueRouteAppSpec struct {
	Application
	WebberEnabled
}

type testUniqueRouteWebServer interface {
	web.Handler

	mustBeTestUniqueRouteWebServer()
}

type defaultTestUniqueRouteWebServer struct {
}

func (*defaultTestUniqueRouteWebServer) Routes(*web.Router) {
	panic("method routes is not implemented")
}

func (*defaultTestUniqueRouteWebServer) mustBeTestUniqueRouteWebServer() {}

func (*testUniqueRouteAppSpec) Name() string {
	return "test.unique.route"
}

func (*testUniqueRouteAppSpec) WebberBind(*di.Binder) {}

func (*testUniqueRouteAppSpec) WebberInitFilters(TypeAdder) {}

func (*testUniqueRouteAppSpec) WebberInitHandlers(addHandler TypeAdder) {
	addHandler(reflect.TypeFor[*testUniqueWebberRouteHandler]())
}

type testAdminWebberHandler struct {
	defaultTestAdminWebServer
	GinCtx *gin.Context `inject:""`
}

func (h *testAdminWebberHandler) Routes(r *web.Router) {
	r.GET("/admin/ping", h.Ping)
}

func (h *testAdminWebberHandler) Ping() {
	h.GinCtx.String(http.StatusOK, "admin")
}

type testOpenWebberHandler struct {
	defaultTestOpenWebServer
	GinCtx *gin.Context `inject:""`
}

func (h *testOpenWebberHandler) Routes(r *web.Router) {
	r.GET("/open/ping", h.Ping)
}

func (h *testOpenWebberHandler) Ping() {
	h.GinCtx.String(http.StatusOK, "open")
}

type testMultiHandlerWebberAppSpec struct {
	Application
	WebberEnabled
}

type testAdminWebServer interface {
	web.Handler

	mustBeTestAdminWebServer()
}

type defaultTestAdminWebServer struct {
}

func (*defaultTestAdminWebServer) Routes(*web.Router) {
	panic("method routes is not implemented")
}

func (*defaultTestAdminWebServer) mustBeTestAdminWebServer() {}

type testOpenWebServer interface {
	web.Handler

	mustBeTestOpenWebServer()
}

type defaultTestOpenWebServer struct {
}

func (*defaultTestOpenWebServer) Routes(*web.Router) {
	panic("method routes is not implemented")
}

func (*defaultTestOpenWebServer) mustBeTestOpenWebServer() {}

func init() {
	web.Register(&web.WebSpec{
		Name:              "TestWebberContextWeb",
		SkelName:          "demo.user.TestWebberContextWeb",
		ServerType:        reflect.TypeOf((*testWebberContextWebServer)(nil)).Elem(),
		DefaultServerType: reflect.TypeFor[*defaultTestWebberContextWebServer](),
	})
	web.Register(&web.WebSpec{
		Name:              "TestUniqueRouteWeb",
		SkelName:          "demo.user.TestUniqueRouteWeb",
		ServerType:        reflect.TypeOf((*testUniqueRouteWebServer)(nil)).Elem(),
		DefaultServerType: reflect.TypeFor[*defaultTestUniqueRouteWebServer](),
	})
	web.Register(&web.WebSpec{
		Name:              "TestAdminWeb",
		SkelName:          "demo.user.TestAdminWeb",
		ServerType:        reflect.TypeOf((*testAdminWebServer)(nil)).Elem(),
		DefaultServerType: reflect.TypeFor[*defaultTestAdminWebServer](),
	})
	web.Register(&web.WebSpec{
		Name:              "TestOpenWeb",
		SkelName:          "demo.user.TestOpenWeb",
		ServerType:        reflect.TypeOf((*testOpenWebServer)(nil)).Elem(),
		DefaultServerType: reflect.TypeFor[*defaultTestOpenWebServer](),
	})
}

func (*testMultiHandlerWebberAppSpec) Name() string {
	return "test.multi.web"
}

func (*testMultiHandlerWebberAppSpec) WebberInitHandlers(addHandler TypeAdder) {
	addHandler(reflect.TypeFor[*testAdminWebberHandler]())
	addHandler(reflect.TypeFor[*testOpenWebberHandler]())
}

func newTestWebRequest(path string) *http.Request {
	trace := meta.InitialTrace()
	initiator, err := meta.NewInitiator("portal.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "https", "127.0.0.1")
	if err != nil {
		panic(err)
	}
	actor := meta.NewAnonymousActor()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("vweb-trace", "id="+trace.Id()+",span="+trace.Span())
	req.Header.Set("vweb-initiator", meta.EncodeInitiatorToBase64(initiator))
	req.Header.Set("vweb-actor", meta.EncodeActorToBase64(actor))
	return req
}

func TestWebberBindContextMapsWebContextToCommonTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := &testWebberContextRecorder{}
	w := &_Webber{
		appInfo: runtime.App(meta.MustNewApp("demo.service", "1.2.3", "123e4567-e89b-12d3-a456-426614174000")),
		spec:    &testWebberContextSpec{},
		bindAppDeps: func(b *di.Binder) {
			b.Bind(di.T[*testWebberContextRecorder]()).ToInstance(recorder)
		},
	}
	w.init()

	trace := meta.InitialTrace()
	initiator, err := meta.NewInitiator("portal.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "https", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}
	actor := meta.NewAnonymousActor()

	req := httptest.NewRequest(http.MethodGet, "/demo.user.TestWebberContextWeb/ctx", nil)
	req.Header.Set("vweb-trace", "id="+trace.Id()+",span="+trace.Span())
	req.Header.Set("vweb-initiator", meta.EncodeInitiatorToBase64(initiator))
	req.Header.Set("vweb-actor", meta.EncodeActorToBase64(actor))
	resp := httptest.NewRecorder()

	w.httpHandler().ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: %d", resp.Code)
	}
	if recorder.MetaCtx == nil || recorder.GinCtx == nil {
		t.Fatalf("expected context bindings")
	}
	if recorder.MetaCtx.Trace() == nil || recorder.MetaCtx.Trace().Id() != trace.Id() {
		t.Fatalf("unexpected trace: %#v", recorder.MetaCtx.Trace())
	}
	if recorder.MetaCtx.Initiator() == nil || recorder.MetaCtx.Initiator().Name() != initiator.Name() {
		t.Fatalf("unexpected initiator: %#v", recorder.MetaCtx.Initiator())
	}
	if recorder.MetaCtx.Actor() == nil || recorder.MetaCtx.Actor().Type() != actor.Type() {
		t.Fatalf("unexpected actor: %#v", recorder.MetaCtx.Actor())
	}
}

func TestCollectWebberUsesWebAccessPathPrefix(t *testing.T) {
	newWebber(
		&testUniqueRouteAppSpec{},
		runtime.App(meta.MustNewApp("test.unique.web", "1.2.3", "123e4567-e89b-12d3-a456-426614174000")),
		func(*di.Binder) {},
	)

	assert.Equal(t, "/web/access", coreapp.PathWebAccess)
}

func TestUniqueWebberMountsAtDefaultPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testUniqueRouteAppSpec{
		Application: Application{AppFlag: &RunFlag{}},
	}, flags)
	app.initInjector()
	app.initServers()

	req := newTestWebRequest("/web/access/demo.user.TestUniqueRouteWeb/ping")
	recorder := httptest.NewRecorder()

	assert.True(t, app.serveHTTPRoutes(recorder, req))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "unique", recorder.Body.String())
}

func TestSingleWebberMountsMultipleHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testMultiHandlerWebberAppSpec{
		Application: Application{AppFlag: &RunFlag{}},
	}, flags)
	app.initInjector()
	app.initServers()

	adminReq := newTestWebRequest("/web/access/demo.user.TestAdminWeb/admin/ping")
	adminResp := httptest.NewRecorder()
	assert.True(t, app.serveHTTPRoutes(adminResp, adminReq))
	assert.Equal(t, http.StatusOK, adminResp.Code)
	assert.Equal(t, "admin", adminResp.Body.String())

	openReq := newTestWebRequest("/web/access/demo.user.TestOpenWeb/open/ping")
	openResp := httptest.NewRecorder()
	assert.True(t, app.serveHTTPRoutes(openResp, openReq))
	assert.Equal(t, http.StatusOK, openResp.Code)
	assert.Equal(t, "open", openResp.Body.String())
}
