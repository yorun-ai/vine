package standalone

import (
	ucli "github.com/urfave/cli/v3"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

// Option configures the infrastructure started by standalone mode.
type Option struct {
	// SeedHubDataFile is the Hub seed configuration file, mutually exclusive with SeedHubData.
	SeedHubDataFile string

	// SeedHubData contains inline Hub seed YAML, mutually exclusive with SeedHubDataFile.
	// No-db mode requires one seed source; use "{}" for empty configuration.
	SeedHubData string

	// SeedHubSource contains an embedded seed source map and requires SeedHubData.
	SeedHubSource string
	// SeedHubSourceFile is the optional field source map and requires SeedHubDataFile.
	SeedHubSourceFile string
	// SeedHubVarsFile supplies a YAML mapping for ${path} and ${path:default} references.
	// Paths use camelCase segments separated by dots. Defaults apply only to
	// missing keys; existing null and zero values are preserved until use.
	// Importing skeled/app registers app.Vars for type checking; unused fields
	// are not required. Values inserted from this file are never re-expanded.
	SeedHubVarsFile string

	// NoDB loads read-only configuration from the seed YAML into memory. This is
	// the default when neither SQLiteFile nor PostgresURL is supplied.
	NoDB bool
	// SQLiteFile selects SQLite persistence and specifies its database file.
	SQLiteFile string
	// PostgresURL selects PostgreSQL persistence and specifies its connection URL.
	PostgresURL string

	// AdminListen is the in-process Hub's Admin API and Dashboard address. Empty
	// serves no Admin API; the API carries no authentication.
	AdminListen string
}

func (o Option) isZero() bool {
	return o.SeedHubDataFile == "" &&
		o.SeedHubData == "" && o.SeedHubSource == "" && o.SeedHubSourceFile == "" && o.SeedHubVarsFile == "" &&
		!o.NoDB &&
		o.SQLiteFile == "" &&
		o.PostgresURL == "" &&
		o.AdminListen == ""
}

const (
	// The business binary carries no command that could scope the Hub parameters
	// it accepts, so its flags and environment variables name the hub explicitly.
	// The names are exported for callers that embed the runtime and compose the
	// same command line.
	FlagHubNoDB           = "hub-no-db"
	FlagHubDBSQLiteFile   = "hub-db-sqlite-file"
	FlagHubDBPostgresURL  = "hub-db-postgres-url"
	FlagHubSeedDataFile   = "hub-seed-data-file"
	FlagHubSeedSourceFile = "hub-seed-source-file"
	FlagHubSeedVarsFile   = "hub-seed-vars-file"
	FlagHubAdminListen    = "hub-admin-listen"

	EnvHubNoDB           = "VINE_HUB_NO_DB"
	EnvHubDBSQLiteFile   = "VINE_HUB_DB_SQLITE_FILE"
	EnvHubDBPostgresURL  = "VINE_HUB_DB_POSTGRES_URL"
	EnvHubSeedDataFile   = "VINE_HUB_SEED_DATA_FILE"
	EnvHubSeedSourceFile = "VINE_HUB_SEED_SOURCE_FILE"
	EnvHubSeedVarsFile   = "VINE_HUB_SEED_VARS_FILE"
	EnvHubAdminListen    = "VINE_HUB_ADMIN_LISTEN"
)

// flags lists the Hub parameters the business binary accepts.
func flags(flag *hubflag.Flag) []ucli.Flag {
	return []ucli.Flag{
		&ucli.BoolFlag{
			Name:        FlagHubNoDB,
			Sources:     ucli.EnvVars(EnvHubNoDB),
			Usage:       "use no persistent database (default); requires the seed data file or Option.SeedHubData; configuration is read-only",
			Destination: &flag.NoDB,
		},
		&ucli.StringFlag{
			Name:        FlagHubDBSQLiteFile,
			Sources:     ucli.EnvVars(EnvHubDBSQLiteFile),
			Usage:       "in-process Hub SQLite database file",
			Destination: &flag.DBSQLiteFile,
		},
		&ucli.StringFlag{
			Name:        FlagHubDBPostgresURL,
			Sources:     ucli.EnvVars(EnvHubDBPostgresURL),
			Usage:       "in-process Hub PostgreSQL database URL",
			Destination: &flag.DBPostgresURL,
		},
		&ucli.StringFlag{
			Name:        FlagHubSeedDataFile,
			Sources:     ucli.EnvVars(EnvHubSeedDataFile),
			Usage:       "in-process Hub seed YAML file",
			Destination: &flag.SeedHubDataFile,
		},
		&ucli.StringFlag{
			Name:        FlagHubSeedSourceFile,
			Sources:     ucli.EnvVars(EnvHubSeedSourceFile),
			Usage:       "in-process Hub seed source YAML file",
			Destination: &flag.SeedHubSourceFile,
		},
		&ucli.StringFlag{
			Name:        FlagHubSeedVarsFile,
			Sources:     ucli.EnvVars(EnvHubSeedVarsFile),
			Usage:       "in-process Hub seed vars YAML file",
			Destination: &flag.SeedHubVarsFile,
		},
		&ucli.StringFlag{
			Name:        FlagHubAdminListen,
			Sources:     ucli.EnvVars(EnvHubAdminListen),
			Usage:       "in-process Hub Admin API and Dashboard listen address; unauthenticated, so loopback unless the network is trusted",
			Destination: &flag.AdminListen,
		},
	}
}

// applyOption copies the declared option over the parsed flags: a value the
// program sets wins over the command line and the environment.
func applyOption(flag *hubflag.Flag, option Option) {
	if option.SeedHubSource != "" {
		flag.SeedHubSource = option.SeedHubSource
	}
	if option.SeedHubSourceFile != "" {
		flag.SeedHubSourceFile = option.SeedHubSourceFile
	}
	if option.SeedHubVarsFile != "" {
		flag.SeedHubVarsFile = option.SeedHubVarsFile
	}

	if option.NoDB {
		flag.NoDB = true
	}
	if option.SQLiteFile != "" {
		flag.DBSQLiteFile = option.SQLiteFile
	}
	if option.PostgresURL != "" {
		flag.DBPostgresURL = option.PostgresURL
	}
	if option.AdminListen != "" {
		flag.AdminListen = option.AdminListen
	}
	if option.SeedHubData != "" {
		flag.SeedHubData = option.SeedHubData
	}
	if option.SeedHubDataFile != "" {
		flag.SeedHubDataFile = option.SeedHubDataFile
	}
}
