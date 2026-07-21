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
	ResetRegistryForTest()

	spec := &WebSpec{
		Name:              "RegistryTestWeb",
		SkelName:          "demo.user.RegistryTestWeb",
		ServerType:        reflect.TypeFor[_RegistryTestWebServer](),
		DefaultServerType: reflect.TypeFor[*_DefaultRegistryTestWebServer](),
	}

	Register(spec)

	infos := RegisteredWebInfos()
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
