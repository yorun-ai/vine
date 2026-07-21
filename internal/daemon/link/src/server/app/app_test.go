package app

import (
	"fmt"
	"strings"
	"testing"

	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func TestLinkAppDIInitAppendsRuntimeNameToInfoInInprocMode(t *testing.T) {
	spec := &LinkApp{
		InternalApplication: internalapp.InternalApplication{
			Application: internalapp.Application{
				AppFlag: &internalapp.RunFlag{},
			},
		},
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag: &flag.Flag{
			HubInprocMode: true,
		},
	}

	spec.DIInit()

	if got := spec.Name(); got != "vine.link" {
		t.Fatalf("unexpected spec name: %s", got)
	}
	if got := spec.InternalAttrs.Info.Name(); got != "vine.link@"+runtime.Application().Name() {
		t.Fatalf("unexpected app info name: %s", got)
	}
	if got := spec.AppFlag.ListenAddr; got != "" {
		t.Fatalf("unexpected app flag listen addr: %s", got)
	}
}

func TestLinkAppStartDoesNotPanicOnHubPubClientBinding(t *testing.T) {
	app := internalapp.NewInternal[*LinkApp](
		internalapp.With(&flag.Flag{
			APIListen:     "127.0.0.1:0",
			IngressListen: "127.0.0.1:0",
			HubEndpoint:   "http://127.0.0.1:7071",
			HubInprocMode: true,
		}),
	)

	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		message := fmt.Sprint(recovered)
		if strings.Contains(message, "implicit binding only supports struct pointer, but skeled.InfoServiceClient found") {
			t.Fatalf("unexpected hub pub client binding panic: %v", recovered)
		}
	}()

	func() {
		app.Start()
		app.StopGracefully()
	}()
}
