package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
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

	FlagHubMQMode         = "mq-mode"
	FlagHubMQNatsEndpoint = "mq-nats-endpoint"

	FlagHubLockMode          = "lock-mode"
	FlagHubLockRedisEndpoint = "lock-redis-endpoint"

	FlagSeedSourceFile   = "seed-source-file"
	FlagSeedVarsFile     = "seed-vars-file"
	FlagSeedDataFile     = "seed-data-file"
	FlagHubNoDB          = "no-db"
	FlagHubDBSQLiteFile  = "db-sqlite-file"
	FlagHubDBPostgresURL = "db-postgres-url"

	EnvHubControlListen = "VINE_CONTROL_LISTEN"
	EnvHubAdminListen   = "VINE_ADMIN_LISTEN"
	EnvHubWatchListen   = "VINE_WATCH_LISTEN"

	EnvHubMQMode         = "VINE_MQ_MODE"
	EnvHubMQNatsEndpoint = "VINE_MQ_NATS_ENDPOINT"

	EnvHubLockMode          = "VINE_LOCK_MODE"
	EnvHubLockRedisEndpoint = "VINE_LOCK_REDIS_ENDPOINT"

	EnvSeedSourceFile   = "VINE_SEED_SOURCE_FILE"
	EnvSeedVarsFile     = "VINE_SEED_VARS_FILE"
	EnvSeedDataFile     = "VINE_SEED_DATA_FILE"
	EnvHubNoDB          = "VINE_NO_DB"
	EnvHubDBSQLiteFile  = "VINE_DB_SQLITE_FILE"
	EnvHubDBPostgresURL = "VINE_DB_POSTGRES_URL"
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
			Usage:   "hub admin API listen address",
		},
		&ucli.StringFlag{
			Name:    FlagHubWatchListen,
			Sources: ucli.EnvVars(EnvHubWatchListen),
			Value:   hubflag.HubDefaultWatchListen,
			Usage:   "hub configuration and service discovery watch listen address",
		},
		&ucli.BoolFlag{
			Name:    FlagHubNoDB,
			Sources: ucli.EnvVars(EnvHubNoDB),
			Usage:   "use no persistent database (default); requires seed-data-file; configuration is read-only",
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
			Name:    FlagSeedDataFile,
			Sources: ucli.EnvVars(EnvSeedDataFile),
			Usage:   "hub seed YAML file",
		},
		&ucli.StringFlag{
			Name:    FlagSeedSourceFile,
			Sources: ucli.EnvVars(EnvSeedSourceFile),
			Usage:   "hub seed source YAML file",
		},
		&ucli.StringFlag{
			Name:    FlagSeedVarsFile,
			Sources: ucli.EnvVars(EnvSeedVarsFile),
			Usage:   "hub seed vars YAML file",
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
				WatchListen:       cmd.String(FlagHubWatchListen),
				MQMode:            cmd.String(FlagHubMQMode),
				MQNatsEndpoint:    cmd.String(FlagHubMQNatsEndpoint),
				LockMode:          cmd.String(FlagHubLockMode),
				LockRedisEndpoint: cmd.String(FlagHubLockRedisEndpoint),
				SeedHubDataFile:   cmd.String(FlagSeedDataFile),
				SeedHubSourceFile: cmd.String(FlagSeedSourceFile),
				SeedHubVarsFile:   cmd.String(FlagSeedVarsFile),
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
