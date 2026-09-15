package spec

import (
	"reflect"
	"testing"
)

type _RegistryTestWebServer interface {
	Handler

	mustBeRegistryTestWebServer()
}

type _DefaultRegistryTestWebServer struct {
}

func (*_DefaultRegistryTestWebServer) Routes(*Router) {
	panic("method routes is not implemented")
}

func (*_DefaultRegistryTestWebServer) mustBeRegistryTestWebServer() {}

func TestRegisterStoresWebInfo(t *testing.T) {
	registry := NewRegistry()

	spec := &WebSpec{
		Name:              "RegistryTestWeb",
		SkelName:          "demo.user.RegistryTestWeb",
		ServerType:        reflect.TypeFor[_RegistryTestWebServer](),
		DefaultServerType: reflect.TypeFor[*_DefaultRegistryTestWebServer](),
	}

	registry.Register(spec)

	infos := registry.RegisteredWebInfos()
	if len(infos) != 1 {
		t.Fatalf("unexpected web info count: %d", len(infos))
	}
	if infos[0].Name() != "RegistryTestWeb" {
		t.Fatalf("unexpected web name: %s", infos[0].Name())
	}
	if infos[0].SkelName() != "demo.user.RegistryTestWeb" {
		t.Fatalf("unexpected web skel name: %s", infos[0].SkelName())
	}
}

func TestRegisterValidatesMountPath(t *testing.T) {
	valid := []string{"", "/", "/app", "/app/", "/app/%2Fdocs"}
	for _, mountPath := range valid {
		t.Run("valid "+mountPath, func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(testWebSpec(mountPath))
		})
	}
	invalid := []string{"//app", "https://example.com/app", "/app?x=1", "/app#section", "/./app", "/app/../admin", "/app path", "/app\\data"}
	for _, mountPath := range invalid {
		t.Run("invalid "+mountPath, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected mount path %q to be rejected", mountPath)
				}
			}()
			NewRegistry().Register(testWebSpec(mountPath))
		})
	}
}

func testWebSpec(mountPath string) *WebSpec {
	return &WebSpec{Name: "RegistryTestWeb", SkelName: "demo.user.RegistryTestWeb", MountPath: mountPath, ServerType: reflect.TypeFor[_RegistryTestWebServer](), DefaultServerType: reflect.TypeFor[*_DefaultRegistryTestWebServer]()}
}
