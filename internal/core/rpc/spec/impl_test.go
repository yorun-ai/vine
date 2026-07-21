package spec

import (
	"reflect"
	"strings"
	"testing"
)

type testImplServer interface {
	Ping()

	mustBeTestImplServer()
}

type defaultTestImplServer struct{}

func (*defaultTestImplServer) Ping() {}

func (*defaultTestImplServer) mustBeTestImplServer() {}

type testImplServerER interface {
	Ping() error

	mustBeTestImplServerER()
}

type defaultTestImplServerER struct{}

func (*defaultTestImplServerER) Ping() error { return nil }

func (*defaultTestImplServerER) mustBeTestImplServerER() {}

type implServerImpl struct {
	defaultTestImplServer
}

type implERServerImpl struct {
	defaultTestImplServerER
}

type testImplPubServer interface {
	Ping()

	mustBeTestImplPubServer()
}

type defaultTestImplPubServer struct{}

func (*defaultTestImplPubServer) Ping() {}

func (*defaultTestImplPubServer) mustBeTestImplPubServer() {}

type testImplPubServerER interface {
	Ping() error

	mustBeTestImplPubServerER()
}

type defaultTestImplPubServerER struct{}

func (*defaultTestImplPubServerER) Ping() error { return nil }

func (*defaultTestImplPubServerER) mustBeTestImplPubServerER() {}

type _InvalidImplValueType struct {
	defaultTestImplServer
}

var (
	_implServiceSpec = &ServiceSpec{
		Type:     ServiceSpecTypeServer,
		Name:     "ImplService",
		SkelName: "rpc.spec.implService",

		ServerType:        reflect.TypeOf((*testImplServer)(nil)).Elem(),
		DefaultServerType: reflect.TypeOf(&defaultTestImplServer{}),

		ERServerType:        reflect.TypeOf((*testImplServerER)(nil)).Elem(),
		DefaultERServerType: reflect.TypeOf(&defaultTestImplServerER{}),

		Methods: []*MethodSpec{{
			Name:     "Ping",
			SkelName: "ping",
		}},
	}
	_implPubServiceSpec = &ServiceSpec{
		Type:         ServiceSpecTypeClient,
		Name:         "ImplServicePub",
		SkelName:     _implServiceSpec.SkelName,
		ClientType:   reflect.TypeOf((*testImplPubServer)(nil)).Elem(),
		ERClientType: reflect.TypeOf((*testImplPubServerER)(nil)).Elem(),
		Methods: []*MethodSpec{{
			Name:     "Ping",
			SkelName: "ping",
		}},
	}
)

func init() {
	Register(_implServiceSpec)
	Register(_implPubServiceSpec)
}

func TestImplDictAddAddsServiceAndMethods(t *testing.T) {
	dict := NewImplDict()

	dict.Add(reflect.TypeOf(&implServerImpl{}))

	var serviceImpl ServiceImpl
	dict.IterateServiceImpl(func(info ServiceImpl) {
		if info.Info().SkelName() == _implServiceSpec.SkelName {
			serviceImpl = info
		}
	})
	if serviceImpl == nil {
		t.Fatal("expected service impl to be registered")
	}
	if serviceImpl.Info().SkelName() != _implServiceSpec.SkelName {
		t.Fatalf("unexpected service info: %#v", serviceImpl.Info())
	}
	if serviceImpl.Type() != reflect.TypeOf(&implServerImpl{}) {
		t.Fatalf("unexpected impl type: %v", serviceImpl.Type())
	}
	methodImpl, err := serviceImpl.MethodImpl(_implServiceSpec.Methods[0].SkelName)
	if err != nil {
		t.Fatalf("expected method impl for %s", _implServiceSpec.Methods[0].SkelName)
	}
	if methodImpl.Info().SkelName() != _implServiceSpec.Methods[0].SkelName {
		t.Fatalf("unexpected method info: %#v", methodImpl.Info())
	}
	if methodImpl.Method().Name != _implServiceSpec.Methods[0].Name {
		t.Fatalf("unexpected reflected method: %s", methodImpl.Method().Name)
	}
	if methodImpl.IsERType() {
		t.Fatal("expected non-er method impl")
	}
}

func TestImplDictAddMarksERType(t *testing.T) {
	dict := NewImplDict()

	dict.Add(reflect.TypeOf(&implERServerImpl{}))

	methodImpl, err := dict.GetMethodImpl(_implServiceSpec.SkelName, _implServiceSpec.Methods[0].SkelName)
	if err != nil {
		t.Fatalf("GetMethodImpl() error = %v", err)
	}
	if !methodImpl.IsERType() {
		t.Fatalf("expected er method impl, got %#v", methodImpl)
	}

	var serviceImpl ServiceImpl
	dict.IterateServiceImpl(func(info ServiceImpl) {
		if info.Info().SkelName() == _implServiceSpec.SkelName {
			serviceImpl = info
		}
	})
	if serviceImpl == nil || !serviceImpl.IsERType() {
		t.Fatalf("expected er service impl, got %#v", serviceImpl)
	}
}

func TestImplDictGetMethodImpl(t *testing.T) {
	dict := NewImplDict()
	dict.Add(reflect.TypeOf(&implServerImpl{}))

	methodImpl, err := dict.GetMethodImpl(_implServiceSpec.SkelName, _implServiceSpec.Methods[0].SkelName)
	if err != nil {
		t.Fatalf("GetMethodImpl() error = %v", err)
	}
	if methodImpl.Info().SkelName() != _implServiceSpec.Methods[0].SkelName {
		t.Fatalf("unexpected method info: %#v", methodImpl.Info())
	}
}

func TestImplDictGetMethodImplByInfo(t *testing.T) {
	dict := NewImplDict()
	dict.Add(reflect.TypeOf(&implServerImpl{}))

	methodImpl, err := dict.GetMethodImplByInfo(_implServiceSpec.Methods[0].Info())
	if err != nil {
		t.Fatalf("GetMethodImplByInfo() error = %v", err)
	}
	if methodImpl.Info().SkelName() != _implServiceSpec.Methods[0].SkelName {
		t.Fatalf("unexpected method info: %#v", methodImpl.Info())
	}
}

func TestImplDictGetMethodImplReturnsErrors(t *testing.T) {
	dict := NewImplDict()
	dict.Add(reflect.TypeOf(&implServerImpl{}))

	_, err := dict.GetMethodImpl("missing.service", _implServiceSpec.Methods[0].SkelName)
	if err == nil || !strings.Contains(err.Error(), "service missing.service not found") {
		t.Fatalf("unexpected missing service error: %v", err)
	}

	_, err = dict.GetMethodImpl(_implServiceSpec.SkelName, "missingMethod")
	if err == nil || !strings.Contains(err.Error(), "method missingMethod not found") {
		t.Fatalf("unexpected missing method error: %v", err)
	}

	_, err = dict.GetMethodImplByInfo(nil)
	if err == nil || !strings.Contains(err.Error(), "method info is nil") {
		t.Fatalf("unexpected nil method info error: %v", err)
	}

	_, err = dict.GetMethodImplByInfo(ConvertSpecToInfoForTest(&ServiceSpec{
		Name:     "MissingService",
		SkelName: "missing.service",
		Methods: []*MethodSpec{{
			Name:     "Missing",
			SkelName: "missing",
		}},
	}).Methods()[0])
	if err == nil || !strings.Contains(err.Error(), "method missing not found") {
		t.Fatalf("unexpected missing method info error: %v", err)
	}
}

func TestImplDictAddPanicsOnDuplicateService(t *testing.T) {
	dict := NewImplDict()
	dict.Add(reflect.TypeOf(&implServerImpl{}))

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate add panic")
		}
		if !strings.Contains(recovered.(error).Error(), "service rpc.spec.implService already added") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	dict.Add(reflect.TypeOf(&implServerImpl{}))
}

func TestImplDictAddRejectsNonPointerStruct(t *testing.T) {
	dict := NewImplDict()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected invalid impl type panic")
		}
		if !strings.Contains(recovered.(error).Error(), "rpc impl type spec._InvalidImplValueType must be a pointer to struct") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	dict.Add(reflect.TypeOf(_InvalidImplValueType{}))
}
