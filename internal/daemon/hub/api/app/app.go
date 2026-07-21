package app

import rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"

const (
	HubInprocHostPath = "vine/hub"
	HubInprocEndpoint = rpcinproc.EndpointScheme + HubInprocHostPath
)
