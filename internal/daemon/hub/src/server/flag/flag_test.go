package flag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagNormalizeRequiresSource(t *testing.T) {
	flags := &Flag{}

	require.PanicsWithError(t, "hub flag normalize failed: one of DBSQLiteFile or DBPostgresURL must be set", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeInfersSQLiteSourceFromPath(t *testing.T) {
	flags := &Flag{
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.Equal(t, SourceSQLite, flags.SourceType)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
}

func TestFlagNormalizeInfersPostgreSQLSourceFromURL(t *testing.T) {
	flags := &Flag{
		DBPostgresURL:  "postgres://demo:demo@127.0.0.1:5432/hub",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.Equal(t, SourcePostgreSQL, flags.SourceType)
	assert.Equal(t, "postgres://demo:demo@127.0.0.1:5432/hub", flags.DBPostgresURL)
}

func TestFlagNormalizeRejectsMultipleSources(t *testing.T) {
	flags := &Flag{
		DBSQLiteFile:  "/tmp/hub.sqlite",
		DBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
	}

	require.PanicsWithError(t, "hub flag normalize failed: only one of DBSQLiteFile or DBPostgresURL can be set", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeKeepsExplicitSourceType(t *testing.T) {
	flags := &Flag{
		SourceType:     SourceSQLite,
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.Equal(t, SourceSQLite, flags.SourceType)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
	assert.Equal(t, "127.0.0.1:7071", flags.APIListen)
	assert.Equal(t, "127.0.0.1:7073", flags.RedisListen)
	assert.Equal(t, HubDefaultDashboardURL, flags.DashboardURL.String())
	assert.False(t, flags.DashboardURLSet)
}

func TestFlagNormalizeNormalizesDashboardURL(t *testing.T) {
	flags := &Flag{
		SourceType:      SourceSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		MQEmbeddedNats:  true,
		DashboardURLRaw: ":7099",
	}

	flags.Normalize(false)

	assert.Equal(t, HubDefaultDashboardURL, flags.DashboardURL.String())
	assert.True(t, flags.DashboardURLSet)
}

func TestFlagNormalizeAddsDashboardURLPath(t *testing.T) {
	flags := &Flag{
		SourceType:      SourceSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		MQEmbeddedNats:  true,
		DashboardURLRaw: "https://hub.example.com:8443",
	}

	flags.Normalize(false)

	assert.Equal(t, "https://hub.example.com:8443/", flags.DashboardURL.String())
}

func TestFlagNormalizeRejectsInvalidDashboardURLScheme(t *testing.T) {
	flags := &Flag{
		SourceType:      SourceSQLite,
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
		SourceType:        SourceSQLite,
		DBSQLiteFile:      "/tmp/hub.sqlite",
		MQExternalNatsURL: "nats://127.0.0.1:4222",
	}

	flags.Normalize(false)

	assert.Equal(t, "nats://127.0.0.1:4222", flags.MQExternalNatsURL)
	assert.False(t, flags.MQEmbeddedNats)
}

func TestFlagNormalizeRejectsMQEndpointWithEnableNats(t *testing.T) {
	flags := &Flag{
		SourceType:        SourceSQLite,
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
		SourceType:        SourceSQLite,
		DBSQLiteFile:      "/tmp/hub.sqlite",
		MQExternalNatsURL: "http://127.0.0.1:4222",
	}

	require.PanicsWithError(t, "hub flag normalize failed: MQExternalNatsURL currently only supports nats://", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeRequiresMQEndpointOrEnableNats(t *testing.T) {
	flags := &Flag{
		SourceType:   SourceSQLite,
		DBSQLiteFile: "/tmp/hub.sqlite",
	}

	require.PanicsWithError(t, "exactly one of MQExternalNatsURL or MQEmbeddedNats must be set", func() {
		flags.Normalize(false)
	})
}

func TestFlagNormalizeAcceptsEnableNats(t *testing.T) {
	flags := &Flag{
		SourceType:     SourceSQLite,
		DBSQLiteFile:   "/tmp/hub.sqlite",
		MQEmbeddedNats: true,
	}

	flags.Normalize(false)

	assert.True(t, flags.MQEmbeddedNats)
}

func TestFlagNormalizeInprocClearsListenAndMQ(t *testing.T) {
	flags := &Flag{
		SourceType:        SourceSQLite,
		DBSQLiteFile:      "/tmp/hub.sqlite",
		APIListen:         "127.0.0.1:7071",
		RedisListen:       "127.0.0.1:7073",
		MQExternalNatsURL: "nats://127.0.0.1:4222",
		DBPostgresURL:     "",
	}

	flags.Normalize(true)

	assert.Equal(t, SourceSQLite, flags.SourceType)
	assert.Equal(t, "/tmp/hub.sqlite", flags.DBSQLiteFile)
	assert.Empty(t, flags.APIListen)
	assert.Empty(t, flags.RedisListen)
	assert.Empty(t, flags.MQExternalNatsURL)
	assert.True(t, flags.MQEmbeddedNats)
}

func TestFlagInferSourceTypeRequiresSource(t *testing.T) {
	flags := &Flag{}

	sourceType, err := flags.inferSourceType()
	require.EqualError(t, err, "one of DBSQLiteFile or DBPostgresURL must be set")

	assert.Empty(t, sourceType)
}

func TestFlagInferSourceTypeReturnsSQLite(t *testing.T) {
	flags := &Flag{
		DBSQLiteFile: "/tmp/hub.sqlite",
	}

	sourceType, err := flags.inferSourceType()
	require.NoError(t, err)

	assert.Equal(t, SourceSQLite, sourceType)
}

func TestFlagInferSourceTypeRejectsMultipleSources(t *testing.T) {
	flags := &Flag{
		DBSQLiteFile:  "/tmp/hub.sqlite",
		DBPostgresURL: "postgres://demo:demo@127.0.0.1:5432/hub",
	}

	sourceType, err := flags.inferSourceType()
	require.EqualError(t, err, "only one of DBSQLiteFile or DBPostgresURL can be set")
	assert.Empty(t, sourceType)
}
