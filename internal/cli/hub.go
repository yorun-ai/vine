package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/src/server/app"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

const (
	commandHub      = "hub"
	commandHubServe = "serve"

	FlagHubControlListen = "control-listen"
	FlagHubAdminListen   = "admin-listen"
	FlagHubWatchListen   = "watch-listen"
	// Deprecated: use FlagHubWatchListen.
	FlagHubRedisListen = "redis-listen"

	FlagHubMQEmbedded     = "mq-embedded"
	FlagHubMQNatsEndpoint = "mq-nats-endpoint"
	// Deprecated: use FlagHubMQEmbedded.
	FlagHubMQEmbeddedNats = "mq-embedded-nats"
	// Deprecated: use FlagHubMQNatsEndpoint.
	FlagHubMQExternalNatsURL = "mq-external-nats-url"

	FlagSeedHubSourceFile = "seed-hub-source-file"
	FlagSeedHubVarsFile   = "seed-hub-vars-file"
	FlagSeedHubDataFile   = "seed-hub-data-file"
	FlagHubDashboardURL   = "dashboard-url"
	FlagHubNoDB           = "no-db"
	FlagHubDBSQLiteFile   = "db-sqlite-file"
	FlagHubDBPostgresURL  = "db-postgres-url"

	EnvHubControlListen = "VINE_CONTROL_LISTEN"
	EnvHubAdminListen   = "VINE_ADMIN_LISTEN"
	EnvHubWatchListen   = "VINE_WATCH_LISTEN"
	// Deprecated: use EnvHubWatchListen.
	EnvHubRedisListen = "VINE_REDIS_LISTEN"

	EnvHubMQEmbedded     = "VINE_MQ_EMBEDDED"
	EnvHubMQNatsEndpoint = "VINE_MQ_NATS_ENDPOINT"
	// Deprecated: use EnvHubMQEmbedded.
	EnvHubMQEmbeddedNats = "VINE_MQ_EMBEDDED_NATS"
	// Deprecated: use EnvHubMQNatsEndpoint.
	EnvHubMQExternalNatsURL = "VINE_MQ_EXTERNAL_NATS_URL"

	EnvSeedHubSourceFile = "VINE_SEED_HUB_SOURCE_FILE"
	EnvSeedHubVarsFile   = "VINE_SEED_HUB_VARS_FILE"
	EnvSeedHubDataFile   = "VINE_SEED_HUB_DATA_FILE"
	EnvHubDashboardURL   = "VINE_DASHBOARD_URL"
	EnvHubNoDB           = "VINE_NO_DB"
	EnvHubDBSQLiteFile   = "VINE_DB_SQLITE_FILE"
	EnvHubDBPostgresURL  = "VINE_DB_POSTGRES_URL"
)

// startHubApp is overridden in tests to assert parsed flags without starting the real app.
var startHubApp = func(flags hubflag.Flag) {
	app.NewInternal[*hubapp.HubApp](
		app.With(&flags),
	).StartAndWait()
}

func newHubCommand() *ucli.Command {
	return &ucli.Command{
		Name:               commandHub,
		Usage:              "configuration and service registry",
		Suggest:            true,
		CustomHelpTemplate: groupCommandHelpTemplate,
		Commands: []*ucli.Command{
			newHubServeCommand(),
		},
	}
}

func newHubServeFlags() []ucli.Flag {
	return append([]ucli.Flag{
		&ucli.StringFlag{
			Name:    FlagHubControlListen,
			Sources: ucli.EnvVars(EnvHubControlListen),
			Value:   hubflag.HubDefaultControlListen,
			Usage:   "hub Control API listen address used by Link and Portal",
		},
		&ucli.StringFlag{
			Name:    FlagHubAdminListen,
			Sources: ucli.EnvVars(EnvHubAdminListen),
			Value:   hubflag.HubDefaultAdminListen,
			Usage:   "hub admin API and Dashboard Web listen address",
		},
		&ucli.StringFlag{
			Name:    FlagHubWatchListen,
			Sources: ucli.EnvVars(EnvHubWatchListen),
			Value:   hubflag.HubDefaultWatchListen,
			Usage:   "hub configuration and service discovery watch listen address",
		},
		&ucli.StringFlag{
			Name:    FlagHubRedisListen,
			Sources: ucli.EnvVars(EnvHubRedisListen),
			Usage:   "deprecated: use --watch-listen or VINE_WATCH_LISTEN",
		},
		&ucli.BoolFlag{
			Name:    FlagHubNoDB,
			Sources: ucli.EnvVars(EnvHubNoDB),
			Usage:   "use no persistent database (default); requires seed-hub-data-file; configuration is read-only",
		},
		&ucli.StringFlag{
			Name:    FlagHubDBSQLiteFile,
			Sources: ucli.EnvVars(EnvHubDBSQLiteFile),
			Usage:   "hub SQLite database file",
		},
		&ucli.StringFlag{
			Name:    FlagHubDBPostgresURL,
			Sources: ucli.EnvVars(EnvHubDBPostgresURL),
			Usage:   "hub PostgreSQL database URL",
		},
		&ucli.BoolFlag{
			Name:    FlagHubMQEmbedded,
			Sources: ucli.EnvVars(EnvHubMQEmbedded),
			Usage:   "start an embedded NATS server",
		},
		&ucli.StringFlag{
			Name:    FlagHubMQNatsEndpoint,
			Sources: ucli.EnvVars(EnvHubMQNatsEndpoint),
			Usage:   "external NATS URL, e.g. nats://127.0.0.1:4222",
		},
		&ucli.BoolFlag{
			Name:    FlagHubMQEmbeddedNats,
			Sources: ucli.EnvVars(EnvHubMQEmbeddedNats),
			Usage:   "deprecated: use --mq-embedded or VINE_MQ_EMBEDDED",
		},
		&ucli.StringFlag{
			Name:    FlagHubMQExternalNatsURL,
			Sources: ucli.EnvVars(EnvHubMQExternalNatsURL),
			Usage:   "deprecated: use --mq-nats-endpoint or VINE_MQ_NATS_ENDPOINT",
		},

		&ucli.StringFlag{
			Name:    FlagSeedHubDataFile,
			Sources: ucli.EnvVars(EnvSeedHubDataFile),
			Usage:   "hub seed YAML file",
		},
		&ucli.StringFlag{
			Name:    FlagSeedHubSourceFile,
			Sources: ucli.EnvVars(EnvSeedHubSourceFile),
			Usage:   "hub seed source YAML file",
		},
		&ucli.StringFlag{
			Name:    FlagSeedHubVarsFile,
			Sources: ucli.EnvVars(EnvSeedHubVarsFile),
			Usage:   "hub seed vars YAML file",
		},
		&ucli.StringFlag{
			Name:    FlagHubDashboardURL,
			Sources: ucli.EnvVars(EnvHubDashboardURL),
			Usage:   "hub dashboard URL",
		},
	}, mtlsFlags()...)
}

func newHubServeCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandHubServe,
		Usage: "start the hub service",
		Flags: newHubServeFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if cmd.Args().Len() > 0 {
				return fmt.Errorf("unexpected args for %s", commandHubServe)
			}

			flags := hubflag.Flag{
				ControlListen:     cmd.String(FlagHubControlListen),
				AdminListen:       cmd.String(FlagHubAdminListen),
				WatchListen:       hubWatchListen(cmd),
				MQNatsEndpoint:    hubMQNatsEndpoint(cmd),
				MQEmbedded:        hubMQEmbedded(cmd),
				SeedHubDataFile:   cmd.String(FlagSeedHubDataFile),
				SeedHubSourceFile: cmd.String(FlagSeedHubSourceFile),
				SeedHubVarsFile:   cmd.String(FlagSeedHubVarsFile),
				DashboardURLRaw:   cmd.String(FlagHubDashboardURL),
				NoDB:              cmd.Bool(FlagHubNoDB),
				DBSQLiteFile:      cmd.String(FlagHubDBSQLiteFile),
				DBPostgresURL:     cmd.String(FlagHubDBPostgresURL),
				MTLS:              mtlsFiles(cmd),
			}
			startHubApp(flags)
			return nil
		},
	}
}

// hubWatchListen normalizes compatibility inputs before constructing Hub flags.
func hubWatchListen(cmd *ucli.Command) string {
	if cmd.IsSet(FlagHubRedisListen) {
		logger.Warn("--redis-listen / VINE_REDIS_LISTEN is deprecated; use --watch-listen / VINE_WATCH_LISTEN")
		if !cmd.IsSet(FlagHubWatchListen) {
			return cmd.String(FlagHubRedisListen)
		}
	}
	return cmd.String(FlagHubWatchListen)
}

func hubMQEmbedded(cmd *ucli.Command) bool {
	if cmd.IsSet(FlagHubMQEmbeddedNats) {
		logger.Warn("--mq-embedded-nats and VINE_MQ_EMBEDDED_NATS are deprecated; use --mq-embedded or VINE_MQ_EMBEDDED")
		if !cmd.IsSet(FlagHubMQEmbedded) {
			return cmd.Bool(FlagHubMQEmbeddedNats)
		}
	}
	return cmd.Bool(FlagHubMQEmbedded)
}

func hubMQNatsEndpoint(cmd *ucli.Command) string {
	if cmd.IsSet(FlagHubMQExternalNatsURL) {
		logger.Warn("--mq-external-nats-url and VINE_MQ_EXTERNAL_NATS_URL are deprecated; use --mq-nats-endpoint or VINE_MQ_NATS_ENDPOINT")
		if !cmd.IsSet(FlagHubMQNatsEndpoint) {
			return cmd.String(FlagHubMQExternalNatsURL)
		}
	}
	return cmd.String(FlagHubMQNatsEndpoint)
}
