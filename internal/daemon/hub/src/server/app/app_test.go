package app

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/infra/rdb"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	adminimpl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/admin"
	controlimpl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/control"
	adminapi "go.yorun.ai/vine/internal/daemon/hub/src/server/mod/admin"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/initializer"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo"
	repodb "go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db"
)

var (
	testSQLitePath     string
	testSQLitePathOnce sync.Once
)

func collectModuleTypes(spec *HubApp) []reflect.Type {
	var moduleTypes []reflect.Type
	spec.InitModules(func(moduleType reflect.Type) {
		moduleTypes = append(moduleTypes, moduleType)
	})
	return moduleTypes
}

func initTestConfigDatabase(component *repodb.HubDatabase) *rdb.DatabaseManager {
	manager := new(rdb.DatabaseManager)
	manager.InitComponent(component)
	return manager
}

func sharedTestSQLitePath(t *testing.T) string {
	t.Helper()

	testSQLitePathOnce.Do(func() {
		root, err := os.MkdirTemp("", "vine-hub-app-*")
		require.NoError(t, err)
		testSQLitePath = filepath.Join(root, "hub.sqlite")
	})
	return testSQLitePath
}

func TestHubAppDIInitNormalizesFlagAndSetsRunFlag(t *testing.T) {
	spec := &HubApp{
		AppFlag:    &internalapp.RunFlag{},
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			MQMode:         flag.MQModeNATS,
			Store:          flag.StoreSQLite,
			DBSQLiteFile:   "/tmp/hub.sqlite",
			MQNatsEndpoint: "nats://127.0.0.1:4222",
		},
	}

	spec.DIInit()

	if got, want := spec.InternalAttrs.CurrentApp.Version(), buildinfo.MustVineVersion(); got != want {
		t.Fatalf("unexpected daemon version: got %q, want Vine version %q", got, want)
	}

	assert.Equal(t, flag.HubDefaultAdminListen, spec.AppFlag.ListenAddr)
	assert.Equal(t, flag.StoreSQLite, spec.Flag.Store)
	assert.Equal(t, flag.HubDefaultControlListen, spec.Flag.ControlListen)
	assert.Equal(t, flag.HubDefaultAdminListen, spec.Flag.AdminListen)
	assert.Equal(t, flag.HubDefaultWatchListen, spec.Flag.WatchListen)
	assert.Equal(t, "/tmp/hub.sqlite", spec.Flag.DBSQLiteFile)
	assert.Equal(t, "nats://127.0.0.1:4222", spec.Flag.MQNatsEndpoint)
	assert.Equal(t, flag.MQModeNATS, spec.Flag.MQMode)
}

func TestHubAppDIInitUsesLogicalNameInInprocMode(t *testing.T) {
	spec := &HubApp{
		AppFlag:    &internalapp.RunFlag{},
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag: &flag.Flag{
			Store:        flag.StoreSQLite,
			DBSQLiteFile: "/tmp/hub.sqlite",
			MQMode:       flag.MQModeEmbedded,
		},
	}

	spec.DIInit()

	if got, want := spec.InternalAttrs.CurrentApp.Version(), buildinfo.MustVineVersion(); got != want {
		t.Fatalf("unexpected daemon version: got %q, want Vine version %q", got, want)
	}

	assert.Equal(t, "vine.hub", spec.Name())
	assert.Equal(t, "vine.hub", spec.InternalAttrs.CurrentApp.Name())
	assert.Empty(t, spec.AppFlag.ListenAddr)
	assert.Empty(t, spec.Flag.ControlListen)
	assert.Empty(t, spec.Flag.AdminListen)
	assert.Empty(t, spec.Flag.WatchListen)
	assert.Equal(t, flag.MQModeEmbedded, spec.Flag.MQMode)
}

func TestHubAdminModuleServesOnlyAdminAPIHandlers(t *testing.T) {
	// The Admin API moved off the application listener: Hub serves no routes of
	// its own, and the admin module keeps the Admin API away from the Control
	// API services.
	assert.NotImplements(t, (*internalapp.ServicerSpec)(nil), new(HubApp))
	assert.Contains(t, collectModuleTypes(new(HubApp)), internalapp.T[*adminapi.Server]())

	handlerTypes := adminapi.HandlerTypes()
	assert.NotContains(t, handlerTypes, internalapp.T[*controlimpl.InfoServiceServerImpl]())
	assert.NotContains(t, handlerTypes, internalapp.T[*controlimpl.RegistryServiceServerImpl]())
	assert.Contains(t, handlerTypes, internalapp.T[*adminimpl.AppConfigApiServiceServerImpl]())
}

// TestInitializerResolvesFromCommonBindings is a composition smoke test: the
// module graph of the initializer must be constructible from the bindings a Hub
// application installs, without a running daemon.
func TestInitializerResolvesFromCommonBindings(t *testing.T) {
	component := &repodb.HubDatabase{
		Flag: &flag.Flag{
			Store:        flag.StoreSQLite,
			DBSQLiteFile: sharedTestSQLitePath(t),
		},
	}
	manager := initTestConfigDatabase(component)
	t.Cleanup(manager.AfterAppStop)
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)

	spec := &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			Store:        flag.StoreSQLite,
			AdminListen:  flag.HubDefaultAdminListen,
			DBSQLiteFile: sharedTestSQLitePath(t),
			WatchListen:  flag.HubDefaultWatchListen,
		},
	}

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.New("vine:test"))
			manager.Bind(b)
			b.BindInstance(watchServer)
			b.Bind(di.T[*initializer.Initializer]()).In(di.SingletonScope)
			b.Bind(di.T[*flag.Flag]()).ToInstance(spec.Flag)
			spec.BindCommon(b)
		},
	)

	var module *initializer.Initializer
	assert.NotPanics(t, func() {
		module = injector.Get(di.T[*initializer.Initializer]()).Interface().(*initializer.Initializer)
	})
	assert.NotNil(t, module)
	assert.IsType(t, &repo.AppConfigRepo{Access: new(configaccess.Access)}, module.AppConfigRepo)
}

func TestHubConfigurationLifecycle(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		name := "no-db"
		if persistent {
			name = "sqlite"
		}
		t.Run(name, func(t *testing.T) {
			flags := &flag.Flag{SeedHubData: "appConfigs: [{name: demo.Config, value: {enabled: true}}]"}
			if persistent {
				flags.DBSQLiteFile = filepath.Join(t.TempDir(), "hub.sqlite")
			}
			flags.Normalize(true)
			manager := initTestConfigDatabase(&repodb.HubDatabase{Flag: flags})
			t.Cleanup(manager.AfterAppStop)
			watch := watchserver.NewServerForTest()
			t.Cleanup(watch.AfterAppStop)
			access := new(configaccess.Access)
			spec := &HubApp{Flag: flags, InprocFlag: &internalapp.InternalInprocFlag{Enabled: true}}
			injector := di.NewInjector(func(b *di.Binder) {
				b.Bind(di.T[context.Context]()).ToInstance(context.Background())
				b.BindInstance(logger.New("vine:test:lifecycle"))
				b.BindInstance(flags)
				b.BindInstance(spec.InprocFlag)
				b.BindInstance(watch)
				b.BindInstance(access)
				manager.Bind(b)
				spec.BindCommon(b)
				b.Bind(di.T[*adminimpl.AdminApiServiceServerImpl]()).In(di.SingletonScope)
				b.Bind(di.T[*seeder.Seeder]()).In(di.SingletonScope)
				b.Bind(di.T[*initializer.Initializer]()).In(di.SingletonScope)
			})
			require.False(t, access.ReadOnly())
			module := injector.Get(di.T[*initializer.Initializer]()).Interface().(*initializer.Initializer)
			siteRepo := injector.Get(di.T[core.PortalSiteRepo]()).Interface().(core.PortalSiteRepo)
			entryRepo := injector.Get(di.T[core.PortalEntryRepo]()).Interface().(core.PortalEntryRepo)
			appConfigRepo := injector.Get(di.T[core.AppConfigRepo]()).Interface().(core.AppConfigRepo)
			ruleRepo := injector.Get(di.T[core.PortalRuleRepo]()).Interface().(core.PortalRuleRepo)
			certRepo := injector.Get(di.T[core.PortalCertRepo]()).Interface().(core.PortalCertRepo)
			require.Equal(t, !persistent, access.ReadOnly())
			item, ok := appConfigRepo.GetByName("demo.Config")
			require.True(t, ok)
			require.Equal(t, `{"enabled":true}`, item.Value)
			// The seed declares no site or rule, so Hub stores none.
			require.Empty(t, siteRepo.List())
			require.Empty(t, ruleRepo.List())
			actions := []struct {
				name string
				call func()
			}{
				{"save config", func() { appConfigRepo.Save(&core.AppConfig{Name: "new.config", Value: "{}", Version: 1}) }},
				{"remove config", func() { appConfigRepo.Remove(item.Id) }},
				{"save site", func() {
					siteRepo.Save(&core.PortalSite{Name: "new.site", Type: core.PortalSiteTypeWEBGW, WebName: "demo.Web"})
				}},
				{"remove site", func() { siteRepo.Remove(-1) }},
				{"save rule", func() {
					// A rule belongs to the entry that serves its access, so Hub
					// stores the entry before the rule.
					entry := &core.PortalEntry{Name: "http:80", Enabled: true, Protocol: "http", Http: &core.PortalEntryHTTP{Http: "ENABLED", Https: "DISABLED", HttpPort: 80}}
					entryRepo.Save(entry)
					ruleRepo.Save(&core.PortalRule{Name: "new.rule", EntryId: entry.Id, RouteType: "SITE", RouteSiteName: "new.site"})
				}},
				{"remove rule", func() { ruleRepo.Remove(-1) }},
				{"save cert", func() { certRepo.Save(&core.PortalCert{Name: "new.cert", Domains: []string{"demo.local"}}) }},
				{"remove cert", func() { certRepo.Remove(-1) }},
			}
			for _, action := range actions {
				t.Run(action.name, func(t *testing.T) {
					if persistent {
						require.NotPanics(t, action.call)
					} else {
						require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", action.call)
					}
				})
			}
			require.NotPanics(t, func() {
				module.RegistryCore.Register(core.AppRegistration{Name: "demo", InstanceId: "test", Version: "test"})
			})
			require.NotEmpty(t, module.RegistryCore.RegistryRepo.ListAppStatuses())
			require.NotEmpty(t, module.DescriptorRepo.ListDomainDescriptorViews())
			service := injector.Get(di.T[*adminimpl.AdminApiServiceServerImpl]()).Interface().(*adminimpl.AdminApiServiceServerImpl)
			require.Equal(t, !persistent, service.ReadOnly())
		})
	}
}
