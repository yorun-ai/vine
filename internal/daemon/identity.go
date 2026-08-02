package daemon

import "go.yorun.ai/vine/internal/core/mtls"

// Identity names a Vine daemon workload.
type Identity string

const (
	// HubIdentity identifies the Vine Hub daemon.
	HubIdentity Identity = "vine.hub"
	// LinkIdentity identifies the Vine Link daemon.
	LinkIdentity Identity = "vine.link"
	// PortalIdentity identifies the Vine Portal daemon.
	PortalIdentity Identity = "vine.portal"
)

// String returns the daemon application name.
func (i Identity) String() string {
	return string(i)
}

// SPIFFEPath returns the daemon's workload path. An empty identity remains
// empty so missing identities cannot be mistaken for a valid workload.
func (i Identity) SPIFFEPath() mtls.SPIFFEPath {
	if i == "" {
		return ""
	}
	return mtls.SPIFFEPath("/vine/daemon/" + i)
}
