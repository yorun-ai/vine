package admin

import (
	"context"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/web/assets"
)

// The empty .gitkeep keeps source-only builds valid. Packaging preserves it so
// generating assets does not modify tracked files or release VCS metadata.
//
//go:embed all:assets/dashboard
var dashboardFS embed.FS

var dashboardAssets = newDashboardAssets(dashboardFS)

func newDashboardAssets(fsys fs.FS) assets.Accessor {
	const root = "assets/dashboard"
	for _, index := range []string{"index.html", "index.html.br"} {
		if info, err := fs.Stat(fsys, root+"/"+index); err == nil && info.Mode().IsRegular() {
			return assets.NewEmbedAccessor(fsys, root)
		}
	}
	return nil
}

const (
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

// dashboardDevProxy returns a development proxy only when this build has no
// embedded Dashboard entry document. A placeholder-only build probes Vite;
// compiling after packaging automatically uses the embedded Dashboard.
func dashboardDevProxy() *_DashboardDevProxy {
	if dashboardAssets != nil {
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
// runs beside Hub, and reports the requests it did not answer, so Hub returns
// the development command instead. A check beside the requests keeps its answer
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
	if dashboardAssets == nil {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Error(writer, "Hub Dashboard assets are not embedded; run bash script/dev-hub-dashboard.sh for local development.", http.StatusNotFound)
		})
	}

	router := gin.New()
	router.Any("/*path", func(ctx *gin.Context) {
		if path.Clean("/"+ctx.Param("path")) == "/.gitkeep" {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		// Asset Server stores request state, while the accessor can be shared.
		server := assets.NewServer(dashboardAssets)
		server.SetContext(ctx)
		server.Serve()
	})
	return router
}
