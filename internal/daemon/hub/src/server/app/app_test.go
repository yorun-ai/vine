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
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
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

func initTestConfigDatabase(component *repodb.HubDatabase) *rdb.DatabaseMinder {
	minder := new(rdb.DatabaseMinder)
	minder.InitComponent(component)
	return minder
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
		InternalApplication: internalapp.InternalApplication{
			Application: internalapp.Application{
				AppFlag: &internalapp.RunFlag{},
			},
		},
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			SourceType:        flag.SourceSQLite,
			DBSQLiteFile:      "/tmp/hub.sqlite",
			MQExternalNatsURL: "nats://127.0.0.1:4222",
		},
	}

	spec.DIInit()

	assert.Equal(t, flag.HubDefaultAPIListen, spec.AppFlag.ListenAddr)
	assert.Equal(t, flag.SourceSQLite, spec.Flag.SourceType)
	assert.Equal(t, flag.HubDefaultAPIListen, spec.Flag.APIListen)
	assert.Equal(t, flag.HubDefaultRedisListen, spec.Flag.RedisListen)
	assert.Equal(t, "/tmp/hub.sqlite", spec.Flag.DBSQLiteFile)
	assert.Equal(t, "nats://127.0.0.1:4222", spec.Flag.MQExternalNatsURL)
	assert.False(t, spec.Flag.MQEmbeddedNats)
}

func TestHubAppDIInitKeepsPGConnUrl(t *testing.T) {
	spec := &HubApp{
		InternalApplication: internalapp.InternalApplication{
			Application: internalapp.Application{
				AppFlag: &internalapp.RunFlag{},
			},
		},
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			SourceType:        flag.SourcePostgreSQL,
			DBPostgresURL:     "postgres://demo:demo@127.0.0.1:5432/hub",
			MQExternalNatsURL: "nats://127.0.0.1:4222",
		},
	}

	spec.DIInit()

	assert.Equal(t, flag.SourcePostgreSQL, spec.Flag.SourceType)
	assert.Equal(t, "postgres://demo:demo@127.0.0.1:5432/hub", spec.Flag.DBPostgresURL)
	assert.Equal(t, "nats://127.0.0.1:4222", spec.Flag.MQExternalNatsURL)
	assert.False(t, spec.Flag.MQEmbeddedNats)
	assert.Equal(t, flag.HubDefaultAPIListen, spec.AppFlag.ListenAddr)
}

func TestHubAppDIInitAppendsRuntimeNameToInfoInInprocMode(t *testing.T) {
	spec := &HubApp{
		InternalApplication: internalapp.InternalApplication{
			Application: internalapp.Application{
				AppFlag: &internalapp.RunFlag{},
			},
		},
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag: &flag.Flag{
			SourceType:     flag.SourceSQLite,
			DBSQLiteFile:   "/tmp/hub.sqlite",
			MQEmbeddedNats: true,
		},
	}

	spec.DIInit()

	assert.Equal(t, "vine.hub", spec.Name())
	assert.Equal(t, "vine.hub@"+runtime.Application().Name(), spec.InternalAttrs.Info.Name())
	assert.Empty(t, spec.AppFlag.ListenAddr)
	assert.Empty(t, spec.Flag.APIListen)
	assert.Empty(t, spec.Flag.RedisListen)
	assert.True(t, spec.Flag.MQEmbeddedNats)
}

func TestHubAppDIInitKeepsEnableNatsOutsideInproc(t *testing.T) {
	spec := &HubApp{
		InternalApplication: internalapp.InternalApplication{
			Application: internalapp.Application{
				AppFlag: &internalapp.RunFlag{},
			},
		},
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			SourceType:     flag.SourceSQLite,
			DBSQLiteFile:   "/tmp/hub.sqlite",
			MQEmbeddedNats: true,
		},
	}

	spec.DIInit()

	assert.True(t, spec.Flag.MQEmbeddedNats)
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
	}, collectModuleTypes(spec))
}

func TestHubAppModuleTypesIncludesRuntimeModulesWhenEnableNats(t *testing.T) {
	spec := &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag:       &flag.Flag{MQEmbeddedNats: true},
	}

	assert.Equal(t, []reflect.Type{
		internalapp.T[*syncer.Syncer](),
		internalapp.T[*seeder.Seeder](),
		internalapp.T[*initializer.Initializer](),
		internalapp.T[*scheduler.Scheduler](),
		internalapp.T[*sweeper.Sweeper](),
	}, collectModuleTypes(spec))
}

func TestHubAppComponentTypesReturnsSQLiteDatabaseWhenSourceIsSQLite(t *testing.T) {
	spec := &HubApp{
		Flag: &flag.Flag{SourceType: flag.SourceSQLite},
	}

	assert.Equal(t, []reflect.Type{
		internalapp.T[*repodb.HubDatabase](),
		internalapp.T[*natsserver.NATSServer](),
		internalapp.T[*redisserver.Server](),
	}, collectComponentTypes(spec))
}

func TestHubAppComponentTypesReturnsPGDatabaseWhenSourceIsPG(t *testing.T) {
	spec := &HubApp{
		Flag: &flag.Flag{SourceType: flag.SourcePostgreSQL},
	}

	assert.Equal(t, []reflect.Type{
		internalapp.T[*repodb.HubDatabase](),
		internalapp.T[*natsserver.NATSServer](),
		internalapp.T[*redisserver.Server](),
	}, collectComponentTypes(spec))
}

func TestConfigDatabaseInitOptionForSQLite(t *testing.T) {
	spec := &repodb.HubDatabase{
		Flag: &flag.Flag{
			SourceType:   flag.SourceSQLite,
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
			SourceType:    flag.SourcePostgreSQL,
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
		Flag:       &flag.Flag{SourceType: flag.SourceSQLite},
	})

	assert.IsType(t, &repo.DBAppConfigRepo{}, configRepo)
}

func TestHubAppBindCommonProvidesDBAppConfigRepoForPG(t *testing.T) {
	configRepo := newHubBoundAppConfigRepo(t, &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			SourceType:    flag.SourcePostgreSQL,
			DBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
		},
	})

	assert.IsType(t, &repo.DBAppConfigRepo{}, configRepo)
}

func TestHubAppBindCommonProvidesMemorySchemaRepoForDBInInprocMode(t *testing.T) {
	schemaRepo := newHubBoundSchemaRepo(t, &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag:       &flag.Flag{SourceType: flag.SourceSQLite},
	})

	assert.IsType(t, &schema.MemorySchemaRepo{}, schemaRepo)
}

func TestHubAppBindCommonProvidesMemorySchemaRepoForDB(t *testing.T) {
	schemaRepo := newHubBoundSchemaRepo(t, &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag:       &flag.Flag{SourceType: flag.SourceSQLite},
	})

	assert.IsType(t, &schema.MemorySchemaRepo{}, schemaRepo)
}

func testDashboardURL() *vnet.HttpURL {
	return vnet.MustParseHttpURL(flag.HubDefaultDashboardURL)
}

func TestHubAppBindCommonProvidesDBAppConfigRepoForInitializerWithSQLite(t *testing.T) {
	component := &repodb.HubDatabase{
		Flag: &flag.Flag{
			SourceType:   flag.SourceSQLite,
			DBSQLiteFile: sharedTestSQLitePath(t),
		},
	}
	minder := initTestConfigDatabase(component)
	t.Cleanup(minder.AfterAppStop)
	redisServer := redisserver.NewServerForTest()
	t.Cleanup(redisServer.AfterAppStop)

	spec := &HubApp{
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag: &flag.Flag{
			SourceType:   flag.SourceSQLite,
			APIListen:    flag.HubDefaultAPIListen,
			DashboardURL: testDashboardURL(),
			DBSQLiteFile: sharedTestSQLitePath(t),
			RedisListen:  flag.HubDefaultRedisListen,
		},
	}

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.NewLogger(logger.GlobalOption()))
			minder.Bind(b)
			b.BindInstance(redisServer)
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
	assert.IsType(t, &repo.DBAppConfigRepo{}, module.AppConfigRepo)
}

func TestConfigDatabaseBindProvidesAppConfigRepoDAO(t *testing.T) {
	component := &repodb.HubDatabase{
		Flag: &flag.Flag{
			SourceType:   flag.SourceSQLite,
			DBSQLiteFile: sharedTestSQLitePath(t),
		},
	}
	minder := initTestConfigDatabase(component)
	t.Cleanup(minder.AfterAppStop)

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.NewLogger(logger.GlobalOption()))
			minder.Bind(b)
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
			SourceType:   flag.SourceSQLite,
			DBSQLiteFile: sharedTestSQLitePath(t),
		},
	}
	minder := initTestConfigDatabase(component)
	t.Cleanup(minder.AfterAppStop)

	daoInjector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.NewLogger(logger.GlobalOption()))
			minder.Bind(b)
		},
	)
	daoExecution := daoInjector.StartExecution()
	defer daoExecution.CompleteExecution()
	daoInstance := daoExecution.Get(rdb.T[*model.AppConfigDao]()).Interface()
	redisServer := redisserver.NewServerForTest()
	t.Cleanup(redisServer.AfterAppStop)

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(di.T[context.Context]()).ToInstance(context.Background())
			b.Bind(di.T[*flag.Flag]()).ToInstance(spec.Flag)
			b.BindInstance(daoInstance)
			b.BindInstance(logger.NewLogger(logger.GlobalOption()))
			b.BindInstance(redisServer)
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
			b.BindInstance(logger.NewLogger(logger.GlobalOption()))
			b.Bind(di.T[*flag.Flag]()).ToInstance(spec.Flag)
			b.Bind(di.T[*internalapp.InternalInprocFlag]()).ToInstance(spec.InprocFlag)
			spec.BindCommon(b)
		},
	)

	return injector.Get(di.T[core.SchemaRepo]()).Interface().(core.SchemaRepo)
}
