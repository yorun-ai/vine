package flag

import (
	"fmt"
	"net/url"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/util/vnet"
	"go.yorun.ai/vine/util/vpre"
)

const (
	HubDefaultControlListen    = "127.0.0.1:7071"
	HubDefaultRedisListen      = "127.0.0.1:7072"
	HubDefaultAdminListen      = "127.0.0.1:7075"
	HubDefaultDashboardURL     = "http://:7099/"
	HubMTLSDefaultDashboardURL = "https://:7099/"

	StoreMemory     = "memory"
	StoreSQLite     = "sqlite"
	StorePostgreSQL = "postgres"
)

type Flag struct {
	app.FlagModel
	MTLS mtls.Files

	ControlListen string
	AdminListen   string
	RedisListen   string

	MQExternalNatsURL string
	MQEmbeddedNats    bool

	Store         string
	NoDB          bool
	DBSQLiteFile  string
	DBPostgresURL string

	SeedYAML       string
	SeedYAMLPath   string
	SeedSource     string
	SeedSourceFile string
	SeedVarsFile   string

	DashboardURLRaw         string
	DashboardURLSet         bool
	DashboardURLMTLSDefault bool
	DashboardURL            *vnet.HttpURL
}

func (f *Flag) Normalize(inproc bool) {
	vpre.CheckNilError(f.MTLS.Validate(), "hub flag normalize failed")
	f.normalizeSeed()
	f.normalizeStore()
	f.normalizeDashboardURL()

	if inproc {
		// Inproc hub is reached through rpc+inproc and uses in-process NATS,
		// so external listen addresses and MQ endpoint must not leak into runtime info.
		f.ControlListen = ""
		f.AdminListen = ""
		f.RedisListen = ""
		f.MQExternalNatsURL = ""
		f.MQEmbeddedNats = true
		return
	}

	f.normalizeListen()
	f.normalizeMQ()
}

func (f *Flag) normalizeListen() {
	if f.ControlListen == "" {
		f.ControlListen = HubDefaultControlListen
	}
	if f.AdminListen == "" {
		f.AdminListen = HubDefaultAdminListen
	}
	if f.RedisListen == "" {
		f.RedisListen = HubDefaultRedisListen
	}
}

func (f *Flag) normalizeSeed() {
	vpre.CheckNot(f.SeedSource != "" && f.SeedSourceFile != "", "SeedSource and seed-source-file are mutually exclusive")
	vpre.CheckNot((f.SeedSource != "" || f.SeedSourceFile != "" || f.SeedVarsFile != "") && f.SeedYAML == "" && f.SeedYAMLPath == "", "seed source and variables require seed YAML")
	vpre.CheckNot(f.SeedYAMLPath != "" && f.SeedYAML != "", "SeedYAML and seed-yaml-file are mutually exclusive")
	vpre.CheckNot(f.SeedSource != "" && f.SeedYAML == "", "SeedSource requires inline SeedYAML")
	vpre.CheckNot(f.SeedSourceFile != "" && f.SeedYAMLPath == "", "seed-source-file requires seed-yaml-file")
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
		vpre.Check(f.SeedYAMLPath != "" || f.SeedYAML != "", "no-db requires seed-yaml-file or SeedYAML")
	case StoreSQLite:
		vpre.CheckNotEmpty(f.DBSQLiteFile, "DBSQLiteFile is empty")
	case StorePostgreSQL:
		vpre.CheckNotEmpty(f.DBPostgresURL, "DBPostgresURL is empty")
	default:
		vpre.Panicf("unsupported hub store %q", kind)
	}
}

func (f *Flag) normalizeMQ() {
	if (f.MQExternalNatsURL != "") == f.MQEmbeddedNats {
		vpre.Panicf("exactly one of MQExternalNatsURL or MQEmbeddedNats must be set")
	}
	if f.MQExternalNatsURL != "" {
		vpre.CheckNilError(validateMQExternalNatsURL(f.MQExternalNatsURL), "hub flag normalize failed")
	}
}

// ControlPort returns the component-facing Control API port.
func (f *Flag) ControlPort() int {
	return vnet.MustParsePort(f.ControlListen)
}

// AdminPort returns the Dashboard admin API and Web port.
func (f *Flag) AdminPort() int {
	return vnet.MustParsePort(f.AdminListen)
}

func (f *Flag) RedisPort() int {
	return vnet.MustParsePort(f.RedisListen)
}

func (f *Flag) normalizeDashboardURL() {
	rawURL := f.DashboardURLRaw
	if rawURL == "" {
		if f.MTLS.Enabled() {
			rawURL = HubMTLSDefaultDashboardURL
			f.DashboardURLMTLSDefault = true
		} else {
			rawURL = HubDefaultDashboardURL
		}
	} else {
		f.DashboardURLSet = true
	}

	parsed, err := vnet.ParseHttpURL(rawURL)
	vpre.CheckNilError(err, "parse DashboardURL failed")
	f.DashboardURL = parsed
}

func validateMQExternalNatsURL(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("MQExternalNatsURL is invalid: %w", err)
	}
	if parsed.Scheme != "nats" {
		return fmt.Errorf("MQExternalNatsURL currently only supports nats://")
	}
	if parsed.Host == "" {
		return fmt.Errorf("MQExternalNatsURL host is empty")
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
