package admin

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/meta"
	vinehttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	vrpchttp "go.yorun.ai/vrpc/transport/http"
)

func TestPortalEntryServiceMapsRulesAndTargetSites(t *testing.T) {
	ruleRepo := &_NamedPortalRuleRepoSpy{items: map[string]*core.PortalRule{
		"demo-rule": {
			Id:              1,
			Name:            "demo-rule",
			EntryId:         1,
			MatchPathPrefix: "/demo",
			RouteType:       core.PortalRuleRouteTypeSite,
			RouteSiteName:   "demo-site",
			FieldSources:    core.FieldSources{"/name": {Source: "app/default"}},
		},
	}}
	siteRepo := &_PortalSiteRepoSpy{items: map[string]*core.PortalSite{
		"demo-site": {
			Id:            2,
			Name:          "demo-site",
			Type:          core.PortalSiteTypeWEBGW,
			ActorSkelName: "demo.Actor",
			WebName:       "demo.Web",
			WebMountPath:  "/demo",
			FieldSources:  core.FieldSources{"/name": {Source: "app/default"}},
		},
	}}
	service := &PortalEntryApiServiceServerImpl{PortalEntryCore: &core.PortalEntryCore{
		PortalEntryRepo: newTestPortalEntryRepoSpy(&core.PortalEntry{Id: 1, Name: "http:8080", Protocol: "http", Http: &core.PortalEntryHTTP{Http: "ENABLED", Https: "DISABLED", HttpPort: 8080}}),
		PortalRuleRepo:  ruleRepo,
		PortalSiteRepo:  siteRepo,
	}}

	entries := service.List()

	require.Len(t, entries, 1)
	assert.Equal(t, 1, entries[0].Id)
	assert.Equal(t, "http:8080", entries[0].Name)
	require.Len(t, entries[0].Rules, 1)

	// The entry payload embeds the list shape of its rule and site: it carries the
	// values a list needs, never the seed provenance of the detail responses.
	assert.Equal(t, "demo-rule", entries[0].Rules[0].Rule.Name)
	require.NotNil(t, entries[0].Rules[0].Site)
	assert.Equal(t, "demo-site", entries[0].Rules[0].Site.Name)
	assert.Equal(t, "/demo", entries[0].Rules[0].Site.WebMountPath)
}

func TestPortalEntryServiceCreatesAndRemovesEntry(t *testing.T) {
	entryRepo := newTestPortalEntryRepoSpy()
	service := &PortalEntryApiServiceServerImpl{PortalEntryCore: &core.PortalEntryCore{
		PortalEntryRepo: entryRepo,
		PortalRuleRepo:  &_NamedPortalRuleRepoSpy{items: map[string]*core.PortalRule{}},
		PortalSiteRepo:  &_PortalSiteRepoSpy{items: map[string]*core.PortalSite{}},
		PortalCertRepo:  new(_PortalCertRepoSpy),
	}}

	created := service.Create(skeled.PortalEntryCreation{Name: "web", Protocol: "http", Http: new(skeled.PortalEntryHttp{HttpPort: 8080, Https: "DISABLED", Autohttps: "DISABLED"}), ListenIPs: []string{"127.0.0.1", "::1"}})

	// An entry routes no rule when the operator creates it.
	require.NotZero(t, created.Id)
	assert.Equal(t, "web", created.Name)
	assert.Equal(t, "http", created.Protocol)
	assert.Equal(t, 8080, created.Http.HttpPort)
	assert.Equal(t, []string{"127.0.0.1", "::1"}, created.ListenIPs)
	assert.Empty(t, created.Rules)

	updated := service.Update(created.Id, skeled.PortalEntryUpdate{Name: new("console"), ListenIPs: new([]string{"127.0.0.2"})})
	assert.Equal(t, "console", updated.Name)
	assert.Equal(t, 8080, updated.Http.HttpPort)
	assert.Equal(t, []string{"127.0.0.2"}, updated.ListenIPs)

	assert.Len(t, service.List(), 1)

	service.Remove(created.Id)

	assert.Empty(t, service.List())
}

func TestPortalEntryProtocolVocabulary(t *testing.T) {
	require.NotPanics(t, func() { validatePortalEntryProtocol(new("http")) })
	require.Panics(t, func() { validatePortalEntryProtocol(new("")) })
	require.Panics(t, func() { validatePortalEntryProtocol(new("tcp")) })
	patch := toCoreHTTPConfig(new(skeled.PortalEntryHttp{Https: "DISABLED", Autohttps: "DISABLED"}))
	config := patch.Effective(false)
	require.True(t, config.HttpEnabled)
	require.False(t, config.HttpsEnabled)
	require.False(t, config.AutoHTTPSEnabled)
}

func TestPortalEntryServiceRetainsAutomaticSwitches(t *testing.T) {
	certRepo := new(_PortalCertRepoSpy)
	entryRepo := newTestPortalEntryRepoSpy()
	entryRepo.certs = certRepo
	service := &PortalEntryApiServiceServerImpl{PortalEntryCore: &core.PortalEntryCore{PortalCertRepo: certRepo, PortalEntryRepo: entryRepo, PortalRuleRepo: &_NamedPortalRuleRepoSpy{items: map[string]*core.PortalRule{}}, PortalSiteRepo: &_PortalSiteRepoSpy{items: map[string]*core.PortalSite{}}}}
	created := service.Create(skeled.PortalEntryCreation{Name: "web", Protocol: "http", Http: &skeled.PortalEntryHttp{Host: "demo.local"}})
	require.EqualValues(t, "AUTO", created.Http.Http)
	require.EqualValues(t, "AUTO", created.Http.Https)
	require.EqualValues(t, "AUTO", created.Http.Autohttps)
	require.True(t, created.EffectiveHttp.HttpEnabled)
	require.False(t, created.EffectiveHttp.HttpsEnabled)
	require.Panics(t, func() {
		service.Update(created.Id, skeled.PortalEntryUpdate{Http: &skeled.PortalEntryHttp{Host: "demo.local", Https: "ENABLED"}})
	})
	certRepo.Save(new(core.PortalCert{Name: "demo", Enabled: true, Domains: []string{"demo.local"}}))
	updated := service.Update(created.Id, skeled.PortalEntryUpdate{Http: &skeled.PortalEntryHttp{Host: "demo.local", Https: "ENABLED", HttpsPort: 8443}})
	require.True(t, updated.EffectiveHttp.HttpsEnabled)
	require.Equal(t, 8443, updated.EffectiveHttp.HttpsPort)
	unchanged := service.Update(created.Id, skeled.PortalEntryUpdate{Name: new("renamed")})
	require.True(t, unchanged.EffectiveHttp.HttpsEnabled)
	automatic := service.Update(created.Id, skeled.PortalEntryUpdate{Http: &skeled.PortalEntryHttp{Host: "demo.local"}})
	require.EqualValues(t, "AUTO", automatic.Http.Https)
	require.True(t, automatic.EffectiveHttp.HttpsEnabled)
}

func TestPortalEntryHTTPModeWireContract(t *testing.T) {
	for _, contentType := range []string{vrpchttp.ContentTypeJson, vrpchttp.ContentTypeCbor} {
		t.Run(contentType, func(t *testing.T) {
			for _, field := range []string{"http", "https", "autohttps"} {
				for _, value := range []any{nil, true, false, 1, "", "enabled", "ON", "UNSPECIFIED"} {
					t.Run(fmt.Sprintf("%s=%v", field, value), func(t *testing.T) {
						body, err := vrpchttp.EncodeRequest(map[string]any{"creation": map[string]any{"protocol": "http", "http": map[string]any{"host": "demo.local", field: value}}}, contentType)
						require.NoError(t, err)
						input, err := decodePortalEntryCreation(t, body, contentType)
						if err == nil {
							require.Panics(t, func() {
								core.NormalizePortalEntry(core.PortalEntry{Protocol: input.Protocol, Http: toCoreHTTPConfig(input.Http)})
							})
						}
					})
				}
			}
			for _, modes := range []map[string]any{{}, {"http": "AUTO", "https": "AUTO", "autohttps": "AUTO"}, {"http": "ENABLED", "https": "DISABLED", "autohttps": "DISABLED"}} {
				body, err := vrpchttp.EncodeRequest(map[string]any{"creation": map[string]any{"protocol": "http", "http": modes}}, contentType)
				require.NoError(t, err)
				input, err := decodePortalEntryCreation(t, body, contentType)
				require.NoError(t, err)
				entry := core.NormalizePortalEntry(core.PortalEntry{Protocol: input.Protocol, Http: toCoreHTTPConfig(input.Http)})
				require.True(t, entry.Http.Http.Valid())
				require.True(t, entry.Http.Https.Valid())
				require.True(t, entry.Http.AutoHTTPS.Valid())
				effective := entry.Http.Effective(false)
				require.JSONEq(t, fmt.Sprintf(`{"host":"","httpEnabled":true,"httpPort":80,"httpsEnabled":false,"httpsPort":443,"autohttpsEnabled":false,"httpsReason":%q}`, effective.HTTPSReason), vcode.MustMarshalJsonS(toServerEffectiveHTTP(effective)))
			}
		})
	}
}

func decodePortalEntryCreation(t *testing.T, body []byte, contentType string) (skeled.PortalEntryCreation, error) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/vine.hub.admin.PortalEntryApiService/create", bytes.NewReader(body))
	request.Header.Set(vrpchttp.HeaderContentType, contentType)
	request.Header.Set(vrpchttp.HeaderAccept, vrpchttp.ContentTypeJson)
	vinehttp.EncodeTraceToHeader(request.Header, meta.InitialTrace())
	client, err := meta.NewApp("test", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	require.NoError(t, err)
	vinehttp.EncodeClientToHeader(request.Header, client)
	decoded, err := vinehttp.DecodeRequest(request)
	if err != nil {
		return skeled.PortalEntryCreation{}, err
	}
	t.Cleanup(decoded.Cancel)
	return reflect.ValueOf(decoded.Arguments()).Elem().FieldByName("Creation").Interface().(skeled.PortalEntryCreation), nil
}
