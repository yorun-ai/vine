package webproxy

import (
	"context"
	"testing"

	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

func TestOnDrainRemovesLocalAppState(t *testing.T) {
	proxy := &WebProxy{
		Context:   context.Background(),
		AppMinder: newTestAppMinder(),
	}
	proxy.DIInit()

	localApp := mustWebProxyMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111111")
	registerLocalWebApp(proxy, localApp, "http://127.0.0.1:8080"+testPathWebAccess, "http://127.0.0.1:8080", []string{"admin@demo.app"})

	proxy.OnDrain(&minder.AppInstance{AppInfo: localApp})

	if _, ok := proxy.webRouteByInstanceID(localApp.InstanceId(), "admin@demo.app"); ok {
		t.Fatal("expected local web route to be removed")
	}
}
