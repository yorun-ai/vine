package flag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/mtls"
)

func TestNormalizeRejectsPartialMTLSFiles(t *testing.T) {
	flags := Flag{MTLS: mtls.Files{CAFile: "ca.pem"}}
	require.PanicsWithError(t, "hub flag normalize failed: mtls-ca-file, mtls-cert-file, and mtls-key-file must be configured together", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeRequiresSeedWithoutDatabase(t *testing.T) {
	flags := &Flag{}

	require.PanicsWithError(t, "no-db requires seed-hub-data-file or SeedHubData", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeInfersSQLiteSourceFromPath(t *testing.T) {
	flags := &Flag{
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.Equal(t, StoreSQLite, flags.Store)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
}

func TestFlagNormalizeInfersPostgreSQLSourceFromURL(t *testing.T) {
	flags := &Flag{
		DBPostgresURL:  "postgres://demo:demo@127.0.0.1:5432/hub",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.Equal(t, StorePostgreSQL, flags.Store)
	assert.Equal(t, "postgres://demo:demo@127.0.0.1:5432/hub", flags.DBPostgresURL)
}

func TestFlagNormalizeRejectsMultipleStores(t *testing.T) {
	flags := &Flag{
		DBSQLiteFile:  "/tmp/hub.sqlite",
		DBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
	}

	require.PanicsWithError(t, "hub flag normalize failed: only one of DBSQLiteFile or DBPostgresURL can be set", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeKeepsExplicitStore(t *testing.T) {
	flags := &Flag{
		Store:          StoreSQLite,
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.Equal(t, StoreSQLite, flags.Store)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
	assert.Equal(t, HubDefaultControlListen, flags.ControlListen)
	assert.Equal(t, HubDefaultAdminListen, flags.AdminListen)
	assert.Equal(t, "127.0.0.1:7072", flags.RedisListen)
	assert.Equal(t, HubDefaultDashboardURL, flags.DashboardURL.String())
	assert.False(t, flags.DashboardURLSet)
}

func TestFlagNormalizeNormalizesDashboardURL(t *testing.T) {
	flags := &Flag{
		Store:           StoreSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		MQEmbeddedNats:  true,
		DashboardURLRaw: ":7099",
	}

	flags.Normalize(false)

	assert.Equal(t, HubDefaultDashboardURL, flags.DashboardURL.String())
	assert.True(t, flags.DashboardURLSet)
}

func TestFlagNormalizeUsesHTTPSDashboardDefaultWithMTLS(t *testing.T) {
	flags := &Flag{
		MTLS: mtls.Files{
			CAFile:   "ca.pem",
			CertFile: "cert.pem",
			KeyFile:  "key.pem",
		},
		Store:          StoreSQLite,
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.Equal(t, HubMTLSDefaultDashboardURL, flags.DashboardURL.String())
	assert.False(t, flags.DashboardURLSet)
	assert.True(t, flags.DashboardURLMTLSDefault)
}

func TestFlagNormalizeKeepsExplicitHTTPDashboardURLWithMTLS(t *testing.T) {
	flags := &Flag{
		MTLS: mtls.Files{
			CAFile:   "ca.pem",
			CertFile: "cert.pem",
			KeyFile:  "key.pem",
		},
		Store:           StoreSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		MQEmbeddedNats:  true,
		DashboardURLRaw: "http://:7099/",
	}

	flags.Normalize(false)

	assert.Equal(t, HubDefaultDashboardURL, flags.DashboardURL.String())
	assert.True(t, flags.DashboardURLSet)
	assert.False(t, flags.DashboardURLMTLSDefault)
}

func TestFlagNormalizeAddsDashboardURLPath(t *testing.T) {
	flags := &Flag{
		Store:           StoreSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		MQEmbeddedNats:  true,
		DashboardURLRaw: "https://hub.example.com:8443",
	}

	flags.Normalize(false)

	assert.Equal(t, "https://hub.example.com:8443/", flags.DashboardURL.String())
}

func TestFlagNormalizeRejectsInvalidDashboardURLScheme(t *testing.T) {
	flags := &Flag{
		Store:           StoreSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		MQEmbeddedNats:  true,
		DashboardURLRaw: "ftp://hub.example.com:8443/admin",
	}

	require.PanicsWithError(t, "parse DashboardURL failed: scheme must be http or https", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeAcceptsValidMQEndpoint(t *testing.T) {
	flags := &Flag{
		Store:             StoreSQLite,
		DBSQLiteFile:      "/tmp/hub.sqlite",
		MQExternalNatsURL: "nats://127.0.0.1:4222",
	}

	flags.Normalize(false)

	assert.Equal(t, "nats://127.0.0.1:4222", flags.MQExternalNatsURL)
	assert.False(t, flags.MQEmbeddedNats)
}

func TestFlagNormalizeRejectsMQEndpointWithEnableNats(t *testing.T) {
	flags := &Flag{
		Store:             StoreSQLite,
		DBSQLiteFile:      "/tmp/hub.sqlite",
		MQExternalNatsURL: "nats://127.0.0.1:4222",
		MQEmbeddedNats:    true,
	}

	require.PanicsWithError(t, "exactly one of MQExternalNatsURL or MQEmbeddedNats must be set", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeRejectsInvalidMQEndpoint(t *testing.T) {
	flags := &Flag{
		Store:             StoreSQLite,
		DBSQLiteFile:      "/tmp/hub.sqlite",
		MQExternalNatsURL: "http://127.0.0.1:4222",
	}

	require.PanicsWithError(t, "hub flag normalize failed: MQExternalNatsURL currently only supports nats://", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeRequiresMQEndpointOrEnableNats(t *testing.T) {
	flags := &Flag{
		Store:        StoreSQLite,
		DBSQLiteFile: "/tmp/hub.sqlite",
	}

	require.PanicsWithError(t, "exactly one of MQExternalNatsURL or MQEmbeddedNats must be set", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeAcceptsEnableNats(t *testing.T) {
	flags := &Flag{
		Store:          StoreSQLite,
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.True(t, flags.MQEmbeddedNats)
}

func TestFlagNormalizeInprocClearsListenAndMQ(t *testing.T) {
	flags := &Flag{
		Store:             StoreSQLite,
		DBSQLiteFile:      "/tmp/hub.sqlite",
		ControlListen:     "127.0.0.1:7071",
		AdminListen:       "127.0.0.1:7075",
		RedisListen:       "127.0.0.1:7072",
		MQExternalNatsURL: "nats://127.0.0.1:4222",
		DBPostgresURL:     "",
	}

	flags.Normalize(true)

	assert.Equal(t, StoreSQLite, flags.Store)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
	assert.Empty(t, flags.ControlListen)
	assert.Empty(t, flags.AdminListen)
	assert.Empty(t, flags.RedisListen)
	assert.Empty(t, flags.MQExternalNatsURL)
	assert.True(t, flags.MQEmbeddedNats)
}

func TestFlagInferStoreDefaultsToMemory(t *testing.T) {
	flags := &Flag{}

	store, err := flags.inferStore()
	require.NoError(t, err)

	assert.Equal(t, StoreMemory, store)
}

func TestFlagInferStoreReturnsSQLite(t *testing.T) {
	flags := &Flag{
		DBSQLiteFile: "/tmp/hub.sqlite",
	}

	store, err := flags.inferStore()
	require.NoError(t, err)

	assert.Equal(t, StoreSQLite, store)
}

func TestFlagInferStoreRejectsMultipleStores(t *testing.T) {
	flags := &Flag{
		DBSQLiteFile:  "/tmp/hub.sqlite",
		DBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
	}

	store, err := flags.inferStore()
	require.EqualError(t, err, "only one of DBSQLiteFile or DBPostgresURL can be set")
	assert.Empty(t, store)
}

func TestNoDBModes(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		f := &Flag{NoDB: explicit, SeedHubDataFile: "seed.yaml"}
		f.Normalize(true)
		require.True(t, f.NoDB)
		require.Equal(t, StoreMemory, f.Store)
	}
	for _, f := range []*Flag{{NoDB: true, DBSQLiteFile: "hub.sqlite"}, {NoDB: true, DBPostgresURL: "postgres://localhost/hub"}} {
		require.PanicsWithError(t, "no-db cannot be used with a database store", func() { f.Normalize(true) })
	}
}

func TestInlineSeedSourceModes(t *testing.T) {
	for _, flags := range []*Flag{
		{SeedHubData: "{}"},
		{SeedHubData: "{}", NoDB: true},
		{SeedHubData: "{}", DBSQLiteFile: "hub.sqlite"},
		{SeedHubData: "{}", DBPostgresURL: "postgres://localhost/hub"},
	} {
		require.NotPanics(t, func() { flags.Normalize(true) })
		require.Equal(t, flags.DBSQLiteFile == "" && flags.DBPostgresURL == "", flags.NoDB)
	}
}

func TestSeedSupplementInputsRequireTemplateAndAreExclusive(t *testing.T) {
	for _, flags := range []*Flag{
		{SeedHubData: "{}", SeedHubSource: "{}", SeedHubSourceFile: "source.yaml"},
		{SeedHubData: "{}", SeedHubSourceFile: "source.yaml"},
		{SeedHubDataFile: "seed.yaml", SeedHubSource: "{}"},
		{DBSQLiteFile: "hub.sqlite", SeedHubSourceFile: "source.yaml"},
		{DBSQLiteFile: "hub.sqlite", SeedHubVarsFile: "vars.yaml"},
	} {
		require.Panics(t, func() { flags.Normalize(true) })
	}
	for _, valid := range []*Flag{
		{SeedHubData: "{}", SeedHubSource: "{}", SeedHubVarsFile: "vars.yaml"},
		{SeedHubDataFile: "seed.yaml", SeedHubSourceFile: "source.yaml", SeedHubVarsFile: "vars.yaml"},
		{SeedHubData: "{}", SeedHubVarsFile: "vars.yaml"},
		{SeedHubDataFile: "seed.yaml", SeedHubVarsFile: "vars.yaml"},
	} {
		require.NotPanics(t, func() { valid.Normalize(true) })
	}
}
