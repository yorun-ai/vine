package redis

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
