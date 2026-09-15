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
		DBSQLiteFile: "/tmp/hub.sqlite",
		MQMode:       MQModeEmbedded,
	}

	flags.Normalize(false)

	assert.Equal(t, StoreSQLite, flags.Store)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
}

func TestFlagNormalizeInfersPostgreSQLSourceFromURL(t *testing.T) {
	flags := &Flag{
		DBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
		MQMode:        MQModeEmbedded,
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
		Store:        StoreSQLite,
		DBSQLiteFile: "/tmp/hub.sqlite",
		MQMode:       MQModeEmbedded,
	}

	flags.Normalize(false)

	assert.Equal(t, StoreSQLite, flags.Store)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
	assert.Equal(t, HubDefaultControlListen, flags.ControlListen)
	assert.Equal(t, HubDefaultAdminListen, flags.AdminListen)
	assert.Equal(t, "127.0.0.1:7072", flags.WatchListen)
	assert.Equal(t, HubDefaultDashboardURL, flags.DashboardURL.String())
	assert.False(t, flags.DashboardURLSet)
}

func TestFlagNormalizeNormalizesDashboardURL(t *testing.T) {
	flags := &Flag{
		Store:           StoreSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		MQMode:          MQModeEmbedded,
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
		Store:        StoreSQLite,
		DBSQLiteFile: "/tmp/hub.sqlite",
		MQMode:       MQModeEmbedded,
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
		MQMode:          MQModeEmbedded,
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
		MQMode:          MQModeEmbedded,
		DashboardURLRaw: "https://hub.example.com:8443",
	}

	flags.Normalize(false)

	assert.Equal(t, "https://hub.example.com:8443/", flags.DashboardURL.String())
}

func TestFlagNormalizeRejectsInvalidDashboardURLScheme(t *testing.T) {
	flags := &Flag{
		Store:           StoreSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		MQMode:          MQModeEmbedded,
		DashboardURLRaw: "ftp://hub.example.com:8443/admin",
	}

	require.PanicsWithError(t, "parse DashboardURL failed: scheme must be http or https", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeAcceptsValidMQEndpoint(t *testing.T) {
	flags := &Flag{
		MQMode:         MQModeNATS,
		Store:          StoreSQLite,
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQNatsEndpoint: "nats://127.0.0.1:4222",
	}

	flags.Normalize(false)

	assert.Equal(t, "nats://127.0.0.1:4222", flags.MQNatsEndpoint)
	assert.Equal(t, MQModeNATS, flags.MQMode)
}

func TestFlagNormalizeRejectsInvalidMQEndpoint(t *testing.T) {
	flags := &Flag{
		MQMode:         MQModeNATS,
		Store:          StoreSQLite,
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQNatsEndpoint: "http://127.0.0.1:4222",
	}

	require.PanicsWithError(t, "hub flag normalize failed: MQNatsEndpoint currently only supports nats://", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeMQModes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mode     string
		endpoint string
		want     string
		failure  string
	}{
		{name: "default", want: MQModeEmbedded},
		{name: "embedded", mode: MQModeEmbedded, want: MQModeEmbedded},
		{name: "external", mode: MQModeNATS, endpoint: "nats://localhost:4222", want: MQModeNATS},
		{name: "default rejects endpoint", endpoint: "nats://localhost:4222", failure: "mq-nats-endpoint cannot be used with mq-mode=embedded"},
		{name: "embedded rejects endpoint", mode: MQModeEmbedded, endpoint: "nats://localhost:4222", failure: "mq-nats-endpoint cannot be used with mq-mode=embedded"},
		{name: "external needs endpoint", mode: MQModeNATS, failure: "mq-nats-endpoint is required when mq-mode=nats"},
		{name: "unknown", mode: "unknown", failure: `unsupported MQ mode "unknown"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &Flag{SeedHubData: "{}", MQMode: tc.mode, MQNatsEndpoint: tc.endpoint}
			if tc.failure != "" {
				require.PanicsWithError(t, tc.failure, func() { f.Normalize(false) })
				return
			}
			f.Normalize(false)
			assert.Equal(t, tc.want, f.MQMode)
			assert.Equal(t, tc.endpoint, f.MQNatsEndpoint)
		})
	}
}

func TestFlagNormalizeInprocClearsListenAndMQ(t *testing.T) {
	flags := &Flag{
		MQMode:         MQModeNATS,
		Store:          StoreSQLite,
		DBSQLiteFile:   "/tmp/hub.sqlite",
		ControlListen:  "127.0.0.1:7071",
		AdminListen:    "127.0.0.1:7075",
		WatchListen:    "127.0.0.1:7072",
		MQNatsEndpoint: "nats://127.0.0.1:4222",
		DBPostgresURL:  "",
	}

	flags.Normalize(true)

	assert.Equal(t, StoreSQLite, flags.Store)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
	assert.Empty(t, flags.ControlListen)
	assert.Empty(t, flags.AdminListen)
	assert.Empty(t, flags.WatchListen)
	assert.Empty(t, flags.MQNatsEndpoint)
	assert.Equal(t, MQModeEmbedded, flags.MQMode)
	assert.Equal(t, "embedded", flags.LockMode)
	assert.Empty(t, flags.LockRedisEndpoint)
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

func TestFlagNormalizeLockModes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mode     string
		endpoint string
		want     string
		failure  bool
	}{
		{name: "default", want: "embedded"},
		{name: "embedded", mode: "embedded", want: "embedded"},
		{name: "redis", mode: "redis", endpoint: "redis://user:pass@localhost:6379/2", want: "redis"},
		{name: "redis TLS", mode: "redis", endpoint: "rediss://localhost:6379", want: "redis"},
		{name: "disable", mode: "disable", want: "disable"},
		{name: "missing endpoint", mode: "redis", failure: true},
		{name: "default rejects endpoint", endpoint: "redis://localhost", failure: true},
		{name: "embedded rejects endpoint", mode: "embedded", endpoint: "redis://localhost", failure: true},
		{name: "disable rejects endpoint", mode: "disable", endpoint: "redis://localhost", failure: true},
		{name: "unsupported mode", mode: "disabled", failure: true},
		{name: "wrong scheme", mode: "redis", endpoint: "http://localhost:6379", failure: true},
		{name: "missing host", mode: "redis", endpoint: "redis://", failure: true},
		{name: "invalid DB", mode: "redis", endpoint: "redis://localhost/not-a-db", failure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &Flag{SeedHubData: "{}", LockMode: tc.mode, LockRedisEndpoint: tc.endpoint}
			if tc.failure {
				require.Panics(t, func() { f.Normalize(false) })
				return
			}
			f.Normalize(false)
			assert.Equal(t, tc.want, f.LockMode)
			assert.Equal(t, tc.endpoint, f.LockRedisEndpoint)
		})
	}
}

func TestFlagNormalizeInprocUsesEmbeddedLock(t *testing.T) {
	f := &Flag{SeedHubData: "{}", LockMode: "redis", LockRedisEndpoint: "redis://localhost:6379"}
	f.Normalize(true)
	assert.Equal(t, "embedded", f.LockMode)
	assert.Empty(t, f.LockRedisEndpoint)
}
