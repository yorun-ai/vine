package flag

import (
	"fmt"
	"net/url"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/util/vnet"
	"go.yorun.ai/vine/util/vpre"
)

const (
	HubDefaultControlListen    = "127.0.0.1:7071"
	HubDefaultManagementListen = "127.0.0.1:7072"
	HubDefaultRedisListen      = "127.0.0.1:7073"
	HubDefaultDashboardURL     = "http://:7099/"

	SourceSQLite     = "sqlite"
	SourcePostgreSQL = "postgres"
)

type Flag struct {
	app.FlagModel
	ControlListen    string
	ManagementListen string
	RedisListen      string

	MQExternalNatsURL string
	MQEmbeddedNats    bool

	SourceType    string
	SeedYAMLPath  string
	DBSQLiteFile  string
	DBPostgresURL string

	DashboardURLRaw string
	DashboardURLSet bool
	DashboardURL    *vnet.HttpURL
}

func (f *Flag) Normalize(inproc bool) {
	f.normalizeSource()
	f.normalizeDashboardURL()

	if inproc {
		// Inproc hub is reached through rpc+inproc and uses in-process NATS,
		// so external listen addresses and MQ endpoint must not leak into runtime info.
		f.ControlListen = ""
		f.ManagementListen = ""
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
	if f.ManagementListen == "" {
		f.ManagementListen = HubDefaultManagementListen
	}
	if f.RedisListen == "" {
		f.RedisListen = HubDefaultRedisListen
	}
}

func (f *Flag) normalizeSource() {
	kind := f.SourceType
	if kind == "" {
		var err error
		kind, err = f.inferSourceType()
		vpre.CheckNilError(err, "hub flag normalize failed")
	}
	f.SourceType = kind

	switch kind {
	case SourceSQLite:
		vpre.CheckNotEmpty(f.DBSQLiteFile, "DBSQLiteFile is empty")
	case SourcePostgreSQL:
		vpre.CheckNotEmpty(f.DBPostgresURL, "DBPostgresURL is empty")
	default:
		vpre.Panicf("unsupported hub source type %q", kind)
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

// ManagementPort returns the Dashboard management API and Web port.
func (f *Flag) ManagementPort() int {
	return vnet.MustParsePort(f.ManagementListen)
}

func (f *Flag) RedisPort() int {
	return vnet.MustParsePort(f.RedisListen)
}

func (f *Flag) normalizeDashboardURL() {
	rawURL := f.DashboardURLRaw
	if rawURL == "" {
		rawURL = HubDefaultDashboardURL
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

func (f *Flag) inferSourceType() (string, error) {
	hasSQLite := f.DBSQLiteFile != ""
	hasPostgreSQL := f.DBPostgresURL != ""

	sourceCount := 0
	if hasSQLite {
		sourceCount++
	}
	if hasPostgreSQL {
		sourceCount++
	}
	if sourceCount > 1 {
		return "", fmt.Errorf("only one of DBSQLiteFile or DBPostgresURL can be set")
	}

	switch {
	case hasSQLite:
		return SourceSQLite, nil
	case hasPostgreSQL:
		return SourcePostgreSQL, nil
	default:
		return "", fmt.Errorf("one of DBSQLiteFile or DBPostgresURL must be set")
	}
}
