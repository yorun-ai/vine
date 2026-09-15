package app

import (
	"context"
	"go.yorun.ai/vine/buildinfo"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	adminimpl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/admin"
	controlimpl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/controlapi"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/initializer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/scheduler"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/sweeper"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo"
	repodb "go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/schema"
	"go.yorun.ai/vine/internal/infra/rdb"
	"go.yorun.ai/vine/util/vnet"
)

var (
	testSQLitePath     string
	testSQLitePathOnce sync.Once
)

func collectComponentTypes(spec *HubApp) []reflect.Type {
	var componentTypes []reflect.Type
	spec.InitComponents(func(componentType reflect.Type) {
		componentTypes = append(componentTypes, componentType)
	})
	return componentTypes
}

func collectModuleTypes(spec *HubApp) []reflect.Type {
	var moduleTypes []reflect.Type
	spec.InitModules(func(moduleType reflect.Type) {
		moduleTypes = append(moduleTypes, moduleType)
	})
	return moduleTypes
}

func collectServicerHandlerTypes(spec *HubApp) []reflect.Type {
	var handlerTypes []reflect.Type
	spec.ServicerInitHandlers(func(handlerType reflect.Type) {
		handlerTypes = append(handlerTypes, handlerType)
	})
	return handlerTypes
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

	if got, want := spec.InternalAttrs.Info.Version(), buildinfo.MustVineVersion(); got != want {
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

func TestHubAppDIInitKeepsPGConnUrl(t *testing.T) {
	spec := &HubApp{
		AppFlag:    &internalapp.RunFlag{},
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			MQMode:         flag.MQModeNATS,
			Store:          flag.StorePostgreSQL,
			DBPostgresURL:  "postgres://demo:demo@127.0.0.1:5432/hub",
			MQNatsEndpoint: "nats://127.0.0.1:4222",
		},
	}

	spec.DIInit()

	if got, want := spec.InternalAttrs.Info.Version(), buildinfo.MustVineVersion(); got != want {
		t.Fatalf("unexpected daemon version: got %q, want Vine version %q", got, want)
	}

	assert.Equal(t, flag.StorePostgreSQL, spec.Flag.Store)
	assert.Equal(t, "postgres://demo:demo@127.0.0.1:5432/hub", spec.Flag.DBPostgresURL)
	assert.Equal(t, "nats://127.0.0.1:4222", spec.Flag.MQNatsEndpoint)
	assert.Equal(t, flag.MQModeNATS, spec.Flag.MQMode)
	assert.Equal(t, flag.HubDefaultAdminListen, spec.AppFlag.ListenAddr)
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

	if got, want := spec.InternalAttrs.Info.Version(), buildinfo.MustVineVersion(); got != want {
		t.Fatalf("unexpected daemon version: got %q, want Vine version %q", got, want)
	}

	assert.Equal(t, "vine.hub", spec.Name())
	assert.Equal(t, "vine.hub", spec.InternalAttrs.Info.Name())
	assert.Equal(t, "vine/hub/admin", spec.InternalAttrs.InprocHostPath)
	assert.Empty(t, spec.AppFlag.ListenAddr)
	assert.Empty(t, spec.Flag.ControlListen)
	assert.Empty(t, spec.Flag.AdminListen)
	assert.Empty(t, spec.Flag.WatchListen)
	assert.Equal(t, flag.MQModeEmbedded, spec.Flag.MQMode)
}

func TestHubAppMainServicerExcludesControlAPIHandlers(t *testing.T) {
	handlerTypes := collectServicerHandlerTypes(new(HubApp))

	assert.NotContains(t, handlerTypes, internalapp.T[*controlimpl.InfoServiceServerImpl]())
	assert.NotContains(t, handlerTypes, internalapp.T[*controlimpl.RegistryServiceServerImpl]())
	assert.Contains(t, handlerTypes, internalapp.T[*adminimpl.AppConfigApiServiceServerImpl]())
}

func TestHubAppDIInitKeepsEnableNatsOutsideInproc(t *testing.T) {
	spec := &HubApp{
		AppFlag:    &internalapp.RunFlag{},
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			Store:        flag.StoreSQLite,
			DBSQLiteFile: "/tmp/hub.sqlite",
			MQMode:       flag.MQModeEmbedded,
		},
	}

	spec.DIInit()

	if got, want := spec.InternalAttrs.Info.Version(), buildinfo.MustVineVersion(); got != want {
		t.Fatalf("unexpected daemon version: got %q, want Vine version %q", got, want)
	}

	assert.Equal(t, flag.MQModeEmbedded, spec.Flag.MQMode)
}

func TestHubAppModuleTypesIncludesRuntimeModulesInInprocMode(t *testing.T) {
	spec := &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag:       &flag.Flag{},
	}

	assert.Equal(t, []reflect.Type{
		internalapp.T[*syncer.Syncer](),
		internalapp.T[*seeder.Seeder](),
		internalapp.T[*initializer.Initializer](),
		internalapp.T[*scheduler.Scheduler](),
		internalapp.T[*sweeper.Sweeper](),
		internalapp.T[*controlapi.Listener](),
	}, collectModuleTypes(spec))
}

func TestHubAppModuleTypesIncludesRuntimeModulesInNormalMode(t *testing.T) {
	spec := &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag:       &flag.Flag{},
	}

	assert.Equal(t, []reflect.Type{
		internalapp.T[*syncer.Syncer](),
		internalapp.T[*seeder.Seeder](),
		internalapp.T[*initializer.Initializer](),
		internalapp.T[*scheduler.Scheduler](),
		internalapp.T[*sweeper.Sweeper](),
		internalapp.T[*controlapi.Listener](),
	}, collectModuleTypes(spec))
}

func TestHubAppModuleTypesIncludesRuntimeModulesWhenEnableNats(t *testing.T) {
	spec := &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag:       &flag.Flag{MQMode: flag.MQModeEmbedded},
	}

	assert.Equal(t, []reflect.Type{
		internalapp.T[*syncer.Syncer](),
		internalapp.T[*seeder.Seeder](),
		internalapp.T[*initializer.Initializer](),
		internalapp.T[*scheduler.Scheduler](),
		internalapp.T[*sweeper.Sweeper](),
		internalapp.T[*controlapi.Listener](),
	}, collectModuleTypes(spec))
}

func TestHubAppComponentTypesReturnsSQLiteDatabaseWhenSourceIsSQLite(t *testing.T) {
	spec := &HubApp{
		Flag: &flag.Flag{Store: flag.StoreSQLite},
	}

	assert.Equal(t, []reflect.Type{
		internalapp.T[*configaccess.Access](),
		internalapp.T[*repodb.HubDatabase](),
		internalapp.T[*natsserver.NATSServer](),
		internalapp.T[*watchserver.Server](),
	}, collectComponentTypes(spec))
}

func TestHubAppComponentTypesReturnsPGDatabaseWhenSourceIsPG(t *testing.T) {
	spec := &HubApp{
		Flag: &flag.Flag{Store: flag.StorePostgreSQL},
	}

	assert.Equal(t, []reflect.Type{
		internalapp.T[*configaccess.Access](),
		internalapp.T[*repodb.HubDatabase](),
		internalapp.T[*natsserver.NATSServer](),
		internalapp.T[*watchserver.Server](),
	}, collectComponentTypes(spec))
}

func TestConfigDatabaseInitOptionForSQLite(t *testing.T) {
	spec := &repodb.HubDatabase{
		Flag: &flag.Flag{
			Store:        flag.StoreSQLite,
			DBSQLiteFile: "/tmp/hub.sqlite",
		},
	}
	option := &rdb.Option{}

	spec.InitOption(option)

	assert.Equal(t, "sqlite:///tmp/hub.sqlite", option.ConnURL)
	daoTypes := []reflect.Type{}
	spec.InitDao(func(daoType reflect.Type) {
		daoTypes = append(daoTypes, daoType)
	})
	assert.Equal(t, []reflect.Type{
		rdb.T[*model.AppConfigDao](),
		rdb.T[*model.PortalCertDao](),
		rdb.T[*model.PortalRuleDao](),
		rdb.T[*model.MetadataDao](),
		rdb.T[*model.PortalSiteDao](),
	}, daoTypes)
}

func TestConfigDatabaseInitOptionForPG(t *testing.T) {
	spec := &repodb.HubDatabase{
		Flag: &flag.Flag{
			Store:         flag.StorePostgreSQL,
			DBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
		},
	}
	option := &rdb.Option{}

	spec.InitOption(option)

	assert.Equal(t, "postgres://demo:demo@127.0.0.1:5432/hub", option.ConnURL)
	daoTypes := []reflect.Type{}
	spec.InitDao(func(daoType reflect.Type) {
		daoTypes = append(daoTypes, daoType)
	})
	assert.Equal(t, []reflect.Type{
		rdb.T[*model.AppConfigDao](),
		rdb.T[*model.PortalCertDao](),
		rdb.T[*model.PortalRuleDao](),
		rdb.T[*model.MetadataDao](),
		rdb.T[*model.PortalSiteDao](),
	}, daoTypes)
}

func TestHubAppBindCommonProvidesDBAppConfigRepoForSQLite(t *testing.T) {
	configRepo := newHubBoundAppConfigRepo(t, &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag:       &flag.Flag{Store: flag.StoreSQLite},
	})

	assert.IsType(t, &repo.DBAppConfigRepo{Access: new(configaccess.Access)}, configRepo)
}

func TestHubAppBindCommonProvidesDBAppConfigRepoForPG(t *testing.T) {
	configRepo := newHubBoundAppConfigRepo(t, &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			Store:         flag.StorePostgreSQL,
			DBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
		},
	})

	assert.IsType(t, &repo.DBAppConfigRepo{Access: new(configaccess.Access)}, configRepo)
}

func TestHubAppBindCommonProvidesMemorySchemaRepoForDBInInprocMode(t *testing.T) {
	schemaRepo := newHubBoundSchemaRepo(t, &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag:       &flag.Flag{Store: flag.StoreSQLite},
	})

	assert.IsType(t, &schema.MemorySchemaRepo{}, schemaRepo)
}

func TestHubAppBindCommonProvidesMemorySchemaRepoForDB(t *testing.T) {
	schemaRepo := newHubBoundSchemaRepo(t, &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag:       &flag.Flag{Store: flag.StoreSQLite},
	})

	assert.IsType(t, &schema.MemorySchemaRepo{}, schemaRepo)
}

func testDashboardURL() *vnet.HttpURL {
	return vnet.MustParseHttpURL(flag.HubDefaultDashboardURL)
}

func TestHubAppBindCommonProvidesDBAppConfigRepoForInitializerWithSQLite(t *testing.T) {
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
			DashboardURL: testDashboardURL(),
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
	assert.IsType(t, &repo.DBAppConfigRepo{Access: new(configaccess.Access)}, module.AppConfigRepo)
}

func TestConfigDatabaseBindProvidesAppConfigRepoDAO(t *testing.T) {
	component := &repodb.HubDatabase{
		Flag: &flag.Flag{
			Store:        flag.StoreSQLite,
			DBSQLiteFile: sharedTestSQLitePath(t),
		},
	}
	manager := initTestConfigDatabase(component)
	t.Cleanup(manager.AfterAppStop)

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.New("vine:test"))
			manager.Bind(b)
			b.Bind(di.T[*repo.DBAppConfigRepo]()).In(di.TransientScope)
		},
	)

	execution := injector.StartExecution()
	defer execution.CompleteExecution()

	daoValue := execution.Get(rdb.T[*model.AppConfigDao]())
	require.True(t, daoValue.IsValid())
	gormDBMethod := daoValue.MethodByName("GormDB")
	require.True(t, gormDBMethod.IsValid())
	gormDBResult := gormDBMethod.Call(nil)
	require.Len(t, gormDBResult, 1)
	assert.False(t, gormDBResult[0].IsNil())
}

func newHubBoundAppConfigRepo(t *testing.T, spec *HubApp) core.AppConfigRepo {
	t.Helper()

	component := &repodb.HubDatabase{
		Flag: &flag.Flag{
			Store:        flag.StoreSQLite,
			DBSQLiteFile: sharedTestSQLitePath(t),
		},
	}
	manager := initTestConfigDatabase(component)
	t.Cleanup(manager.AfterAppStop)

	daoInjector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.New("vine:test"))
			manager.Bind(b)
		},
	)
	daoExecution := daoInjector.StartExecution()
	defer daoExecution.CompleteExecution()
	daoInstance := daoExecution.Get(rdb.T[*model.AppConfigDao]()).Interface()
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.Bind(di.T[*flag.Flag]()).ToInstance(spec.Flag)
			b.BindInstance(daoInstance)
			b.BindInstance(logger.New("vine:test"))
			b.BindInstance(watchServer)
			spec.BindCommon(b)
		},
	)

	configRepo := injector.Get(di.T[core.AppConfigRepo]()).Interface().(core.AppConfigRepo)
	return configRepo
}

func newHubBoundSchemaRepo(t *testing.T, spec *HubApp) core.SchemaRepo {
	t.Helper()

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.New("vine:test"))
			b.Bind(di.T[*flag.Flag]()).ToInstance(spec.Flag)
			b.Bind(di.T[*internalapp.InternalInprocFlag]()).ToInstance(spec.InprocFlag)
			spec.BindCommon(b)
		},
	)

	return injector.Get(di.T[core.SchemaRepo]()).Interface().(core.SchemaRepo)
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
				b.Bind(di.T[*adminimpl.MaintenanceApiServiceServerImpl]()).In(di.SingletonScope)
				b.Bind(di.T[*seeder.Seeder]()).In(di.SingletonScope)
				b.Bind(di.T[*initializer.Initializer]()).In(di.SingletonScope)
			})
			require.False(t, access.ReadOnly())
			module := injector.Get(di.T[*initializer.Initializer]()).Interface().(*initializer.Initializer)
			require.Equal(t, !persistent, access.ReadOnly())
			item, ok := module.AppConfigRepo.GetItemByName("demo.Config")
			require.True(t, ok)
			require.Equal(t, `{"enabled":true}`, item.Value)
			require.NotEmpty(t, module.EntryRepo.ListEntries())
			require.NotEmpty(t, module.RuleRepo.ListRules())
			actions := []struct {
				name string
				call func()
			}{
				{"save config", func() { module.AppConfigRepo.SaveItem(&core.AppConfig{Name: "new.config", Value: "{}", Version: 1}) }},
				{"remove config", func() { module.AppConfigRepo.RemoveItem(item.Id) }},
				{"save site", func() {
					module.EntryRepo.SaveEntry(&core.PortalSite{Name: "new.site", Type: core.PortalSiteTypeWEBGW, WebName: "demo.Web"})
				}},
				{"remove site", func() { module.EntryRepo.RemoveEntry(-1) }},
				{"save rule", func() {
					module.RuleRepo.SaveRule(&core.PortalRule{Name: "new.rule", RouteType: "SITE", RouteSiteName: "new.site"})
				}},
				{"remove rule", func() { module.RuleRepo.RemoveRule(-1) }},
				{"save cert", func() { module.CertRepo.SaveCert(&core.PortalCert{Name: "new.cert", Domains: []string{"demo.local"}}) }},
				{"remove cert", func() { module.CertRepo.RemoveCert(-1) }},
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
			require.NotEmpty(t, module.SchemaRepo.ListDomainSchemaViews())
			service := injector.Get(di.T[*adminimpl.MaintenanceApiServiceServerImpl]()).Interface().(*adminimpl.MaintenanceApiServiceServerImpl)
			require.Equal(t, !persistent, service.ConfigReadOnly())
		})
	}
}
