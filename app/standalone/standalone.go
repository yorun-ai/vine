package standalone

import (
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/app"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
	vinecli "go.yorun.ai/vine/internal/cli"
	"go.yorun.ai/vine/internal/core/logger"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/src/server/app"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	linkapp "go.yorun.ai/vine/internal/daemon/link/src/server/app"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	portalapp "go.yorun.ai/vine/internal/daemon/portal/src/server/app"
	portalflag "go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
)

type _App struct {
	option Option

	hub    app.App
	portal app.App
	link   app.App
	apps   []app.App
}

// Option configures the infrastructure started by standalone mode.
type Option struct {
	// Deprecated: use SeedHubDataFile.
	SeedYAMLFile string
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

	// DashboardURL is the optional URL from which Hub dashboard assets are loaded.
	DashboardURL string
}

func (o Option) isZero() bool {
	return o.SeedYAMLFile == "" && o.SeedHubDataFile == "" &&
		o.SeedHubData == "" && o.SeedHubSource == "" && o.SeedHubSourceFile == "" && o.SeedHubVarsFile == "" &&
		!o.NoDB &&
		o.SQLiteFile == "" &&
		o.PostgresURL == "" &&
		o.DashboardURL == ""
}

// New constructs an application with an in-process Hub, Portal, and Link.
func New[S app.ApplicationSpec](appliers ...app.FlagApplier) app.App {
	return NewWithOption[S](Option{}, appliers...)
}

// NewWithOption constructs a standalone application using option.
func NewWithOption[S app.ApplicationSpec](option Option, appliers ...app.FlagApplier) app.App {
	return &_App{
		option: option,
		apps:   []app.App{internalapp.NewInproc[S](appliers...)},
	}
}

// NewBundled combines standalone applications into one in-process runtime.
func NewBundled(apps ...app.App) app.App {
	return NewBundledWithOption(Option{}, apps...)
}

// NewBundledWithOption combines standalone applications and configures their shared runtime.
func NewBundledWithOption(option Option, apps ...app.App) app.App {
	vpre.Check(len(apps) > 0, "standalone app expected")

	bundle := &_App{option: option}
	for _, application := range apps {
		standaloneApp, ok := application.(*_App)
		vpre.Check(ok, "standalone app expected")
		vpre.Check(standaloneApp.option.isZero(), "bundled standalone app must not have option")
		bundle.apps = append(bundle.apps, standaloneApp.apps...)
	}
	return bundle
}

func (*_App) Name() string {
	return ""
}

func (a *_App) Start() {
	a.initInfra()
	a.startInfra()
	for _, application := range a.apps {
		application.Start()
		logger.Info("standalone app started", "name", application.Name())
	}
}

func (a *_App) StopGracefully() {
	for i := range a.apps {
		application := a.apps[len(a.apps)-1-i]
		application.StopGracefully()
		logger.Info("standalone app stopped", "name", application.Name())
	}
	a.stopInfraGracefully()
}

func (a *_App) StartAndWait() {
	a.Start()
	internalapp.WaitExitSignal()
	a.StopGracefully()
}

const (
	flagSQLiteFile      = vinecli.FlagHubDBSQLiteFile
	flagPostgresURL     = vinecli.FlagHubDBPostgresURL
	flagSeedHubDataFile = vinecli.FlagSeedHubDataFile
	flagDashboardURL    = vinecli.FlagHubDashboardURL

	envSQLiteFile      = vinecli.EnvHubDBSQLiteFile
	envPostgresURL     = vinecli.EnvHubDBPostgresURL
	envSeedHubDataFile = vinecli.EnvSeedHubDataFile
	envDashboardURL    = vinecli.EnvHubDashboardURL
)

func (a *_App) initInfra() {
	flag := &hubflag.Flag{}
	appcli.Handle(
		&ucli.BoolFlag{
			Name:        vinecli.FlagHubNoDB,
			Sources:     ucli.EnvVars(vinecli.EnvHubNoDB),
			Usage:       "use no persistent database (default); requires seed-hub-data-file or Option.SeedHubData; configuration is read-only",
			Destination: &flag.NoDB,
		},
		&ucli.StringFlag{
			Name:        flagSQLiteFile,
			Sources:     ucli.EnvVars(envSQLiteFile),
			Usage:       "hub SQLite file",
			Destination: &flag.DBSQLiteFile,
		},
		&ucli.StringFlag{
			Name:        flagPostgresURL,
			Sources:     ucli.EnvVars(envPostgresURL),
			Usage:       "hub PostgreSQL URL",
			Destination: &flag.DBPostgresURL,
		},
		&ucli.StringFlag{
			Name:        vinecli.FlagHubSeedYAMLFile,
			Sources:     ucli.EnvVars(vinecli.EnvHubSeedYAMLFile),
			Usage:       "deprecated: use --seed-hub-data-file",
			Destination: &flag.SeedYAMLFile,
		},
		&ucli.StringFlag{
			Name:        flagSeedHubDataFile,
			Sources:     ucli.EnvVars(envSeedHubDataFile),
			Usage:       "seed YAML file",
			Destination: &flag.SeedHubDataFile,
		},
		&ucli.StringFlag{
			Name:        vinecli.FlagSeedHubSourceFile,
			Sources:     ucli.EnvVars(vinecli.EnvSeedHubSourceFile),
			Usage:       "seed source YAML file",
			Destination: &flag.SeedHubSourceFile,
		},
		&ucli.StringFlag{
			Name:        vinecli.FlagSeedHubVarsFile,
			Sources:     ucli.EnvVars(vinecli.EnvSeedHubVarsFile),
			Usage:       "seed vars YAML file",
			Destination: &flag.SeedHubVarsFile,
		},
		&ucli.StringFlag{
			Name:        flagDashboardURL,
			Sources:     ucli.EnvVars(envDashboardURL),
			Usage:       "hub dashboard URL",
			Destination: &flag.DashboardURLRaw,
		},
	)
	applyOption(flag, a.option)

	a.hub = internalapp.NewInternalInproc[*hubapp.HubApp](internalapp.With(flag))
	a.link = internalapp.NewInternalInproc[*linkapp.LinkApp](internalapp.With(&linkflag.Flag{
		HubInprocMode: true,
	}))
	a.portal = internalapp.NewInternalInproc[*portalapp.PortalApp](internalapp.With(&portalflag.Flag{
		HubInprocMode: true,
	}))
}

func applyOption(flag *hubflag.Flag, option Option) {
	if option.SeedYAMLFile != "" {
		flag.SeedYAMLFile = option.SeedYAMLFile
	}
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
	if option.SeedHubData != "" {
		flag.SeedHubData = option.SeedHubData
	}
	if option.SeedHubDataFile != "" {
		flag.SeedHubDataFile = option.SeedHubDataFile
	}
	if option.DashboardURL != "" {
		flag.DashboardURLRaw = option.DashboardURL
	}
}

func (a *_App) startInfra() {
	a.hub.Start()
	logger.Info("standalone hub started")
	a.portal.Start()
	logger.Info("standalone portal started")
	a.link.Start()
	logger.Info("standalone link started")
}

func (a *_App) stopInfraGracefully() {
	a.link.StopGracefully()
	logger.Info("standalone link stopped")
	a.portal.StopGracefully()
	logger.Info("standalone portal stopped")
	a.hub.StopGracefully()
	logger.Info("standalone hub stopped")
}
