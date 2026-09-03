package spec

import (
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/util/reflectutil"
)

type testRegistryServer interface {
	Ping()
	mustBeTestRegistryServer()
}

type defaultTestRegistryServer struct{}

func (*defaultTestRegistryServer) mustBeTestRegistryServer() {}
func (*defaultTestRegistryServer) Ping()                     {}

type testRegistryServerER interface {
	Ping() error
	mustBeTestRegistryServerER()
}

type defaultTestRegistryServerER struct{}

func (*defaultTestRegistryServerER) mustBeTestRegistryServerER() {}
func (*defaultTestRegistryServerER) Ping() error                 { return nil }

type testRegistryServerImpl struct {
	defaultTestRegistryServer
}

type testRegistryServerERImpl struct {
	defaultTestRegistryServerER
}

type pubDefaultTestRegistryServer struct{}

func (*pubDefaultTestRegistryServer) mustBeTestRegistryServer() {}
func (*pubDefaultTestRegistryServer) Ping()                     {}

type pubDefaultTestRegistryServerER struct{}

func (*pubDefaultTestRegistryServerER) mustBeTestRegistryServerER() {}
func (*pubDefaultTestRegistryServerER) Ping() error                 { return nil }

type testRegistryPubServerImpl struct {
	pubDefaultTestRegistryServer
}

type testRegistryPubServerERImpl struct {
	pubDefaultTestRegistryServerER
}

type embeddedTypeA struct{}

type embeddedTypeB struct{}

type embeddedTypesHolder struct {
	embeddedTypeA
	embeddedTypeB
	Named string
	Alias embeddedTypeA
}

type testRegistryCloneArguments struct {
	Value string `arg:"0"`
}

func newRegistryTestServiceInfo(skelName string) *ServiceSpec {
	return &ServiceSpec{
		Type:                ServiceSpecTypeBoth,
		Name:                "TestRegistryService",
		SkelName:            skelName,
		ServerType:          reflect.TypeOf((*testRegistryServer)(nil)).Elem(),
		DefaultServerType:   reflect.TypeOf(&defaultTestRegistryServer{}),
		ClientType:          reflect.TypeOf((*testRegistryServer)(nil)).Elem(),
		ClientCtor:          func() {},
		ERServerType:        reflect.TypeOf((*testRegistryServerER)(nil)).Elem(),
		DefaultERServerType: reflect.TypeOf(&defaultTestRegistryServerER{}),
		ERClientType:        reflect.TypeOf((*testRegistryServerER)(nil)).Elem(),
		ERClientCtor:        func() {},
		Methods: []*MethodSpec{{
			Name:     "Ping",
			SkelName: "ping",
		}},
	}
}

func containsFactory(factories []any, wantFactory any) bool {
	wantPointer := reflect.ValueOf(wantFactory).Pointer()
	for _, factory := range factories {
		if reflect.ValueOf(factory).Pointer() == wantPointer {
			return true
		}
	}
	return false
}

func TestRegisterAddsServiceInfoAndInitializesMethods(t *testing.T) {
	registry := NewRegistry()

	serviceInfo := newRegistryTestServiceInfo("test.registry")
	registry.Register(serviceInfo)

	gotInfo := registry.serviceInfoBySkelName[serviceInfo.SkelName]
	if gotInfo.SkelName() != serviceInfo.SkelName {
		t.Fatalf("unexpected registered service: got %s want %s", gotInfo.SkelName(), serviceInfo.SkelName)
	}
	if gotInfo.Methods()[0].Service() != gotInfo {
		t.Fatalf("expected method service to be initialized")
	}
	if gotInfo.Methods()[0].FullURLPath() != "/test.registry/ping" {
		t.Fatalf("unexpected method full url path: %s", gotInfo.Methods()[0].FullURLPath())
	}
}

func TestRegisterMapsMethodPointerToSkelNames(t *testing.T) {
	registry := NewRegistry()

	serviceInfo := newRegistryTestServiceInfo("test.registry.methodPointer")
	serviceInfo.Methods[0].MethodFuncs = []any{
		testRegistryServer.Ping,
		testRegistryServerER.Ping,
	}
	registry.Register(serviceInfo)

	methodPointer := reflect.ValueOf(testRegistryServerER.Ping).Pointer()
	serviceSkelName, methodSkelName, ok := registry.GetMethodSkelNamesByPointer(methodPointer)
	if !ok {
		t.Fatal("expected method pointer skel names")
	}
	if serviceSkelName != serviceInfo.SkelName || methodSkelName != "ping" {
		t.Fatalf("unexpected method skel names: %s %s", serviceSkelName, methodSkelName)
	}
}

func TestRegisterReusesPubMethodInfoWhenProtocolTypesMatch(t *testing.T) {
	registry := NewRegistry()

	pubServiceInfo := newRegistryTestServiceInfo("test.registry.set.pubMethod")
	pubServiceInfo.Type = ServiceSpecTypeClient
	pubServiceInfo.ServerType = nil
	pubServiceInfo.DefaultServerType = nil
	pubServiceInfo.ERServerType = nil
	pubServiceInfo.DefaultERServerType = nil
	registry.Register(pubServiceInfo)
	pubMethodInfo := pubServiceInfo.Methods[0].Info()

	serverServiceInfo := newRegistryTestServiceInfo("test.registry.set.pubMethod")
	serverServiceInfo.Type = ServiceSpecTypeServer
	serverServiceInfo.ClientType = nil
	serverServiceInfo.ClientCtor = nil
	serverServiceInfo.ERClientType = nil
	serverServiceInfo.ERClientCtor = nil
	registry.Register(serverServiceInfo)

	if registry.serviceInfoBySkelName[serverServiceInfo.SkelName].Methods()[0] != pubMethodInfo {
		t.Fatal("expected register to reuse pub method info")
	}
}

func TestRegisterAllowsServerMethodInfoAndClientBlock(t *testing.T) {
	registry := NewRegistry()

	serverServiceInfo := newRegistryTestServiceInfo("test.registry.serverMethod")
	serverServiceInfo.Type = ServiceSpecTypeServer
	serverServiceInfo.ClientType = nil
	serverServiceInfo.ClientCtor = nil
	serverServiceInfo.ERClientType = nil
	serverServiceInfo.ERClientCtor = nil
	registry.Register(serverServiceInfo)
	serverMethodInfo := serverServiceInfo.Methods[0].Info()

	clientServiceInfo := newRegistryTestServiceInfo("test.registry.serverMethod")
	clientServiceInfo.Type = ServiceSpecTypeClient
	clientServiceInfo.ServerType = nil
	clientServiceInfo.DefaultServerType = nil
	clientServiceInfo.ERServerType = nil
	clientServiceInfo.DefaultERServerType = nil
	registry.Register(clientServiceInfo)

	if registry.serviceInfoBySkelName[clientServiceInfo.SkelName].Methods()[0] != serverMethodInfo {
		t.Fatal("expected service methods to keep server method info")
	}
	if registry.serviceInfoBySkelName[clientServiceInfo.SkelName].ClientType() != clientServiceInfo.ClientType {
		t.Fatal("expected client spec type to register client info")
	}
}

func TestRegisterAllowsDuplicateMethodInfoBySkelName(t *testing.T) {
	registry := NewRegistry()

	clientInfo := newRegistryTestServiceInfo("test.registry.equivalentMethod")
	clientInfo.Type = ServiceSpecTypeClient
	clientInfo.ServerType = nil
	clientInfo.DefaultServerType = nil
	clientInfo.ERServerType = nil
	clientInfo.DefaultERServerType = nil
	clientInfo.Methods = append(clientInfo.Methods, &MethodSpec{
		Name:     "Pong",
		SkelName: "pong",
	})
	registry.Register(clientInfo)
	clientPingMethodInfo := clientInfo.Methods[0].Info()
	clientPongMethodInfo := clientInfo.Methods[1].Info()

	serverInfo := newRegistryTestServiceInfo("test.registry.equivalentMethod")
	serverInfo.Type = ServiceSpecTypeServer
	serverInfo.ClientType = nil
	serverInfo.ClientCtor = nil
	serverInfo.ERClientType = nil
	serverInfo.ERClientCtor = nil
	serverInfo.Methods = []*MethodSpec{{
		Name:     "Pong",
		SkelName: "pong",
	}, {
		Name:        "Ping",
		SkelName:    "ping",
		MethodFuncs: []any{testRegistryServer.Ping},
	}}
	registry.Register(serverInfo)

	if serverInfo.Methods[0].Info() != clientPongMethodInfo {
		t.Fatal("expected server pong method spec to reuse client method info")
	}
	if serverInfo.Methods[1].Info() != clientPingMethodInfo {
		t.Fatal("expected server ping method spec to reuse client method info")
	}
	methodPointer := reflect.ValueOf(testRegistryServer.Ping).Pointer()
	serviceSkelName, methodSkelName, ok := registry.GetMethodSkelNamesByPointer(methodPointer)
	if !ok || serviceSkelName != serverInfo.SkelName || methodSkelName != "ping" {
		t.Fatalf("unexpected server method pointer mapping: %s %s %v", serviceSkelName, methodSkelName, ok)
	}
}

func TestRegisterRejectsDuplicateMethodInfo(t *testing.T) {
	registry := NewRegistry()

	pubServiceInfo := newRegistryTestServiceInfo("test.registry.set.differentMethod")
	pubServiceInfo.Type = ServiceSpecTypeClient
	pubServiceInfo.ServerType = nil
	pubServiceInfo.DefaultServerType = nil
	pubServiceInfo.ERServerType = nil
	pubServiceInfo.DefaultERServerType = nil
	registry.Register(pubServiceInfo)

	serverServiceInfo := newRegistryTestServiceInfo("test.registry.set.differentMethod")
	serverServiceInfo.Type = ServiceSpecTypeServer
	serverServiceInfo.ClientType = nil
	serverServiceInfo.ClientCtor = nil
	serverServiceInfo.ERClientType = nil
	serverServiceInfo.ERClientCtor = nil
	serverServiceInfo.Methods[0].SkelName = "pong"

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate method registration panic")
		}
		if !strings.Contains(recovered.(error).Error(), "service test.registry.set.differentMethod method already registered") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	registry.Register(serverServiceInfo)
}

func TestRegisterInitializesMethodBinaryFlags(t *testing.T) {
	registry := NewRegistry()

	serviceInfo := newRegistryTestServiceInfo("test.registry.binary")
	serviceInfo.Methods[0].ArgumentsContainsBinaryType = true
	serviceInfo.Methods[0].ResultContainsBinaryType = true
	registry.Register(serviceInfo)

	if !registry.serviceInfoBySkelName[serviceInfo.SkelName].Methods()[0].ArgumentsContainsBinaryType() {
		t.Fatal("expected method argumentsContainsBinaryType to be initialized")
	}
	if !registry.serviceInfoBySkelName[serviceInfo.SkelName].Methods()[0].ResultContainsBinaryType() {
		t.Fatal("expected method resultContainsBinaryType to be initialized")
	}
}

func TestRegisterInitializesMethodSensitiveFlags(t *testing.T) {
	registry := NewRegistry()

	serviceInfo := newRegistryTestServiceInfo("test.registry.sensitive")
	serviceInfo.Methods[0].ArgumentsSensitive = true
	serviceInfo.Methods[0].ResultSensitive = true
	registry.Register(serviceInfo)

	method := registry.serviceInfoBySkelName[serviceInfo.SkelName].Methods()[0]
	if !method.ArgumentsSensitive() || !method.ResultSensitive() {
		t.Fatal("expected method sensitive flags to be initialized")
	}
}

func TestRegisterKeepsFirstMethodCloneFuncs(t *testing.T) {
	registry := NewRegistry()

	clientInfo := newRegistryTestServiceInfo("test.registry.clone.first")
	clientInfo.Type = ServiceSpecTypeClient
	clientInfo.ServerType = nil
	clientInfo.DefaultServerType = nil
	clientInfo.ERServerType = nil
	clientInfo.DefaultERServerType = nil
	clientInfo.Methods[0].ArgumentsType = reflect.TypeFor[testRegistryCloneArguments]()
	clientInfo.Methods[0].ResultType = reflect.TypeFor[string]()
	clientInfo.Methods[0].CloneArguments = func(any) any { return "first arguments" }
	clientInfo.Methods[0].CloneResult = func(any) any { return "first result" }
	registry.Register(clientInfo)
	methodInfo := clientInfo.Methods[0].Info().(*_MethodInfo)

	serverInfo := newRegistryTestServiceInfo("test.registry.clone.first")
	serverInfo.Type = ServiceSpecTypeServer
	serverInfo.ClientType = nil
	serverInfo.ClientCtor = nil
	serverInfo.ERClientType = nil
	serverInfo.ERClientCtor = nil
	serverInfo.Methods[0].CloneArguments = func(any) any { return "second arguments" }
	serverInfo.Methods[0].CloneResult = func(any) any { return "second result" }
	registry.Register(serverInfo)

	if cloned := methodInfo.CloneArguments(&testRegistryCloneArguments{}); cloned != "first arguments" {
		t.Fatalf("CloneArguments() = %#v; want first registration", cloned)
	}
	if cloned := methodInfo.CloneResult(""); cloned != "first result" {
		t.Fatalf("CloneResult() = %#v; want first registration", cloned)
	}
}

func TestRegisterMergesPartialClientAndServerInfo(t *testing.T) {
	registry := NewRegistry()

	clientCtor := func() {}
	erClientCtor := func() {}

	clientInfo := newRegistryTestServiceInfo("test.registry.factories")
	clientInfo.Type = ServiceSpecTypeClient
	clientInfo.ServerType = nil
	clientInfo.DefaultServerType = nil
	clientInfo.ERServerType = nil
	clientInfo.DefaultERServerType = nil
	clientInfo.ClientCtor = clientCtor
	clientInfo.ERClientCtor = erClientCtor
	registry.Register(clientInfo)

	serverInfo := newRegistryTestServiceInfo("test.registry.factories")
	serverInfo.Type = ServiceSpecTypeServer
	serverInfo.ClientType = nil
	serverInfo.ClientCtor = nil
	serverInfo.ERClientType = nil
	serverInfo.ERClientCtor = nil
	registry.Register(serverInfo)

	gotFactories := registry.RegisteredClientFactories()
	wantFactories := []any{
		clientCtor,
		erClientCtor,
	}
	for _, wantFactory := range wantFactories {
		if !containsFactory(gotFactories, wantFactory) {
			t.Fatalf("expected registered client factory %p, got %#v", wantFactory, gotFactories)
		}
	}

	got, isERType := registry.getServiceInfoByImplType(reflect.TypeOf(&testRegistryServerImpl{}))
	if got.ServerType() != serverInfo.ServerType {
		t.Fatalf("unexpected runtime service info: %#v", got)
	}
	if isERType {
		t.Fatal("expected non-er pub server impl")
	}
}

func TestRegisterUsesServiceSpecTypeToSelectRegisteredSide(t *testing.T) {
	registry := NewRegistry()

	clientInfo := newRegistryTestServiceInfo("test.registry.type")
	clientInfo.Type = ServiceSpecTypeClient
	registry.Register(clientInfo)

	if registry.serviceInfoBySkelName[clientInfo.SkelName].ServerType() != nil {
		t.Fatal("expected client spec type to ignore server info")
	}

	serverInfo := newRegistryTestServiceInfo("test.registry.type")
	serverInfo.Type = ServiceSpecTypeServer
	registry.Register(serverInfo)

	if reflect.ValueOf(registry.serviceInfoBySkelName[serverInfo.SkelName].ClientCtor()).Pointer() != reflect.ValueOf(clientInfo.ClientCtor).Pointer() {
		t.Fatal("expected server spec type to keep existing client info")
	}
	if registry.serviceInfoBySkelName[serverInfo.SkelName].ServerType() != serverInfo.ServerType {
		t.Fatal("expected server spec type to register server info")
	}
}

func TestRegisterRejectsDuplicateClientInfo(t *testing.T) {
	registry := NewRegistry()

	clientInfo := newRegistryTestServiceInfo("test.registry.duplicateClient")
	clientInfo.Type = ServiceSpecTypeClient
	registry.Register(clientInfo)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected duplicate client registration panic")
		}
		if !strings.Contains(recovered.(error).Error(), "service test.registry.duplicateClient client already registered") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	duplicateClientInfo := newRegistryTestServiceInfo("test.registry.duplicateClient")
	duplicateClientInfo.Type = ServiceSpecTypeClient
	registry.Register(duplicateClientInfo)
}

func TestRegisterRejectsDuplicateServerInfo(t *testing.T) {
	registry := NewRegistry()

	serverServiceInfo := newRegistryTestServiceInfo("test.registry.duplicateServer")
	registry.Register(serverServiceInfo)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected duplicate server registration panic")
		}
		if !strings.Contains(recovered.(error).Error(), "service test.registry.duplicateServer server already registered") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	duplicateServerInfo := newRegistryTestServiceInfo("test.registry.duplicateServer")
	duplicateServerInfo.Type = ServiceSpecTypeServer
	duplicateServerInfo.ClientType = nil
	duplicateServerInfo.ClientCtor = nil
	duplicateServerInfo.ERClientType = nil
	duplicateServerInfo.ERClientCtor = nil
	registry.Register(duplicateServerInfo)
}

func TestRegisterRejectsMismatchedDefaultServerTypes(t *testing.T) {
	registry := NewRegistry()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected mismatched default server types panic")
		}
		if !strings.Contains(recovered.(error).Error(), "default server type and default er server type must both exist or both be nil") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	registry.Register(&ServiceSpec{
		Type:              ServiceSpecTypeServer,
		Name:              "BrokenService",
		SkelName:          "test.registry.broken",
		DefaultServerType: reflect.TypeOf(&defaultTestRegistryServer{}),
		Methods: []*MethodSpec{{
			Name:     "Ping",
			SkelName: "ping",
		}},
	})
}

func TestGetServiceInfoReturnsServiceInfoForDefaultServerAndERServer(t *testing.T) {
	registry := NewRegistry()

	serviceInfo := newRegistryTestServiceInfo("test.registry")
	registry.Register(serviceInfo)

	got, isERType := registry.getServiceInfoByImplType(reflect.TypeOf(&testRegistryServerImpl{}))
	if got.SkelName() != serviceInfo.SkelName {
		t.Fatalf("unexpected service info for server impl: got %s want %s", got.SkelName(), serviceInfo.SkelName)
	}
	if isERType {
		t.Fatalf("expected non-er server impl")
	}

	got, isERType = registry.getServiceInfoByImplType(reflect.TypeOf(&testRegistryServerERImpl{}))
	if got.SkelName() != serviceInfo.SkelName {
		t.Fatalf("unexpected service info for er server impl: got %s want %s", got.SkelName(), serviceInfo.SkelName)
	}
	if !isERType {
		t.Fatalf("expected er server impl")
	}
}

func TestGetEmbeddedTypesReturnsOnlyAnonymousStructFields(t *testing.T) {
	embeddedTypes := reflectutil.EmbeddedStructTypes(reflect.TypeOf(embeddedTypesHolder{}))

	if len(embeddedTypes) != 2 {
		t.Fatalf("unexpected embedded type count: got %d", len(embeddedTypes))
	}
	if embeddedTypes[0] != reflect.TypeOf(embeddedTypeA{}) {
		t.Fatalf("unexpected first embedded type: got %v", embeddedTypes[0])
	}
	if embeddedTypes[1] != reflect.TypeOf(embeddedTypeB{}) {
		t.Fatalf("unexpected second embedded type: got %v", embeddedTypes[1])
	}
}

func TestRegisterRejectsDuplicateSkelName(t *testing.T) {
	registry := NewRegistry()

	registry.Register(newRegistryTestServiceInfo("test.registry"))

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected duplicate registration panic")
		}
		if !strings.Contains(recovered.(error).Error(), "service test.registry server already registered") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	serviceInfo := newRegistryTestServiceInfo("test.registry")
	serviceInfo.DefaultServerType = reflect.TypeOf(&struct{ defaultTestRegistryServer }{})
	serviceInfo.DefaultERServerType = reflect.TypeOf(&struct{ defaultTestRegistryServerER }{})
	registry.Register(serviceInfo)
}
