package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/src/server/app"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

const (
	commandHub      = "hub"
	commandHubServe = "serve"

	FlagHubAPIListen         = "api-listen"
	FlagHubRedisListen       = "redis-listen"
	FlagHubMQExternalNatsURL = "mq-external-nats-url"
	FlagHubMQEmbeddedNats    = "mq-embedded-nats"
	FlagHubSeedYAMLFile      = "seed-yaml-file"
	FlagHubDashboardURL      = "dashboard-url"
	FlagHubDBSQLiteFile      = "db-sqlite-file"
	FlagHubDBPostgresURL     = "db-postgres-url"

	EnvHubAPIListen         = "VINE_API_LISTEN"
	EnvHubRedisListen       = "VINE_REDIS_LISTEN"
	EnvHubMQExternalNatsURL = "VINE_MQ_EXTERNAL_NATS_URL"
	EnvHubMQEmbeddedNats    = "VINE_MQ_EMBEDDED_NATS"
	EnvHubSeedYAMLFile      = "VINE_SEED_YAML_FILE"
	EnvHubDashboardURL      = "VINE_DASHBOARD_URL"
	EnvHubDBSQLiteFile      = "VINE_DB_SQLITE_FILE"
	EnvHubDBPostgresURL     = "VINE_DB_POSTGRES_URL"
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
	return []ucli.Flag{
		&ucli.StringFlag{Name: FlagHubAPIListen, Sources: ucli.EnvVars(EnvHubAPIListen), Value: hubflag.HubDefaultAPIListen, Usage: "hub API listen address"},
		&ucli.StringFlag{Name: FlagHubRedisListen, Sources: ucli.EnvVars(EnvHubRedisListen), Value: hubflag.HubDefaultRedisListen, Usage: "hub redis listen address"},
		&ucli.StringFlag{Name: FlagHubDBSQLiteFile, Sources: ucli.EnvVars(EnvHubDBSQLiteFile), Usage: "hub SQLite database file"},
		&ucli.StringFlag{Name: FlagHubDBPostgresURL, Sources: ucli.EnvVars(EnvHubDBPostgresURL), Usage: "hub PostgreSQL database URL"},
		&ucli.StringFlag{Name: FlagHubMQExternalNatsURL, Sources: ucli.EnvVars(EnvHubMQExternalNatsURL), Usage: "external NATS URL, e.g. nats://127.0.0.1:4222"},
		&ucli.BoolFlag{Name: FlagHubMQEmbeddedNats, Sources: ucli.EnvVars(EnvHubMQEmbeddedNats), Usage: "start an embedded NATS server"},
		&ucli.StringFlag{Name: FlagHubSeedYAMLFile, Sources: ucli.EnvVars(EnvHubSeedYAMLFile), Usage: "hub seed YAML file"},
		&ucli.StringFlag{Name: FlagHubDashboardURL, Sources: ucli.EnvVars(EnvHubDashboardURL), Usage: "hub dashboard URL"},
	}
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
				APIListen:         cmd.String(FlagHubAPIListen),
				RedisListen:       cmd.String(FlagHubRedisListen),
				MQExternalNatsURL: cmd.String(FlagHubMQExternalNatsURL),
				MQEmbeddedNats:    cmd.Bool(FlagHubMQEmbeddedNats),
				SeedYAMLPath:      cmd.String(FlagHubSeedYAMLFile),
				DashboardURLRaw:   cmd.String(FlagHubDashboardURL),
				DBSQLiteFile:      cmd.String(FlagHubDBSQLiteFile),
				DBPostgresURL:     cmd.String(FlagHubDBPostgresURL),
			}
			startHubApp(flags)
			return nil
		},
	}
}
