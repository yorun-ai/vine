package standalone

import (
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/app"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
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
	// The business binary carries no command that could scope the Hub parameters
	// it accepts, so its flags and environment variables name the hub explicitly.
	flagHubNoDB           = "hub-no-db"
	flagHubDBSQLiteFile   = "hub-db-sqlite-file"
	flagHubDBPostgresURL  = "hub-db-postgres-url"
	flagHubSeedDataFile   = "hub-seed-data-file"
	flagHubSeedSourceFile = "hub-seed-source-file"
	flagHubSeedVarsFile   = "hub-seed-vars-file"
	flagHubAdminListen    = "hub-admin-listen"

	envHubNoDB           = "VINE_HUB_NO_DB"
	envHubDBSQLiteFile   = "VINE_HUB_DB_SQLITE_FILE"
	envHubDBPostgresURL  = "VINE_HUB_DB_POSTGRES_URL"
	envHubSeedDataFile   = "VINE_HUB_SEED_DATA_FILE"
	envHubSeedSourceFile = "VINE_HUB_SEED_SOURCE_FILE"
	envHubSeedVarsFile   = "VINE_HUB_SEED_VARS_FILE"
	envHubAdminListen    = "VINE_HUB_ADMIN_LISTEN"
)

func (a *_App) initInfra() {
	flag := &hubflag.Flag{}
	appcli.Handle(flags(flag)...)
	applyOption(flag, a.option)

	a.hub = internalapp.NewInternalInproc[*hubapp.HubApp](internalapp.With(flag))
	a.link = internalapp.NewInternalInproc[*linkapp.LinkApp](internalapp.With(&linkflag.Flag{
		HubInprocMode: true,
	}))
	a.portal = internalapp.NewInternalInproc[*portalapp.PortalApp](internalapp.With(&portalflag.Flag{
		HubInprocMode: true,
	}))
}

// flags lists the Hub parameters the business binary accepts.
func flags(flag *hubflag.Flag) []ucli.Flag {
	return []ucli.Flag{
		&ucli.BoolFlag{
			Name:        flagHubNoDB,
			Sources:     ucli.EnvVars(envHubNoDB),
			Usage:       "use no persistent database (default); requires the seed data file or Option.SeedHubData; configuration is read-only",
			Destination: &flag.NoDB,
		},
		&ucli.StringFlag{
			Name:        flagHubDBSQLiteFile,
			Sources:     ucli.EnvVars(envHubDBSQLiteFile),
			Usage:       "in-process Hub SQLite database file",
			Destination: &flag.DBSQLiteFile,
		},
		&ucli.StringFlag{
			Name:        flagHubDBPostgresURL,
			Sources:     ucli.EnvVars(envHubDBPostgresURL),
			Usage:       "in-process Hub PostgreSQL database URL",
			Destination: &flag.DBPostgresURL,
		},
		&ucli.StringFlag{
			Name:        flagHubSeedDataFile,
			Sources:     ucli.EnvVars(envHubSeedDataFile),
			Usage:       "in-process Hub seed YAML file",
			Destination: &flag.SeedHubDataFile,
		},
		&ucli.StringFlag{
			Name:        flagHubSeedSourceFile,
			Sources:     ucli.EnvVars(envHubSeedSourceFile),
			Usage:       "in-process Hub seed source YAML file",
			Destination: &flag.SeedHubSourceFile,
		},
		&ucli.StringFlag{
			Name:        flagHubSeedVarsFile,
			Sources:     ucli.EnvVars(envHubSeedVarsFile),
			Usage:       "in-process Hub seed vars YAML file",
			Destination: &flag.SeedHubVarsFile,
		},
		&ucli.StringFlag{
			Name:        flagHubAdminListen,
			Sources:     ucli.EnvVars(envHubAdminListen),
			Usage:       "in-process Hub Admin API and Dashboard listen address; unauthenticated, so loopback unless the network is trusted",
			Destination: &flag.AdminListen,
		},
	}
}

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
