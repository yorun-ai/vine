package app

import rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"

const (
	// HubControlInprocHostPath is kept at the original Hub inproc address so
	// Link and Portal continue to reach the component-facing Control API.
	HubControlInprocHostPath = "vine/hub"
	HubControlInprocEndpoint = rpcinproc.EndpointScheme + HubControlInprocHostPath

	// HubAdminInprocHostPath isolates Dashboard admin Rpc and Web
	// handlers from the component-facing Control API inside standalone mode.
	HubAdminInprocHostPath = "vine/hub/admin"

	// HubInprocHostPath and HubInprocEndpoint retain their existing names for
	// callers that treat the Hub endpoint as the Control API endpoint.
	HubInprocHostPath = HubControlInprocHostPath
	HubInprocEndpoint = HubControlInprocEndpoint
)
