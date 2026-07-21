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
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerLoadsRpcgwSiteFromRedis(t *testing.T) {
	manager := newTestManager(map[string]string{
		redised.FormatPortalSiteKey("demo-api"): vcode.MustMarshalJsonS(redised.PortalSite{
			Name: "demo-api",
			Type: siteTypeRpcgw,
			ActorVia: redised.PortalActorVia{
				ActorSkelName: "demo.UserActor",
			},
			RpcgwConfig: &redised.PortalRpcgwConfig{
				Services: []redised.PortalRpcgwService{{SkelName: "demo.UserService"}},
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
		redised.FormatPortalSiteKey("demo-web"): vcode.MustMarshalJsonS(redised.PortalSite{
			Name: "demo-web",
			Type: siteTypeWebgw,
			WebgwConfig: &redised.PortalWebgwConfig{
				WebName: "demo.Web",
			},
		}),
		redised.FormatPortalRuleKey("demo-web"): vcode.MustMarshalJsonS(redised.PortalRule{
			Name:       "demo-web",
			TargetType: "SITE",
			SiteName:   "demo-web",
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

	manager.handleSiteEvent(hubapiredis.Event{
		Kind: hubapiredis.EventKindUpsert,
		Key:  redised.FormatPortalSiteKey("demo-web"),
		Value: vcode.MustMarshalJsonS(redised.PortalSite{
			Name: "demo-web",
			Type: siteTypeWebgw,
			WebgwConfig: &redised.PortalWebgwConfig{
				WebName: "admin@demo.app",
			},
		}),
	})
	if _, ok := manager.Site("demo-web"); !ok {
		t.Fatal("expected demo-web site")
	}

	manager.handleSiteEvent(hubapiredis.Event{
		Kind: hubapiredis.EventKindDelete,
		Key:  redised.FormatPortalSiteKey("demo-web"),
	})
	if _, ok := manager.Site("demo-web"); ok {
		t.Fatal("expected deleted demo-web site")
	}
}

func TestManagerUpsertReplacesSiteWithoutRemovingName(t *testing.T) {
	manager := newTestManager(nil)
	oldSite := &_TestSite{name: "demo-api"}
	newSite := &_TestSite{name: "demo-api"}
	manager.sitesByKey[redised.FormatPortalSiteKey("demo-api")] = oldSite
	manager.sitesByName["demo-api"] = oldSite

	stopSite(manager.replaceSite(redised.FormatPortalSiteKey("demo-api"), newSite))

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
		redised.FormatPortalSiteKey("demo-api"): vcode.MustMarshalJsonS(redised.PortalSite{
			Name: "demo-api",
			Type: siteTypeRpcgw,
			RpcgwConfig: &redised.PortalRpcgwConfig{
				Services: []redised.PortalRpcgwService{{SkelName: "demo.UserService"}},
			},
		}),
	})
	before, ok := manager.Site("demo-api")
	if !ok {
		t.Fatal("expected demo-api site")
	}

	manager.handleSiteEvent(hubapiredis.Event{
		Kind: hubapiredis.EventKindUpsert,
		Key:  redised.FormatPortalSiteKey("demo-api"),
		Value: vcode.MustMarshalJsonS(redised.PortalSite{
			Name: "demo-api",
			Type: siteTypeRpcgw,
			RpcgwConfig: &redised.PortalRpcgwConfig{
				Services: []redised.PortalRpcgwService{{SkelName: "demo.OrderService"}},
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
		redised.FormatPortalSiteKey("demo-unknown"): vcode.MustMarshalJsonS(redised.PortalSite{
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
		Redis:   hubredis.NewTestClient(valuesByKey),
		Access:  newTestAccess(),
		Epmgr:   epmgrManager,
	}
	manager.DIInit()
	return manager
}

func newTestEpmgr(valuesByKey map[string]string) *epmgr.Manager {
	manager := &epmgr.Manager{
		Context: context.Background(),
		Redis:   hubredis.NewTestClient(valuesByKey),
	}
	manager.DIInit()
	return manager
}

func newTestAccess() *access.Access {
	redisClient := newTestSchemaRedis()
	epmgrManager := &epmgr.Manager{
		Context: context.Background(),
		Redis:   redisClient,
	}
	epmgrManager.DIInit()
	manager := &access.Access{
		Context: context.Background(),
		Redis:   redisClient,
		Epmgr:   epmgrManager,
	}
	manager.DIInit()
	return manager
}

func newTestSchemaRedis() *hubredis.Client {
	redisClient := hubredis.NewTestClient(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(redised.SchemaActor{
			SkelName: "demo.UserActor",
			AuthCredential: &skel.DataSchema{
				SkelName: "demo.UserCredential",
				Members: []*skel.MemberSchema{
					{Name: "key"},
				},
			},
			AuthInfo: &skel.DataSchema{SkelName: "demo.UserInfo"},
		}),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
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
	return redisClient
}

type _TestSite struct {
	name    string
	stopped bool
	served  int
}

func (s *_TestSite) Name() string {
	return s.name
}

func (s *_TestSite) Serve(ctx *spec.Context) {
	s.served++
}

func (s *_TestSite) Update(config redised.PortalSite) bool {
	return false
}

func (s *_TestSite) Stop() {
	s.stopped = true
}
