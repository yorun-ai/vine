package admin

import (
	"context"
	"embed"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/web/assets"
)

// Build Dashboard resources before compiling; dist must contain real assets.
//
//go:embed all:dashboard/dist
var dashboardFS embed.FS

var dashboardAssets = assets.NewEmbedAccessor(dashboardFS, "dashboard/dist")

const (
	// dashboardDevProxyEnv enables probing and preferring the local Dashboard server.
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

// dashboardDevProxy starts probing only when explicitly enabled by the environment.
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
		"target", dashboardDevServerURL, "answering", devProxy.available.Load())
	return devProxy
}

// _DashboardDevProxy serves the Dashboard from the development server a developer
// runs beside Hub, and reports the requests it did not answer, so Hub serves
// the embedded build instead. A check beside the requests keeps its answer
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
// change so developers know when the local Dashboard becomes available.
func (p *_DashboardDevProxy) refreshAvailable() {
	available := p.detectAvailable()
	if p.available.Swap(available) == available {
		return
	}
	if available {
		adminLogger.Info("dashboard development proxy is serving the development server", "target", p.target.String())
		return
	}
	adminLogger.Info("dashboard development server is unavailable; run script/dev-hub-dashboard.sh", "target", p.target.String())
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

	// Dashboard is served by its own Gin router instead of Vine's shared Web
	// server, so apply the same production mode before registering routes.
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Any("/*path", func(ctx *gin.Context) {
		// Asset Server stores request state, while the accessor can be shared.
		server := assets.NewServer(dashboardAssets)
		server.SetContext(ctx)
		server.Serve()
	})
	return router
}
