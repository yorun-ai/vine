package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	hublock "go.yorun.ai/vine/internal/daemon/hub/api/lock"
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

	FlagHubMQMode         = "mq-mode"
	FlagHubMQNatsEndpoint = "mq-nats-endpoint"
	// Deprecated: use FlagHubMQMode.
	FlagHubMQEmbeddedNats = "mq-embedded-nats"
	// Deprecated: use FlagHubMQNatsEndpoint.
	FlagHubMQExternalNatsURL = "mq-external-nats-url"

	FlagHubLockMode          = "lock-mode"
	FlagHubLockRedisEndpoint = "lock-redis-endpoint"

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

	EnvHubMQMode         = "VINE_MQ_MODE"
	EnvHubMQNatsEndpoint = "VINE_MQ_NATS_ENDPOINT"
	// Deprecated: use EnvHubMQMode.
	EnvHubMQEmbeddedNats = "VINE_MQ_EMBEDDED_NATS"
	// Deprecated: use EnvHubMQNatsEndpoint.
	EnvHubMQExternalNatsURL = "VINE_MQ_EXTERNAL_NATS_URL"

	EnvHubLockMode          = "VINE_LOCK_MODE"
	EnvHubLockRedisEndpoint = "VINE_LOCK_REDIS_ENDPOINT"

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
		&ucli.StringFlag{
			Name:    FlagHubMQMode,
			Sources: ucli.EnvVars(EnvHubMQMode),
			Value:   hubflag.MQModeEmbedded,
			Usage:   "MQ mode: embedded or nats",
		},
		&ucli.StringFlag{
			Name:    FlagHubMQNatsEndpoint,
			Sources: ucli.EnvVars(EnvHubMQNatsEndpoint),
			Usage:   "external NATS URL, e.g. nats://127.0.0.1:4222",
		},
		&ucli.BoolFlag{
			Name:    FlagHubMQEmbeddedNats,
			Sources: ucli.EnvVars(EnvHubMQEmbeddedNats),
			Usage:   "deprecated: use --mq-mode or VINE_MQ_MODE",
		},
		&ucli.StringFlag{
			Name:    FlagHubMQExternalNatsURL,
			Sources: ucli.EnvVars(EnvHubMQExternalNatsURL),
			Usage:   "deprecated: use --mq-nats-endpoint or VINE_MQ_NATS_ENDPOINT",
		},
		&ucli.StringFlag{
			Name:    FlagHubLockMode,
			Sources: ucli.EnvVars(EnvHubLockMode),
			Value:   hublock.ModeEmbedded,
			Usage:   "lock mode: embedded, redis or disable",
		},
		&ucli.StringFlag{
			Name:    FlagHubLockRedisEndpoint,
			Sources: ucli.EnvVars(EnvHubLockRedisEndpoint),
			Usage:   "external Redis URL for locks",
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

			mqMode, mqEndpoint := hubMQConfig(cmd)
			flags := hubflag.Flag{
				ControlListen:     cmd.String(FlagHubControlListen),
				AdminListen:       cmd.String(FlagHubAdminListen),
				WatchListen:       hubWatchListen(cmd),
				MQMode:            mqMode,
				MQNatsEndpoint:    mqEndpoint,
				LockMode:          cmd.String(FlagHubLockMode),
				LockRedisEndpoint: cmd.String(FlagHubLockRedisEndpoint),
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

// hubMQConfig translates only published compatibility inputs at the CLI boundary.
func hubMQConfig(cmd *ucli.Command) (string, string) {
	endpoint := cmd.String(FlagHubMQNatsEndpoint)
	if cmd.IsSet(FlagHubMQExternalNatsURL) {
		logger.Warn("--mq-external-nats-url / VINE_MQ_EXTERNAL_NATS_URL is deprecated; use --mq-mode=nats and --mq-nats-endpoint")
		if !cmd.IsSet(FlagHubMQNatsEndpoint) {
			endpoint = cmd.String(FlagHubMQExternalNatsURL)
		}
	}
	if cmd.IsSet(FlagHubMQEmbeddedNats) {
		logger.Warn("--mq-embedded-nats / VINE_MQ_EMBEDDED_NATS is deprecated; use --mq-mode")
	}
	mode := cmd.String(FlagHubMQMode)
	if !cmd.IsSet(FlagHubMQMode) {
		if cmd.IsSet(FlagHubMQEmbeddedNats) {
			if cmd.Bool(FlagHubMQEmbeddedNats) {
				mode = hubflag.MQModeEmbedded
			} else {
				mode = hubflag.MQModeNATS
			}
		} else if cmd.IsSet(FlagHubMQExternalNatsURL) && !cmd.IsSet(FlagHubMQNatsEndpoint) && endpoint != "" {
			mode = hubflag.MQModeNATS
		}
	}
	return mode, endpoint
}
