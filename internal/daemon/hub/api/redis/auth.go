package redis

const (
	HubUsername    = "vine.hub"
	LinkUsername   = "vine.link"
	PortalUsername = "vine.portal"

	// LinkPassword and PortalPassword are intentionally empty only as a temporary
	// migration step. The distinct usernames let Hub Redis enforce separate
	// least-privilege ACLs, but empty passwords do not authenticate the caller:
	// any client that can reach the Redis endpoint can still impersonate Link or
	// Portal. Replace them with deployment-provided secrets before treating the
	// Redis endpoint as safe on an untrusted network.
	LinkPassword   = ""
	PortalPassword = ""
)
