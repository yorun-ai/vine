package dashboard

import (
	_ "embed"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/core/web"
	"go.yorun.ai/vine/internal/core/web/assets"
	"go.yorun.ai/vine/internal/core/web/proxy"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
)

const (
	dashboardProxyDialTimeout       = 100 * time.Millisecond
	dashboardProxyDetectionInterval = time.Second
	envDashboardDevProxy            = "VINE_HUB_DASHBOARD_DEV_PROXY"
)

//go:embed assets/dashboard.tar.zst
var dashboardTarZst []byte

var (
	dashboardAssetsAccessor = assets.NewTarZstAccessor(dashboardTarZst)
	dashboardProxy          = proxy.NewReverseProxy(proxy.Option{
		Target: &url.URL{
			Scheme: "http",
			Host:   "localhost:7098",
		},
		DialTimeout:       dashboardProxyDialTimeout,
		DetectionInterval: dashboardProxyDetectionInterval,
	})
)

type WebServerImpl struct {
	skeled.DefaultDashboardWebServer

	GinCtx *gin.Context `inject:""`

	assetsServer assets.Server
}

func (h *WebServerImpl) Routes(r *web.Router) {
	r.ANY("/*path", h.ProxyDashboard)
}

func (h *WebServerImpl) ProxyDashboard() {
	if !h.proxyDashboard() {
		h.assetsServer.ServeAsset(h.GinCtx, dashboardAssetsAccessor)
	}
}

func (h *WebServerImpl) proxyDashboard() bool {
	if dashboardDevProxyEnabled() {
		return dashboardProxy.Serve(h.GinCtx)
	}
	return false
}

func dashboardDevProxyEnabled() bool {
	return os.Getenv(envDashboardDevProxy) != ""
}
