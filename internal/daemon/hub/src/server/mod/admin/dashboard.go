package admin

import (
	"bytes"
	_ "embed"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"go.yorun.ai/vine/internal/core/web/assets"
	"go.yorun.ai/vine/internal/core/web/proxy"
)

//go:embed assets/dashboard.tar.zst
var dashboardTarZst []byte

var dashboardAssets = assets.NewTarZstAccessor(dashboardTarZst)

const dashboardIndexPath = "/index.html"

// dashboardDevProxyEnv starts Hub against a Dashboard development server
// instead of the embedded build, so a Dashboard source change needs no build.
const dashboardDevProxyEnv = "VINE_HUB_DASHBOARD_DEV_PROXY"

// dashboardDevServerURL is the development server script/dev-hub-dashboard.sh
// starts: it fails when Vite cannot take this port, so Hub always proxies the
// server the script started.
var dashboardDevServerURL = "http://localhost:7098"

// DashboardDevProxy returns the reverse proxy a development Hub serves the
// Dashboard through, or nil when no development server was asked for.
func DashboardDevProxy() *proxy.ReverseProxy {
	if os.Getenv(dashboardDevProxyEnv) == "" {
		return nil
	}
	target, err := url.Parse(dashboardDevServerURL)
	if err != nil {
		adminLogger.Error("dashboard development proxy target is invalid",
			"target", dashboardDevServerURL, "error", err)
		return nil
	}
	adminLogger.Info("dashboard development proxy enabled",
		"target", dashboardDevServerURL, "env", dashboardDevProxyEnv)
	return proxy.NewReverseProxy(proxy.Option{Target: target})
}

// DashboardHandler serves the embedded Dashboard build on the admin listener,
// which also serves the Admin API, so the Dashboard reaches Hub on one origin
// instead of another component's route.
func DashboardHandler() http.Handler {
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
