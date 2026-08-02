package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/src/server/app"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	linkapp "go.yorun.ai/vine/internal/daemon/link/src/server/app"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	portalapp "go.yorun.ai/vine/internal/daemon/portal/src/server/app"
	portalflag "go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
)

const (
	commandDev = "dev"

	flagDevLinkAPIListen = "link-api-listen"
)

type _DevOption struct {
	LinkAPIListen string
	SeedYAMLFile  string
	DashboardURL  string
	DBSQLiteFile  string
	DBPostgresURL string
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
			&ucli.StringFlag{Name: flagDevLinkAPIListen, Sources: ucli.EnvVars(EnvLinkAPIListen), Value: linkflag.LinkDefaultAPIListen, Usage: "link API listen address for external apps"},
			&ucli.StringFlag{Name: FlagHubDBSQLiteFile, Sources: ucli.EnvVars(EnvHubDBSQLiteFile), Usage: "hub SQLite database file; temporary when omitted"},
			&ucli.StringFlag{Name: FlagHubDBPostgresURL, Sources: ucli.EnvVars(EnvHubDBPostgresURL), Usage: "hub PostgreSQL database URL"},
			&ucli.StringFlag{Name: FlagHubSeedYAMLFile, Sources: ucli.EnvVars(EnvHubSeedYAMLFile), Usage: "hub seed YAML file"},
			&ucli.StringFlag{Name: FlagHubDashboardURL, Sources: ucli.EnvVars(EnvHubDashboardURL), Usage: "hub dashboard URL"},
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if cmd.Args().Len() > 0 {
				return fmt.Errorf("unexpected args for %s", commandDev)
			}

			startDevRuntime(_DevOption{
				LinkAPIListen: cmd.String(flagDevLinkAPIListen),
				SeedYAMLFile:  cmd.String(FlagHubSeedYAMLFile),
				DashboardURL:  cmd.String(FlagHubDashboardURL),
				DBSQLiteFile:  cmd.String(FlagHubDBSQLiteFile),
				DBPostgresURL: cmd.String(FlagHubDBPostgresURL),
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
		SeedYAMLPath:    option.SeedYAMLFile,
		DashboardURLRaw: option.DashboardURL,
		DBSQLiteFile:    option.DBSQLiteFile,
		DBPostgresURL:   option.DBPostgresURL,
	}
	if flag.DBSQLiteFile != "" || flag.DBPostgresURL != "" {
		return flag, func() {}
	}

	dir, err := os.MkdirTemp("", "vine-dev-")
	vpre.CheckNilError(err, "create temporary dev runtime directory failed")
	flag.DBSQLiteFile = filepath.Join(dir, "hub.sqlite")
	return flag, func() {
		if err := os.RemoveAll(dir); err != nil {
			logger.Warn("remove temporary dev runtime directory failed", "path", dir, "error", err)
		}
	}
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
