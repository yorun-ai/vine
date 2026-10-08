package admin

import (
	"go.yorun.ai/vine/buildinfo"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
)

type AdminApiServiceServerImpl struct {
	skeled.DefaultAdminApiServiceServer

	Access *configaccess.Access `inject:""`
}

// ReadOnly reports whether Hub serves a configuration it cannot write, so the
// Dashboard tells the operator that the configuration source decides.
func (s *AdminApiServiceServerImpl) ReadOnly() bool {
	return s.Access.ReadOnly()
}

// Version reports the Vine runtime version of the running Hub.
func (s *AdminApiServiceServerImpl) Version() string {
	return buildinfo.MustVineVersion()
}
