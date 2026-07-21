package flag

import (
	"net/url"

	"go.yorun.ai/vine/internal/app"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	"go.yorun.ai/vine/util/vpre"
)

type Flag struct {
	app.FlagModel
	HubInprocMode  bool
	HubEndpoint    string
	HubEndpointURL *url.URL
}

func (o *Flag) Normalize() {
	if o.HubInprocMode {
		o.HubEndpoint = hubapp.HubInprocEndpoint
	}
	vpre.CheckNotEmpty(o.HubEndpoint, "hub-endpoint is empty")
	parsed, err := url.Parse(o.HubEndpoint)
	vpre.CheckNilError(err, "hub-endpoint is invalid")
	if !o.HubInprocMode {
		vpre.Check(parsed.Hostname() != "", "hub-endpoint host is empty")
	}
	o.HubEndpointURL = parsed
}
