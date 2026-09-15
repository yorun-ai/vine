package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubwatch"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerLoadsRpcgwSiteFromWatch(t *testing.T) {
	manager := newTestManager(map[string]string{
		watched.FormatPortalSiteKey("demo-api"): vcode.MustMarshalJsonS(watched.PortalSite{
			Name: "demo-api",
			Type: siteTypeRpcgw,
			ActorVia: watched.PortalActorVia{
				ActorSkelName: "demo.UserActor",
			},
			RpcgwConfig: &watched.PortalRpcgwConfig{
				Services: []watched.PortalRpcgwService{{SkelName: "demo.UserService"}},
			},
		}),
	})

	target, ok := manager.Site("demo-api")
	if !ok {
		t.Fatal("expected demo-api site")
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/invoke/demo.UserService/Get", nil)
	request.Header.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeJson)
	rpchttp.EncodeTraceToHeader(request.Header, meta.InitialTrace())
	request.Header.Set(rpchttp.HeaderRpcClient, "name=demo.client,version=0.0.0,instanceId=123e4567-e89b-12d3-a456-426614174001")
	request.Header.Set("Authorization", "key token")

	target.Serve(testContext(recorder, request))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Header().Get(rpchttp.HeaderRpcStatus) != string(ex.ServiceUnavailable) {
		t.Fatalf("unexpected rpc status: %s", recorder.Header().Get(rpchttp.HeaderRpcStatus))
	}
	if !strings.Contains(recorder.Body.String(), "demo.UserService") {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestManagerDoesNotLoadPortalRulesAsSites(t *testing.T) {
	manager := newTestManager(map[string]string{
		watched.FormatPortalSiteKey("demo-web"): vcode.MustMarshalJsonS(watched.PortalSite{
			Name: "demo-web",
			Type: siteTypeWebgw,
			WebgwConfig: &watched.PortalWebgwConfig{
				WebName: "demo.Web",
			},
		}),
		watched.FormatPortalRuleKey("demo-web"): vcode.MustMarshalJsonS(watched.PortalRule{
			Name:          "demo-web",
			RouteType:     "SITE",
			RouteSiteName: "demo-web",
		}),
	})

	target, ok := manager.Site("demo-web")
	if !ok {
		t.Fatal("expected demo-web site")
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/", nil)
	target.Serve(testContext(recorder, request))
	if strings.Contains(recorder.Body.String(), "portal site type is not supported") {
		t.Fatalf("portal rule was loaded as site: %s", recorder.Body.String())
	}
}

func TestManagerSiteReturnsFalseForUnknownSite(t *testing.T) {
	manager := newTestManager(nil)

	_, ok := manager.Site("missing-site")
	if ok {
		t.Fatal("expected missing site")
	}
}

func TestManagerHandlesSiteEvents(t *testing.T) {
	manager := newTestManager(nil)

	manager.handleSiteEvent(hubapiwatch.Event{
		Kind: hubapiwatch.EventKindUpsert,
		Key:  watched.FormatPortalSiteKey("demo-web"),
		Value: vcode.MustMarshalJsonS(watched.PortalSite{
			Name: "demo-web",
			Type: siteTypeWebgw,
			WebgwConfig: &watched.PortalWebgwConfig{
				WebName: "admin@demo.app",
			},
		}),
	})
	if _, ok := manager.Site("demo-web"); !ok {
		t.Fatal("expected demo-web site")
	}

	manager.handleSiteEvent(hubapiwatch.Event{
		Kind: hubapiwatch.EventKindDelete,
		Key:  watched.FormatPortalSiteKey("demo-web"),
	})
	if _, ok := manager.Site("demo-web"); ok {
		t.Fatal("expected deleted demo-web site")
	}
}

func TestManagerUpsertReplacesSiteWithoutRemovingName(t *testing.T) {
	manager := newTestManager(nil)
	oldSite := &_TestSite{name: "demo-api"}
	newSite := &_TestSite{name: "demo-api"}
	manager.sitesByKey[watched.FormatPortalSiteKey("demo-api")] = oldSite
	manager.sitesByName["demo-api"] = oldSite

	stopSite(manager.replaceSite(watched.FormatPortalSiteKey("demo-api"), newSite))

	if !oldSite.stopped {
		t.Fatal("expected old site to stop")
	}
	target, ok := manager.Site("demo-api")
	if !ok {
		t.Fatal("expected demo-api site")
	}
	if target != newSite {
		t.Fatal("expected new site")
	}
}

func TestManagerUpsertUpdatesSameSiteTypeInPlace(t *testing.T) {
	manager := newTestManager(map[string]string{
		watched.FormatPortalSiteKey("demo-api"): vcode.MustMarshalJsonS(watched.PortalSite{
			Name: "demo-api",
			Type: siteTypeRpcgw,
			RpcgwConfig: &watched.PortalRpcgwConfig{
				Services: []watched.PortalRpcgwService{{SkelName: "demo.UserService"}},
			},
		}),
	})
	before, ok := manager.Site("demo-api")
	if !ok {
		t.Fatal("expected demo-api site")
	}

	manager.handleSiteEvent(hubapiwatch.Event{
		Kind: hubapiwatch.EventKindUpsert,
		Key:  watched.FormatPortalSiteKey("demo-api"),
		Value: vcode.MustMarshalJsonS(watched.PortalSite{
			Name: "demo-api",
			Type: siteTypeRpcgw,
			RpcgwConfig: &watched.PortalRpcgwConfig{
				Services: []watched.PortalRpcgwService{{SkelName: "demo.OrderService"}},
			},
		}),
	})

	after, ok := manager.Site("demo-api")
	if !ok {
		t.Fatal("expected demo-api site")
	}
	if after != before {
		t.Fatal("expected same site instance")
	}
}

func TestManagerSiteReturnsRegisteredUnknownKindSite(t *testing.T) {
	manager := newTestManager(map[string]string{
		watched.FormatPortalSiteKey("demo-unknown"): vcode.MustMarshalJsonS(watched.PortalSite{
			Name: "demo-unknown",
			Type: "unknown",
		}),
	})
	target, ok := manager.Site("demo-unknown")
	if !ok {
		t.Fatal("expected demo-unknown site")
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/api", nil)

	target.Serve(testContext(recorder, request))
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "unknown") {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func testContext(recorder http.ResponseWriter, request *http.Request) *spec.Context {
	return &spec.Context{
		Request:        request,
		ResponseWriter: recorder,
		RemoteAddr:     "192.0.2.1",
	}
}

func newTestManager(valuesByKey map[string]string) *Manager {
	epmgrManager := newTestEpmgr(valuesByKey)
	manager := &Manager{
		Context: context.Background(),
		App:     meta.MustNewApp("vine.portal", "0.0.0", "123e4567-e89b-12d3-a456-426614174099"),
		Watch:   hubwatch.NewTestClient(valuesByKey),
		Access:  newTestAccess(),
		Epmgr:   epmgrManager,
	}
	manager.DIInit()
	return manager
}

func newTestEpmgr(valuesByKey map[string]string) *epmgr.Manager {
	manager := &epmgr.Manager{
		Context: context.Background(),
		Watch:   hubwatch.NewTestClient(valuesByKey),
	}
	manager.DIInit()
	return manager
}

func newTestAccess() *access.Access {
	watchClient := newTestSchemaWatch()
	epmgrManager := &epmgr.Manager{
		Context: context.Background(),
		Watch:   watchClient,
	}
	epmgrManager.DIInit()
	manager := &access.Access{
		Context: context.Background(),
		Watch:   watchClient,
		Epmgr:   epmgrManager,
	}
	manager.DIInit()
	return manager
}

func newTestSchemaWatch() *hubwatch.Client {
	watchClient := hubwatch.NewTestClient(map[string]string{
		watched.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(watched.SchemaActor{
			SkelName: "demo.UserActor",
			AuthCredential: &skel.DataSchema{
				SkelName: "demo.UserCredential",
				Members: []*skel.MemberSchema{
					{Name: "key"},
				},
			},
			AuthInfo: &skel.DataSchema{SkelName: "demo.UserInfo"},
		}),
		watched.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.SchemaService{
			SkelName: "demo.UserService",
			AuthMode: skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{SkelName: "demo.UserActor"},
			},
			Methods: []*skel.MethodSchema{
				{SkelName: "Get", AuthMode: skel.AuthModeNoAuth},
			},
		}),
	})
	return watchClient
}

type _TestSite struct {
	name    string
	stopped bool
}

func (s *_TestSite) Name() string {
	return s.name
}

func (s *_TestSite) Serve(ctx *spec.Context) {
}

func (s *_TestSite) Update(config watched.PortalSite) bool {
	return false
}

func (s *_TestSite) Stop() {
	s.stopped = true
}
