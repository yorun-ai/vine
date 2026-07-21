package flag

import (
	"net/url"

	"go.yorun.ai/vine/internal/app"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	"go.yorun.ai/vine/util/vpre"
)

const (
	LinkDefaultAPIListen     = "127.0.0.1:7079"
	LinkDefaultIngressListen = "0.0.0.0:0"
)

type Flag struct {
	app.FlagModel
	APIListen      string
	IngressListen  string
	HubInprocMode  bool
	HubEndpoint    string
	HubEndpointURL *url.URL
}

func (f *Flag) Normalize(linkInproc bool) {
	f.normalizeHubEndpoint()
	f.normalizeAPIListen(linkInproc)
	f.normalizeIngressListen()
}

func (f *Flag) normalizeHubEndpoint() {
	if f.HubInprocMode {
		f.HubEndpoint = hubapp.HubInprocEndpoint
		return
	}

	vpre.CheckNotEmpty(f.HubEndpoint, "hub-endpoint is empty")
	parsed, err := url.Parse(f.HubEndpoint)
	vpre.CheckNilError(err, "hub-endpoint is invalid")
	vpre.Check(parsed.Hostname() != "", "hub-endpoint host is empty")
	f.HubEndpointURL = parsed
}

func (f *Flag) normalizeAPIListen(linkInproc bool) {
	if linkInproc {
		// Inproc link exposes its Rpc services through rpc+inproc, so the
		// external API listen address must not leak into runtime info.
		f.APIListen = ""
		return
	}

	if f.APIListen == "" {
		f.APIListen = LinkDefaultAPIListen
	}
}

func (f *Flag) normalizeIngressListen() {
	if f.HubInprocMode {
		// When Hub is inproc, Link registers ingress through inproc transport
		// instead of exposing an external HTTP ingress listener.
		f.IngressListen = ""
		return
	}

	if f.IngressListen == "" {
		f.IngressListen = LinkDefaultIngressListen
	}
}
