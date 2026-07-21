package entry

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerReconcileEntriesBindsPortAndRules(t *testing.T) {
	manager := &Manager{
		entryRulesByName: map[string]redised.PortalRule{},
		entriesByKey:     map[_Key]*_Entry{},
		SiteManager:      newTestSiteManager("admin@demo.app", "home@demo.app"),
	}

	manager.entryRulesByName["admin"] = redised.PortalRule{
		Name:       "admin",
		Scheme:     string(spec.SchemeHTTPS),
		Host:       "demo.local",
		Port:       8443,
		PathPrefix: "/admin",
		TargetType: "SITE",
		SiteName:   "admin@demo.app",
	}
	manager.entryRulesByName["home"] = redised.PortalRule{
		Name:       "home",
		Scheme:     string(spec.SchemeHTTPS),
		Host:       "demo.local",
		Port:       8443,
		PathPrefix: "/",
		TargetType: "SITE",
		SiteName:   "home@demo.app",
	}
	manager.entryRulesByName["redirect"] = redised.PortalRule{
		Name:               "redirect",
		Scheme:             string(spec.SchemeHTTP),
		Host:               "demo.local",
		Port:               8080,
		PathPrefix:         "/old",
		TargetType:         "PERMANENT_REDIRECT",
		RedirectionPattern: "https://demo.local/new",
	}

	manager.reconcileEntriesLocked()

	httpsKey := _Key{scheme: spec.SchemeHTTPS, port: 8443}
	httpKey := _Key{scheme: spec.SchemeHTTP, port: 8080}
	assert.Len(t, manager.entriesByKey, 2)
	assert.Len(t, manager.entriesByKey[httpsKey].rules, 2)
	assert.Equal(t, "admin@demo.app", manager.entriesByKey[httpsKey].rules[0].targetSiteName)
	assert.Len(t, manager.entriesByKey[httpKey].rules, 1)
	assert.Equal(t, "redirection", manager.entriesByKey[httpKey].rules[0].redirectionSite.Name())
}

func TestManagerReconcileEntriesDeduplicatesBySchemeAndPort(t *testing.T) {
	manager := &Manager{
		entryRulesByName: map[string]redised.PortalRule{},
		entriesByKey:     map[_Key]*_Entry{},
		SiteManager:      newTestSiteManager("admin@demo.app", "home@demo.app"),
	}

	manager.entryRulesByName["admin"] = redised.PortalRule{
		Name:       "admin",
		Scheme:     string(spec.SchemeHTTPS),
		Port:       8443,
		TargetType: "SITE",
		SiteName:   "admin@demo.app",
	}
	manager.entryRulesByName["home"] = redised.PortalRule{
		Name:       "home",
		Scheme:     string(spec.SchemeHTTP),
		Port:       8443,
		TargetType: "SITE",
		SiteName:   "home@demo.app",
	}

	manager.reconcileEntriesLocked()

	httpsKey := _Key{scheme: spec.SchemeHTTPS, port: 8443}
	httpKey := _Key{scheme: spec.SchemeHTTP, port: 8443}
	assert.Len(t, manager.entriesByKey, 2)
	assert.Len(t, manager.entriesByKey[httpsKey].rules, 1)
	assert.Len(t, manager.entriesByKey[httpKey].rules, 1)
}

func TestManagerReconcileEntriesUpdatesExistingPortalRules(t *testing.T) {
	existing := newEntry(spec.SchemeHTTPS, 8443, nil)
	existing.SetOrUpdateRules([]*_Rule{{name: "old"}})
	manager := &Manager{
		entryRulesByName: map[string]redised.PortalRule{},
		entriesByKey: map[_Key]*_Entry{
			{scheme: spec.SchemeHTTPS, port: 8443}: existing,
		},
		SiteManager: newTestSiteManager("admin@demo.app"),
	}

	manager.entryRulesByName["admin"] = redised.PortalRule{
		Name:       "admin",
		Scheme:     string(spec.SchemeHTTPS),
		Port:       8443,
		TargetType: "SITE",
		SiteName:   "admin@demo.app",
	}

	manager.reconcileEntriesLocked()

	assert.Same(t, existing, manager.entriesByKey[_Key{scheme: spec.SchemeHTTPS, port: 8443}])
	assert.Len(t, existing.rules, 1)
	assert.Equal(t, "admin", existing.rules[0].name)
}

func TestManagerAfterAppStartStartsEntriesCreatedBeforeStart(t *testing.T) {
	prev := listenEntryTCP
	var listenAddress string
	listenEntryTCP = func(network, address string) (net.Listener, error) {
		listenAddress = address
		return newTestListener(), nil
	}
	t.Cleanup(func() {
		listenEntryTCP = prev
	})

	existing := newEntry(spec.SchemeHTTP, 8080, nil)
	manager := &Manager{
		entryRulesByName: map[string]redised.PortalRule{
			"admin": {
				Name:       "admin",
				Scheme:     string(spec.SchemeHTTP),
				Port:       8080,
				TargetType: "SITE",
				SiteName:   "admin@demo.app",
			},
		},
		entriesByKey: map[_Key]*_Entry{
			{scheme: spec.SchemeHTTP, port: 8080}: existing,
		},
		SiteManager: newTestSiteManager("admin@demo.app"),
	}

	manager.AfterAppStart()

	assert.True(t, existing.started)
	assert.Equal(t, "0.0.0.0:8080", listenAddress)
}

func newTestSiteManager(names ...string) *site.Manager {
	valuesByKey := map[string]string{}
	for _, name := range names {
		valuesByKey[redised.FormatPortalSiteKey(name)] = vcode.MustMarshalJsonS(redised.PortalSite{
			Name: name,
			Type: "RPCGW",
			RpcgwConfig: &redised.PortalRpcgwConfig{
				Services: []redised.PortalRpcgwService{{SkelName: "demo.UserService"}},
			},
		})
	}
	epmgrManager := &epmgr.Manager{
		Context: context.Background(),
		Redis:   hubredis.NewTestClient(valuesByKey),
	}
	epmgrManager.DIInit()
	manager := &site.Manager{
		Context: context.Background(),
		Redis:   hubredis.NewTestClient(valuesByKey),
		Epmgr:   epmgrManager,
	}
	manager.DIInit()
	return manager
}

type _TestListener struct {
}

func newTestListener() *_TestListener {
	return &_TestListener{}
}

func (l *_TestListener) Accept() (net.Conn, error) {
	return nil, net.ErrClosed
}

func (l *_TestListener) Close() error {
	return nil
}

func (*_TestListener) Addr() net.Addr {
	return &net.TCPAddr{Port: 8080}
}
