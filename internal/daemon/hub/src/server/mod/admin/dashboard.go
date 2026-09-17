package admin

import (
	"bytes"
	"context"
	_ "embed"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"sync/atomic"
	"time"

	"go.yorun.ai/vine/internal/core/web/assets"
)

//go:embed assets/dashboard.tar.zst
var dashboardTarZst []byte

var dashboardAssets = assets.NewTarZstAccessor(dashboardTarZst)

const (
	// dashboardIndexPath is the entry document of the embedded build, and the
	// document a path the build does not carry answers with.
	dashboardIndexPath = "/index.html"
	// dashboardDevProxyEnv starts Hub against a Dashboard development server
	// instead of the embedded build, so a Dashboard source change needs no build.
	dashboardDevProxyEnv = "VINE_HUB_DASHBOARD_DEV_PROXY"
	// dashboardDevProxyProbeInterval is how often the development server is
	// checked for an answer: the Dashboard asks for its build often, so the check
	// runs beside the requests and a request only reads its result.
	dashboardDevProxyProbeInterval = time.Second
	// dashboardDevProxyDialTimeout bounds one check of the development server.
	dashboardDevProxyDialTimeout = 100 * time.Millisecond
)

// dashboardDevServerURL is the development server script/dev-hub-dashboard.sh
// starts: it fails when Vite cannot take this port, so Hub always proxies the
// server the script started.
var dashboardDevServerURL = "http://localhost:7098"

// dashboardDevProxy returns the development server a development Hub serves the
// Dashboard through, or nil when no development server was asked for.
func dashboardDevProxy() *_DashboardDevProxy {
	if os.Getenv(dashboardDevProxyEnv) == "" {
		return nil
	}
	target, err := url.Parse(dashboardDevServerURL)
	if err != nil {
		adminLogger.Error("dashboard development proxy target is invalid",
			"target", dashboardDevServerURL, "error", err)
		return nil
	}
	devProxy := newDashboardDevProxy(target)
	adminLogger.Info("dashboard development proxy enabled",
		"target", dashboardDevServerURL, "env", dashboardDevProxyEnv, "answering", devProxy.available.Load())
	return devProxy
}

// _DashboardDevProxy serves the Dashboard from the development server a developer
// runs beside Hub, and reports the requests it did not answer, so Hub serves them
// from the embedded build instead. A check beside the requests keeps its answer
// current, so a request never waits for a connection of its own.
type _DashboardDevProxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy

	context context.Context
	cancel  context.CancelFunc

	available atomic.Bool
}

func newDashboardDevProxy(target *url.URL) *_DashboardDevProxy {
	devProxy := &_DashboardDevProxy{
		target: target,
		proxy: &httputil.ReverseProxy{
			Rewrite: func(request *httputil.ProxyRequest) {
				request.SetURL(target)
				request.Out.Host = target.Host
			},
		},
	}
	devProxy.context, devProxy.cancel = context.WithCancel(context.Background())
	devProxy.available.Store(devProxy.detectAvailable())
	go devProxy.watch()
	return devProxy
}

// ServeHTTP forwards request to the development server while it answers, and
// reports whether it did.
func (p *_DashboardDevProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) bool {
	if !p.available.Load() {
		return false
	}

	requestContext, release := p.requestContext(request)
	defer release()
	p.proxy.ServeHTTP(writer, request.WithContext(requestContext))
	return true
}

// Close cancels the requests the development server did not answer, so a request
// it never answers cannot hold the listener open while Hub stops.
func (p *_DashboardDevProxy) Close() {
	p.cancel()
}

// requestContext returns the context one forwarded request runs under: it ends
// with the request, and with the proxy.
func (p *_DashboardDevProxy) requestContext(request *http.Request) (context.Context, func()) {
	ctx, cancel := context.WithCancel(request.Context())
	stop := context.AfterFunc(p.context, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

func (p *_DashboardDevProxy) watch() {
	ticker := time.NewTicker(dashboardDevProxyProbeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.context.Done():
			return
		case <-ticker.C:
			p.refreshAvailable()
		}
	}
}

// refreshAvailable records whether the development server answers, and reports the
// change: a developer who sees the embedded build wants to know the development
// server stopped answering, and the other way around.
func (p *_DashboardDevProxy) refreshAvailable() {
	available := p.detectAvailable()
	if p.available.Swap(available) == available {
		return
	}
	if available {
		adminLogger.Info("dashboard development proxy is serving the development server", "target", p.target.String())
		return
	}
	adminLogger.Info("dashboard development proxy is not answering, Hub serves the embedded Dashboard", "target", p.target.String())
}

// detectAvailable reports whether the development server accepts a connection.
func (p *_DashboardDevProxy) detectAvailable() bool {
	conn, err := net.DialTimeout("tcp", p.target.Host, dashboardDevProxyDialTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// dashboardHandler serves the embedded Dashboard build on the admin listener,
// which also serves the Admin API, so the Dashboard reaches Hub on one origin
// instead of another component's route.
func dashboardHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		name := path.Clean("/" + request.URL.Path)
		if file, ok := dashboardAssets.Open(name, nil); ok {
			serveDashboardFile(writer, request, name, file.ModTime, file.Content)
			return
		}
		// The Dashboard is a single-page app: a path the build does not carry is
		// one of its own routes, so Hub answers with the entry document and lets
		// the app resolve it.
		file, ok := dashboardAssets.Open(dashboardIndexPath, nil)
		if !ok {
			http.NotFound(writer, request)
			return
		}
		serveDashboardFile(writer, request, dashboardIndexPath, file.ModTime, file.Content)
	})
}

func serveDashboardFile(writer http.ResponseWriter, request *http.Request, name string, modTime time.Time, content []byte) {
	// ServeContent deduces the content type from the name and answers HEAD and
	// conditional requests.
	http.ServeContent(writer, request, path.Base(name), modTime, bytes.NewReader(content))
}
