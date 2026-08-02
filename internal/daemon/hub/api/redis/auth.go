package redis

import (
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/daemon"
)

const (
	HubUsername    = "vine.hub"
	LinkUsername   = "vine.link"
	PortalUsername = "vine.portal"

	// LinkPassword and PortalPassword are intentionally empty for inproc mode and
	// debugging separated deployments. In production, Hub Redis mTLS binds each
	// username to the matching vine.link or vine.portal certificate identity;
	// without mTLS these empty passwords still provide authorization but not
	// caller authentication, so the Redis endpoint must remain on a trusted network.
	LinkPassword   = ""
	PortalPassword = ""
)

// SPIFFEPathForUsername returns the daemon identity bound to a Hub Redis ACL
// username.
func SPIFFEPathForUsername(username string) (mtls.SPIFFEPath, bool) {
	switch username {
	case HubUsername:
		return daemon.HubIdentity.SPIFFEPath(), true
	case LinkUsername:
		return daemon.LinkIdentity.SPIFFEPath(), true
	case PortalUsername:
		return daemon.PortalIdentity.SPIFFEPath(), true
	default:
		return "", false
	}
}
