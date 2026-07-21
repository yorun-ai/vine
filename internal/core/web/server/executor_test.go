package server

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/web/spec"
)

type _ContainerExecutorRecorder struct {
	WebCtx      spec.Context
	GinCtx      *gin.Context
	HasDeadline bool
}

type _ContainerExecutorHandler struct {
	_DefaultContainerExecutorWebServer
	WebCtx   spec.Context                `inject:""`
	GinCtx   *gin.Context                `inject:""`
	Recorder *_ContainerExecutorRecorder `inject:""`
}

func (h *_ContainerExecutorHandler) Routes(r *spec.Router) {
	r.GET("/inspect", h.Inspect)
}

func (h *_ContainerExecutorHandler) Inspect() {
	h.Recorder.WebCtx = h.WebCtx
	h.Recorder.GinCtx = h.GinCtx
	_, h.Recorder.HasDeadline = h.GinCtx.Request.Context().Deadline()
	h.GinCtx.Status(http.StatusAccepted)
}

type _ContainerExecutorWebServer interface {
	spec.Handler

	mustBeContainerExecutorWebServer()
}

type _DefaultContainerExecutorWebServer struct {
}

func (*_DefaultContainerExecutorWebServer) Routes(*spec.Router) {
	panic("method routes is not implemented")
}

func (*_DefaultContainerExecutorWebServer) mustBeContainerExecutorWebServer() {}

func registerContainerExecutorWeb() {
	spec.Register(&spec.WebSpec{
		Name:              "ContainerExecutorWeb",
		SkelName:          "demo.user.ContainerExecutorWeb",
		ServerType:        reflect.TypeFor[_ContainerExecutorWebServer](),
		DefaultServerType: reflect.TypeFor[*_DefaultContainerExecutorWebServer](),
	})
}

func newContainerExecutorTestServer(recorder *_ContainerExecutorRecorder) *Server {
	spec.ResetRegistryForTest()
	registerContainerExecutorWeb()

	return NewServer(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_ContainerExecutorHandler]()},
		Executor: NewContainerExecutor(nil, []di.BindApplier{
			func(b *di.Binder) {
				b.Bind(di.T[*_ContainerExecutorRecorder]()).ToInstance(recorder)
			},
		}),
	})
}

func TestContainerExecutorInjectsExecutionScopeValuesWithActorHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	actor := meta.NewAuthenticatedActorForTest()
	initiator, err := meta.NewInitiator("portal.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "https", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}

	recorder := &_ContainerExecutorRecorder{}
	server := newContainerExecutorTestServer(recorder)
	trace := meta.InitialTrace()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ContainerExecutorWeb/inspect", nil)
	encodeTestTraceToHeader(request.Header, trace)
	encodeTestInitiatorToHeader(request.Header, initiator)
	encodeTestActorToHeader(request.Header, actor)
	responseRecorder := httptest.NewRecorder()
	server.HTTPHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", responseRecorder.Code)
	}
	if recorder.GinCtx == nil {
		t.Fatalf("expected gin context to be injected")
	}
	if recorder.WebCtx == nil {
		t.Fatalf("expected web context to be injected")
	}
	if recorder.WebCtx.Route() == nil {
		t.Fatalf("expected route on web context")
	}
	if recorder.WebCtx.Route().Path() != "/demo.user.ContainerExecutorWeb/inspect" {
		t.Fatalf("unexpected route path: %s", recorder.WebCtx.Route().Path())
	}
	if recorder.WebCtx.Trace() == nil {
		t.Fatalf("expected trace to be injected")
	}
	if recorder.WebCtx.Trace().Id() != trace.Id() {
		t.Fatalf("unexpected trace id: got=%s want=%s", recorder.WebCtx.Trace().Id(), trace.Id())
	}
	if recorder.WebCtx.Trace().ParentSpan() != trace.Span() {
		t.Fatalf("unexpected parent span: got=%s want=%s", recorder.WebCtx.Trace().ParentSpan(), trace.Span())
	}
	if recorder.WebCtx.Trace().Span() == trace.Span() {
		t.Fatalf("expected child span to differ from parent span")
	}
	if got := responseRecorder.Header().Get(spec.HeaderWebTrace); got != "id="+recorder.WebCtx.Trace().Id()+",span="+recorder.WebCtx.Trace().Span() {
		t.Fatalf("unexpected response trace: got=%s", got)
	}
	if recorder.WebCtx.Initiator() == nil || recorder.WebCtx.Initiator().Name() != initiator.Name() {
		t.Fatalf("unexpected initiator: %#v", recorder.WebCtx.Initiator())
	}
	if recorder.WebCtx.Actor() == nil || recorder.WebCtx.Actor().Type() != meta.ActorTypeAuthenticated {
		t.Fatalf("unexpected actor: %#v", recorder.WebCtx.Actor())
	}
}

func TestContainerExecutorReturnsBadRequestWhenRequestMetaInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	initiator, err := meta.NewInitiator("portal.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "https", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}
	trace := meta.InitialTrace()

	tests := []struct {
		name   string
		header func(http.Header)
	}{
		{
			name: "trace missing",
		},
		{
			name: "initiator malformed",
			header: func(header http.Header) {
				encodeTestTraceToHeader(header, trace)
				header.Set(spec.HeaderWebInitiator, "bad-base64")
			},
		},
		{
			name: "actor missing",
			header: func(header http.Header) {
				encodeTestTraceToHeader(header, trace)
				encodeTestInitiatorToHeader(header, initiator)
			},
		},
		{
			name: "actor malformed",
			header: func(header http.Header) {
				encodeTestTraceToHeader(header, trace)
				encodeTestInitiatorToHeader(header, initiator)
				header.Set(spec.HeaderWebActor, "bad-base64")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &_ContainerExecutorRecorder{}
			server := newContainerExecutorTestServer(recorder)
			request := httptest.NewRequest(http.MethodGet, "/demo.user.ContainerExecutorWeb/inspect", nil)
			if tt.header != nil {
				tt.header(request.Header)
			}
			responseRecorder := httptest.NewRecorder()
			server.HTTPHandler().ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != http.StatusBadRequest {
				t.Fatalf("unexpected status code: %d", responseRecorder.Code)
			}
			if recorder.WebCtx != nil {
				t.Fatalf("did not expect web context injection on bad request")
			}
		})
	}
}

func TestContainerExecutorAppliesWebOptionsTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	actor := meta.NewAuthenticatedActorForTest()
	initiator, err := meta.NewInitiator("portal.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "https", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}

	recorder := &_ContainerExecutorRecorder{}
	server := newContainerExecutorTestServer(recorder)
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ContainerExecutorWeb/inspect", nil)
	encodeTestTraceToHeader(request.Header, meta.InitialTrace())
	encodeTestInitiatorToHeader(request.Header, initiator)
	encodeTestActorToHeader(request.Header, actor)
	spec.EncodeOptionsToHeader(request.Header, &spec.Options{Timeout: time.Second})
	responseRecorder := httptest.NewRecorder()

	server.HTTPHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", responseRecorder.Code)
	}
	if !recorder.HasDeadline {
		t.Fatal("expected handler request context deadline")
	}
}

func encodeTestTraceToHeader(header http.Header, trace meta.Trace) {
	spec.EncodeTraceToHeader(header, trace)
}

func encodeTestInitiatorToHeader(header http.Header, initiator meta.Initiator) {
	header.Set(spec.HeaderWebInitiator, meta.EncodeInitiatorToBase64(initiator))
}

func encodeTestActorToHeader(header http.Header, actor meta.Actor) {
	header.Set(spec.HeaderWebActor, meta.EncodeActorToBase64(actor))
}
