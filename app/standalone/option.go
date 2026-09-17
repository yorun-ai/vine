package standalone

import (
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/appcli"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

// Option configures the infrastructure started by standalone mode.
type Option struct {
	// HubAdminListen is the in-process Hub's Admin API and Dashboard address. Empty
	// serves no Admin API; the API carries no authentication.
	HubAdminListen string

	// HubNoDB loads read-only configuration from the seed YAML into memory. This is
	// the default when neither HubDBSQLiteFile nor HubDBPostgresURL is supplied.
	HubNoDB bool
	// HubDBSQLiteFile selects SQLite persistence and specifies its database file.
	HubDBSQLiteFile string
	// HubDBPostgresURL selects PostgreSQL persistence and specifies its connection URL.
	HubDBPostgresURL string

	// HubSeedData contains inline Hub seed YAML, mutually exclusive with HubSeedDataFile.
	// No-db mode requires one seed source; use "{}" for empty configuration.
	HubSeedData string
	// HubSeedDataFile is the Hub seed configuration file, mutually exclusive with HubSeedData.
	HubSeedDataFile string
	// HubSeedSource contains an embedded seed source map and requires HubSeedData.
	HubSeedSource string
	// HubSeedSourceFile is the optional field source map and requires HubSeedDataFile.
	HubSeedSourceFile string
	// HubSeedVarsFile supplies a YAML mapping for ${path} and ${path:default} references.
	// Paths use camelCase segments separated by dots. Defaults apply only to
	// missing keys; existing null and zero values are preserved until use.
	// Importing skeled/app registers app.Vars for type checking; unused fields
	// are not required. Values inserted from this file are never re-expanded.
	HubSeedVarsFile string

	// IgnoredFlags lists the flags the binary accepts but discards, named with the
	// Flag constants of this package. The named flags and their environment
	// variables stop reaching the runtime, so an embedding program can own that
	// parameter; setting the matching Option field still applies it.
	IgnoredFlags []string

	// RenamedFlags maps a declared flag name, such as FlagHubAdminListen, to the
	// name the binary registers it under. The declared flag and its environment
	// variable are dropped: the new name carries the environment variable derived
	// from it. A renamed flag cannot also be ignored.
	RenamedFlags map[string]string
}

func (o Option) isZero() bool {
	return o.HubAdminListen == "" &&
		!o.HubNoDB &&
		o.HubDBSQLiteFile == "" &&
		o.HubDBPostgresURL == "" &&
		o.HubSeedData == "" &&
		o.HubSeedDataFile == "" &&
		o.HubSeedSource == "" &&
		o.HubSeedSourceFile == "" &&
		o.HubSeedVarsFile == "" &&
		len(o.IgnoredFlags) == 0 &&
		len(o.RenamedFlags) == 0
}

const (
	// The business binary carries no command that could scope the Hub parameters
	// it accepts, so its flags and environment variables name the hub explicitly.
	// The names are exported for callers that embed the runtime and compose the
	// same command line.
	FlagHubAdminListen    = "hub-admin-listen"
	FlagHubNoDB           = "hub-no-db"
	FlagHubDBSQLiteFile   = "hub-db-sqlite-file"
	FlagHubDBPostgresURL  = "hub-db-postgres-url"
	FlagHubSeedDataFile   = "hub-seed-data-file"
	FlagHubSeedSourceFile = "hub-seed-source-file"
	FlagHubSeedVarsFile   = "hub-seed-vars-file"

	EnvHubAdminListen    = "VINE_HUB_ADMIN_LISTEN"
	EnvHubNoDB           = "VINE_HUB_NO_DB"
	EnvHubDBSQLiteFile   = "VINE_HUB_DB_SQLITE_FILE"
	EnvHubDBPostgresURL  = "VINE_HUB_DB_POSTGRES_URL"
	EnvHubSeedDataFile   = "VINE_HUB_SEED_DATA_FILE"
	EnvHubSeedSourceFile = "VINE_HUB_SEED_SOURCE_FILE"
	EnvHubSeedVarsFile   = "VINE_HUB_SEED_VARS_FILE"
)

// flags lists the Hub parameters the business binary accepts, as the option
// presents them: a name it ignores keeps its command line and environment source
// but stops reaching the runtime, and a name it renames answers to the new name
// alone.
func flags(flag *hubflag.Flag, option Option) []ucli.Flag {
	names := appcli.NewFlagNames(option.IgnoredFlags, option.RenamedFlags)

	list := []ucli.Flag{
		names.String(FlagHubAdminListen, EnvHubAdminListen, &flag.AdminListen,
			"in-process Hub Admin API and Dashboard listen address; unauthenticated, so loopback unless the network is trusted"),
		names.Bool(FlagHubNoDB, EnvHubNoDB, &flag.NoDB,
			"use no persistent database (default); requires the seed data file or Option.HubSeedData; configuration is read-only"),
		names.String(FlagHubDBSQLiteFile, EnvHubDBSQLiteFile, &flag.DBSQLiteFile, "in-process Hub SQLite database file"),
		names.String(FlagHubDBPostgresURL, EnvHubDBPostgresURL, &flag.DBPostgresURL, "in-process Hub PostgreSQL database URL"),
		names.String(FlagHubSeedDataFile, EnvHubSeedDataFile, &flag.SeedHubDataFile, "in-process Hub seed YAML file"),
		names.String(FlagHubSeedSourceFile, EnvHubSeedSourceFile, &flag.SeedHubSourceFile, "in-process Hub seed source YAML file"),
		names.String(FlagHubSeedVarsFile, EnvHubSeedVarsFile, &flag.SeedHubVarsFile, "in-process Hub seed vars YAML file"),
	}

	names.Validate()
	return list
}

// applyOption copies the declared option over the parsed flags: a value the
// program sets wins over the command line and the environment.
func applyOption(flag *hubflag.Flag, option Option) {
	if option.HubAdminListen != "" {
		flag.AdminListen = option.HubAdminListen
	}
	if option.HubNoDB {
		flag.NoDB = true
	}
	if option.HubDBSQLiteFile != "" {
		flag.DBSQLiteFile = option.HubDBSQLiteFile
	}
	if option.HubDBPostgresURL != "" {
		flag.DBPostgresURL = option.HubDBPostgresURL
	}
	if option.HubSeedData != "" {
		flag.SeedHubData = option.HubSeedData
	}
	if option.HubSeedDataFile != "" {
		flag.SeedHubDataFile = option.HubSeedDataFile
	}
	if option.HubSeedSource != "" {
		flag.SeedHubSource = option.HubSeedSource
	}
	if option.HubSeedSourceFile != "" {
		flag.SeedHubSourceFile = option.HubSeedSourceFile
	}
	if option.HubSeedVarsFile != "" {
		flag.SeedHubVarsFile = option.HubSeedVarsFile
	}
}
