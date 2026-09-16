package app

import rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"

const (
	// HubControlInprocHostPath is kept at the original Hub inproc address so
	// Link and Portal continue to reach the component-facing Control API.
	HubControlInprocHostPath = "vine/hub"
	HubControlInprocEndpoint = rpcinproc.EndpointScheme + HubControlInprocHostPath

	// HubInprocEndpoint retains its existing name for callers that treat the Hub
	// endpoint as the Control API endpoint.
	HubInprocEndpoint = HubControlInprocEndpoint
)
