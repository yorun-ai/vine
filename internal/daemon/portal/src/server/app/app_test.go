package app

import (
	"reflect"
	"testing"

	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/entry"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
)

func collectComponentTypes(spec *PortalApp) []reflect.Type {
	var componentTypes []reflect.Type
	spec.InitComponents(func(componentType reflect.Type) {
		componentTypes = append(componentTypes, componentType)
	})
	return componentTypes
}

func collectModuleTypes(spec *PortalApp) []reflect.Type {
	var moduleTypes []reflect.Type
	spec.InitModules(func(moduleType reflect.Type) {
		moduleTypes = append(moduleTypes, moduleType)
	})
	return moduleTypes
}

func TestPortalAppDIInitSetsRunFlagListenAddr(t *testing.T) {
	spec := &PortalApp{
		InternalApplication: internalapp.InternalApplication{
			Application: internalapp.Application{
				AppFlag: &internalapp.RunFlag{},
			},
		},
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag:       &flag.Flag{HubEndpoint: "http://demo.local:7071"},
	}

	spec.DIInit()

	if got := spec.AppFlag.ListenAddr; got != "" {
		t.Fatalf("expected empty listen addr, got %q", got)
	}
	if spec.InternalAttrs.Linker == nil {
		t.Fatal("expected noop linker")
	}
	if got := spec.InternalAttrs.Linker.RpcProxyEndpoint(); got != "http://demo.local:7071/rpc/invoke" {
		t.Fatalf("unexpected hub redirect endpoint: %s", got)
	}
	if !spec.InternalAttrs.DisableConsole {
		t.Fatal("expected console to be disabled")
	}
	if !spec.InternalAttrs.DisableHTTPServer {
		t.Fatal("expected http server to be disabled")
	}
	if spec.InternalAttrs.InprocHostPath != "" {
		t.Fatalf("expected empty inproc host path, got %q", spec.InternalAttrs.InprocHostPath)
	}
}

func TestPortalAppDIInitAppendsRuntimeNameToInfoInInprocMode(t *testing.T) {
	spec := &PortalApp{
		InternalApplication: internalapp.InternalApplication{
			Application: internalapp.Application{
				AppFlag: &internalapp.RunFlag{},
			},
		},
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag:       &flag.Flag{HubInprocMode: true},
	}

	spec.DIInit()

	if got := spec.Name(); got != "vine.portal" {
		t.Fatalf("unexpected spec name: %s", got)
	}
	if got := spec.InternalAttrs.Info.Name(); got != "vine.portal@"+runtime.Application().Name() {
		t.Fatalf("unexpected app info name: %s", got)
	}
}

func TestPortalAppInitComponentsAddsHubRedisClient(t *testing.T) {
	spec := &PortalApp{}

	componentTypes := collectComponentTypes(spec)

	if len(componentTypes) != 2 {
		t.Fatalf("unexpected component count: %d", len(componentTypes))
	}
	if got := componentTypes[0]; got != internalapp.T[*hubinfo.HubInfo]() {
		t.Fatalf("unexpected first component type: %v", got)
	}
	if got := componentTypes[1]; got != internalapp.T[*hubredis.Client]() {
		t.Fatalf("unexpected second component type: %v", got)
	}
}

func TestPortalAppInitModulesAddsManagers(t *testing.T) {
	spec := &PortalApp{}

	moduleTypes := collectModuleTypes(spec)

	if len(moduleTypes) != 5 {
		t.Fatalf("unexpected module count: %d", len(moduleTypes))
	}
	if got := moduleTypes[0]; got != internalapp.T[*epmgr.Manager]() {
		t.Fatalf("unexpected module type: %v", got)
	}
	if got := moduleTypes[1]; got != internalapp.T[*access.Access]() {
		t.Fatalf("unexpected module type: %v", got)
	}
	if got := moduleTypes[2]; got != internalapp.T[*site.Manager]() {
		t.Fatalf("unexpected module type: %v", got)
	}
	if got := moduleTypes[3]; got != internalapp.T[*vault.Vault]() {
		t.Fatalf("unexpected module type: %v", got)
	}
	if got := moduleTypes[4]; got != internalapp.T[*entry.Manager]() {
		t.Fatalf("unexpected module type: %v", got)
	}
}
