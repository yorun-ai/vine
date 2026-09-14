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
	HubDefaultWatchListen      = "127.0.0.1:7072"
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
	WatchListen   string

	MQNatsEndpoint string
	MQEmbedded     bool

	Store         string
	NoDB          bool
	DBSQLiteFile  string
	DBPostgresURL string

	SeedHubData       string
	SeedHubDataFile   string
	SeedHubSource     string
	SeedHubSourceFile string
	SeedHubVarsFile   string

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
		f.WatchListen = ""
		f.MQNatsEndpoint = ""
		f.MQEmbedded = true
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
	if f.WatchListen == "" {
		f.WatchListen = HubDefaultWatchListen
	}
}

func (f *Flag) normalizeSeed() {
	vpre.CheckNot(f.SeedHubSource != "" && f.SeedHubSourceFile != "", "SeedHubSource and seed-hub-source-file are mutually exclusive")
	vpre.CheckNot((f.SeedHubSource != "" || f.SeedHubSourceFile != "" || f.SeedHubVarsFile != "") && f.SeedHubData == "" && f.SeedHubDataFile == "", "seed source and variables require seed YAML")
	vpre.CheckNot(f.SeedHubDataFile != "" && f.SeedHubData != "", "SeedHubData and seed-hub-data-file are mutually exclusive")
	vpre.CheckNot(f.SeedHubSource != "" && f.SeedHubData == "", "SeedHubSource requires inline SeedHubData")
	vpre.CheckNot(f.SeedHubSourceFile != "" && f.SeedHubDataFile == "", "seed-hub-source-file requires seed-hub-data-file")
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
		vpre.Check(f.SeedHubDataFile != "" || f.SeedHubData != "", "no-db requires seed-hub-data-file or SeedHubData")
	case StoreSQLite:
		vpre.CheckNotEmpty(f.DBSQLiteFile, "DBSQLiteFile is empty")
	case StorePostgreSQL:
		vpre.CheckNotEmpty(f.DBPostgresURL, "DBPostgresURL is empty")
	default:
		vpre.Panicf("unsupported hub store %q", kind)
	}
}

func (f *Flag) normalizeMQ() {
	if (f.MQNatsEndpoint != "") == f.MQEmbedded {
		vpre.Panicf("exactly one of MQNatsEndpoint or MQEmbedded must be set")
	}
	if f.MQNatsEndpoint != "" {
		vpre.CheckNilError(validateMQNatsEndpoint(f.MQNatsEndpoint), "hub flag normalize failed")
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

func (f *Flag) WatchPort() int {
	return vnet.MustParsePort(f.WatchListen)
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
