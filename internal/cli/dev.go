package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/src/server/app"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	linkapp "go.yorun.ai/vine/internal/daemon/link/src/server/app"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	portalapp "go.yorun.ai/vine/internal/daemon/portal/src/server/app"
	portalflag "go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

const (
	commandDev = "dev"

	flagDevLinkAPIListen = "link-api-listen"
)

type _DevOption struct {
	LinkAPIListen     string
	SeedHubDataFile   string
	SeedHubSourceFile string
	SeedHubVarsFile   string
	NoDB              bool
	DBSQLiteFile      string
	DBPostgresURL     string
}

type _DevRuntime struct {
	hub     app.App
	portal  app.App
	link    app.App
	cleanup func()
}

// startDevRuntime is overridden in tests to assert parsed flags without starting the real runtime.
var startDevRuntime = func(option _DevOption) {
	runtime := newDevRuntime(option)
	runtime.StartAndWait()
}

func newDevCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandDev,
		Usage: "start a local runtime for external app development",
		Flags: []ucli.Flag{
			&ucli.StringFlag{
				Name:    flagDevLinkAPIListen,
				Sources: ucli.EnvVars(EnvLinkAPIListen),
				Value:   linkflag.LinkDefaultAPIListen,
				Usage:   "link API listen address for external apps",
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
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if cmd.Args().Len() > 0 {
				return fmt.Errorf("unexpected args for %s", commandDev)
			}

			startDevRuntime(_DevOption{
				LinkAPIListen:     cmd.String(flagDevLinkAPIListen),
				SeedHubDataFile:   cmd.String(FlagSeedDataFile),
				SeedHubSourceFile: cmd.String(FlagSeedSourceFile),
				SeedHubVarsFile:   cmd.String(FlagSeedVarsFile),
				NoDB:              cmd.Bool(FlagHubNoDB),
				DBSQLiteFile:      cmd.String(FlagHubDBSQLiteFile),
				DBPostgresURL:     cmd.String(FlagHubDBPostgresURL),
			})
			return nil
		},
	}
}

func newDevRuntime(option _DevOption) *_DevRuntime {
	hubFlag, cleanup := prepareDevHubFlag(option)
	completed := false
	defer func() {
		if !completed {
			cleanup()
		}
	}()

	runtime := &_DevRuntime{
		hub: app.NewInternalInproc[*hubapp.HubApp](
			app.With(hubFlag),
		),
		portal: app.NewInternalInproc[*portalapp.PortalApp](
			app.With(&portalflag.Flag{HubInprocMode: true}),
		),
		link: app.NewInternal[*linkapp.LinkApp](
			app.With(&linkflag.Flag{
				APIListen:     option.LinkAPIListen,
				HubInprocMode: true,
			}),
		),
		cleanup: cleanup,
	}
	completed = true
	return runtime
}

func prepareDevHubFlag(option _DevOption) (*hubflag.Flag, func()) {
	flag := &hubflag.Flag{
		SeedHubDataFile:   option.SeedHubDataFile,
		SeedHubSourceFile: option.SeedHubSourceFile,
		SeedHubVarsFile:   option.SeedHubVarsFile,
		NoDB:              option.NoDB,
		DBSQLiteFile:      option.DBSQLiteFile,
		DBPostgresURL:     option.DBPostgresURL,
	}
	return flag, func() {}
}

func (r *_DevRuntime) Start() {
	r.hub.Start()
	logger.Info("dev hub started")
	r.portal.Start()
	logger.Info("dev portal started")
	r.link.Start()
	logger.Info("dev link started")
}

func (r *_DevRuntime) StopGracefully() {
	r.link.StopGracefully()
	logger.Info("dev link stopped")
	r.portal.StopGracefully()
	logger.Info("dev portal stopped")
	r.hub.StopGracefully()
	logger.Info("dev hub stopped")
}

func (r *_DevRuntime) StartAndWait() {
	defer r.cleanup()
	r.Start()
	app.WaitExitSignal()
	r.StopGracefully()
}
