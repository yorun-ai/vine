package admin

import (
	"bytes"
	_ "embed"
	"net/http"
	"path"
	"time"

	"go.yorun.ai/vine/internal/core/web/assets"
)

//go:embed assets/dashboard.tar.zst
var dashboardTarZst []byte

var dashboardAssets = assets.NewTarZstAccessor(dashboardTarZst)

const dashboardIndexPath = "/index.html"

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
