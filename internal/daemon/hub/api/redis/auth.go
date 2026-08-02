package redis

const (
	HubUsername    = "vine.hub"
	LinkUsername   = "vine.link"
	PortalUsername = "vine.portal"

	// LinkPassword and PortalPassword are intentionally empty for inproc mode and
	// debugging separated deployments. The distinct usernames select separate
	// least-privilege ACLs, but empty passwords do not authenticate the caller.
	// TODO: Production deployments must authenticate Hub Redis connections with
	// mutual TLS (mTLS) before the endpoint can be exposed outside a trusted
	// network.
	LinkPassword   = ""
	PortalPassword = ""
)
