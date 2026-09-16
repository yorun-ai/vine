package flag

import (
	"fmt"
	"net/url"

	goredis "github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/mtls"
	hublock "go.yorun.ai/vine/internal/daemon/hub/api/lock"
	"go.yorun.ai/vine/util/vnet"
	"go.yorun.ai/vine/util/vpre"
)

const (
	HubDefaultControlListen = "127.0.0.1:7071"
	HubDefaultWatchListen   = "127.0.0.1:7072"
	// HubDefaultAdminListen is the Dashboard port operators already know: the
	// Dashboard reached Hub through Portal on 7099, and Hub serves it itself now.
	HubDefaultAdminListen = "127.0.0.1:7099"

	MQModeEmbedded = "embedded"
	MQModeNATS     = "nats"

	StoreMemory     = "memory"
	StoreSQLite     = "sqlite"
	StorePostgreSQL = "postgres"
)

type Flag struct {
	app.FlagModel
	MTLS mtls.Files

	ControlListen string
	AdminListen   string
	WatchListen   string

	MQMode         string
	MQNatsEndpoint string

	LockMode          string
	LockRedisEndpoint string

	Store         string
	NoDB          bool
	DBSQLiteFile  string
	DBPostgresURL string

	SeedHubData       string
	SeedHubDataFile   string
	SeedHubSource     string
	SeedHubSourceFile string
	SeedHubVarsFile   string
}

func (f *Flag) Normalize(inproc bool) {
	vpre.CheckNilError(f.MTLS.Validate(), "hub flag normalize failed")
	f.normalizeSeed()
	f.normalizeStore()

	if inproc {
		// Inproc hub is reached through rpc+inproc and uses in-process NATS,
		// so external listen addresses and MQ endpoint must not leak into runtime info.
		f.ControlListen = ""
		f.WatchListen = ""
		f.MQNatsEndpoint = ""
		f.MQMode = MQModeEmbedded
		f.LockMode = hublock.ModeEmbedded
		f.LockRedisEndpoint = ""
		// AdminListen stays as declared: standalone opens that listener on request.
		return
	}

	f.normalizeListen()
	f.normalizeMQ()
	f.normalizeLock()
}

func (f *Flag) normalizeListen() {
	if f.ControlListen == "" {
		f.ControlListen = HubDefaultControlListen
	}
	if f.AdminListen == "" {
		f.AdminListen = HubDefaultAdminListen
	}
	if f.WatchListen == "" {
		f.WatchListen = HubDefaultWatchListen
	}
}

func (f *Flag) normalizeSeed() {
	vpre.CheckNot(f.SeedHubSource != "" && f.SeedHubSourceFile != "", "SeedHubSource and the seed source file are mutually exclusive")
	vpre.CheckNot((f.SeedHubSource != "" || f.SeedHubSourceFile != "" || f.SeedHubVarsFile != "") && f.SeedHubData == "" && f.SeedHubDataFile == "", "seed source and variables require seed YAML")
	vpre.CheckNot(f.SeedHubDataFile != "" && f.SeedHubData != "", "SeedHubData and the seed data file are mutually exclusive")
	vpre.CheckNot(f.SeedHubSource != "" && f.SeedHubData == "", "SeedHubSource requires inline SeedHubData")
	vpre.CheckNot(f.SeedHubSourceFile != "" && f.SeedHubDataFile == "", "a seed source file requires a seed data file")
}

func (f *Flag) normalizeStore() {
	vpre.CheckNot(f.NoDB && (f.DBSQLiteFile != "" || f.DBPostgresURL != "" || (f.Store != "" && f.Store != StoreMemory)), "no-db cannot be used with a database store")
	kind := f.Store
	if kind == "" {
		var err error
		kind, err = f.inferStore()
		vpre.CheckNilError(err, "hub flag normalize failed")
	}
	f.Store = kind

	switch kind {
	case StoreMemory:
		f.NoDB = true
		vpre.Check(f.SeedHubDataFile != "" || f.SeedHubData != "", "no-db requires a seed data file or SeedHubData")
	case StoreSQLite:
		vpre.CheckNotEmpty(f.DBSQLiteFile, "DBSQLiteFile is empty")
	case StorePostgreSQL:
		vpre.CheckNotEmpty(f.DBPostgresURL, "DBPostgresURL is empty")
	default:
		vpre.Panicf("unsupported hub store %q", kind)
	}
}

func (f *Flag) normalizeMQ() {
	if f.MQMode == "" {
		f.MQMode = MQModeEmbedded
	}
	switch f.MQMode {
	case MQModeEmbedded:
		vpre.Check(f.MQNatsEndpoint == "", "mq-nats-endpoint cannot be used with mq-mode=embedded")
	case MQModeNATS:
		vpre.CheckNotEmpty(f.MQNatsEndpoint, "mq-nats-endpoint is required when mq-mode=nats")
		vpre.CheckNilError(validateMQNatsEndpoint(f.MQNatsEndpoint), "hub flag normalize failed")
	default:
		vpre.Panicf("unsupported MQ mode %q", f.MQMode)
	}
}

// ControlPort returns the component-facing Control API port.
func (f *Flag) normalizeLock() {
	if f.LockMode == "" {
		f.LockMode = hublock.ModeEmbedded
	}
	switch f.LockMode {
	case hublock.ModeEmbedded, hublock.ModeDisable:
		vpre.Check(f.LockRedisEndpoint == "", "lock-redis-endpoint cannot be used with lock-mode=%s", f.LockMode)
	case hublock.ModeRedis:
		vpre.CheckNotEmpty(f.LockRedisEndpoint, "lock-redis-endpoint is required when lock-mode=redis")
		endpoint, err := url.Parse(f.LockRedisEndpoint)
		vpre.Check(err == nil, "invalid lock-redis-endpoint")
		vpre.Check(endpoint.Scheme == "redis" || endpoint.Scheme == "rediss", "lock-redis-endpoint must use redis:// or rediss://")
		vpre.CheckNotEmpty(endpoint.Hostname(), "lock-redis-endpoint host is empty")
		_, err = goredis.ParseURL(f.LockRedisEndpoint)
		vpre.Check(err == nil, "invalid lock-redis-endpoint")
	default:
		vpre.Panicf("unsupported lock mode %q", f.LockMode)
	}
}

func (f *Flag) ControlPort() int {
	return vnet.MustParsePort(f.ControlListen)
}

func (f *Flag) WatchPort() int {
	return vnet.MustParsePort(f.WatchListen)
}

func validateMQNatsEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("MQNatsEndpoint is invalid: %w", err)
	}
	if parsed.Scheme != "nats" {
		return fmt.Errorf("MQNatsEndpoint currently only supports nats://")
	}
	if parsed.Host == "" {
		return fmt.Errorf("MQNatsEndpoint host is empty")
	}
	return nil
}

func (f *Flag) inferStore() (string, error) {
	hasSQLite := f.DBSQLiteFile != ""
	hasPostgreSQL := f.DBPostgresURL != ""

	storeCount := 0
	if hasSQLite {
		storeCount++
	}
	if hasPostgreSQL {
		storeCount++
	}
	if storeCount > 1 {
		return "", fmt.Errorf("only one of DBSQLiteFile or DBPostgresURL can be set")
	}

	switch {
	case hasSQLite:
		return StoreSQLite, nil
	case hasPostgreSQL:
		return StorePostgreSQL, nil
	default:
		return StoreMemory, nil
	}
}
