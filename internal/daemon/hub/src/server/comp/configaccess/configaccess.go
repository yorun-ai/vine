package configaccess

import (
	"sync"
	"sync/atomic"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
)

type Access struct {
	app.BaseComponent

	// PortalRouteWrites orders entry, rule and site changes, including full site
	// refreshes, through Watch publication. It is shared by all Hub requests.
	PortalRouteWrites sync.Mutex

	readOnly atomic.Bool
}

func (a *Access) Lock() {
	a.readOnly.Store(true)
}

func (a *Access) ReadOnly() bool {
	return a.readOnly.Load()
}

func (a *Access) CheckWrite() {
	ex.PanicNewIfNot(!a.ReadOnly(), ex.PermissionDenied, "Configuration is read-only; update the configuration source and restart Hub.")
}
