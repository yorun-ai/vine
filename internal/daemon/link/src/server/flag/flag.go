package flag

import (
	"net"
	"net/url"
	"strings"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/mtls"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	"go.yorun.ai/vine/util/vpre"
)

var linkFlagLogger = logger.New("daemon:link:flag")

const (
	LinkDefaultAPIListen     = "127.0.0.1:7079"
	LinkDefaultIngressListen = "0.0.0.0:0"
)

type Flag struct {
	app.FlagModel
	MTLS           mtls.Files
	APIListen      string
	IngressListen  string
	HubInprocMode  bool
	HubEndpoint    string
	HubEndpointURL *url.URL
}

func (f *Flag) Normalize(linkInproc bool) {
	vpre.CheckNilError(f.MTLS.Validate(), "link flag normalize failed")
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
	if f.MTLS.Enabled() {
		vpre.Check(parsed.Scheme == "https", "hub-endpoint must use https when mTLS is enabled")
	}
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
	if !isExpectedAppAPIListen(f.APIListen) {
		linkFlagLogger.Warn(
			"link API listens outside loopback; cross-host App-to-Link traffic is allowed but is not the expected sidecar topology and remains unauthenticated h2c",
			"listen", f.APIListen,
		)
	}
}

func isExpectedAppAPIListen(listen string) bool {
	host, _, err := net.SplitHostPort(listen)
	if err != nil || host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
