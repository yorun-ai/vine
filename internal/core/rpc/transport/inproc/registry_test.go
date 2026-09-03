package inproc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type registryTestHandler struct{}

func (registryTestHandler) ServeRpc(rpcRequest spec.Request) spec.Response {
	return &spec.ResponseImpl{
		ServerValue: rpcRequest.Client(),
		ErrorValue:  ex.NewOK(),
	}
}

func resetRegistryForTest(t *testing.T) {
	t.Helper()

	registryMutex.Lock()
	prev := handlerByEndpoint
	handlerByEndpoint = map[string]*_Registration{}
	registryMutex.Unlock()
	t.Cleanup(func() {
		registryMutex.Lock()
		defer registryMutex.Unlock()
		handlerByEndpoint = prev
	})
}

func registryTestRequest(t *testing.T) spec.Request {
	t.Helper()

	app, err := meta.NewApp("registry.client", "1.0.0", "123e4567-e89b-12d3-a456-426614174010")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	return &spec.RequestImpl{
		ContextValue: context.Background(),
		TraceValue:   meta.InitialTrace(),
		ClientValue:  app,
		MethodInfoValue: spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
			Name:     "RegistryTestService",
			SkelName: "registry.test.service",
			Methods: []*spec.MethodSpec{{
				Name:     "Ping",
				SkelName: "ping",
			}},
		}).Methods()[0],
	}
}

func TestRegisterAndGetHandler(t *testing.T) {
	resetRegistryForTest(t)

	handler := registryTestHandler{}
	Register("rpc+inproc://hub.api", handler)

	got, ok := getHandler("rpc+inproc://hub.api")
	if !ok {
		t.Fatal("expected registered handler")
	}
	if got != handler {
		t.Fatalf("unexpected handler: got %#v want %#v", got, handler)
	}
}

func TestGetHandlerReturnsFalseForMissingEndpoint(t *testing.T) {
	resetRegistryForTest(t)

	got, ok := getHandler("rpc+inproc://link.api")
	if ok {
		t.Fatalf("expected missing handler, got %#v", got)
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	resetRegistryForTest(t)

	assertPanicsWith := func(want string, fn func()) {
		t.Helper()
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatalf("expected panic containing %q", want)
			}
			if !strings.Contains(recovered.(error).Error(), want) {
				t.Fatalf("unexpected panic: %v", recovered)
			}
		}()
		fn()
	}

	assertPanicsWith("must start with rpc+inproc://", func() {
		Register("http://hub.api", registryTestHandler{})
	})
	assertPanicsWith("host is empty", func() {
		Register("rpc+inproc://", registryTestHandler{})
	})
	assertPanicsWith("cannot be nil", func() {
		Register("rpc+inproc://hub.api", nil)
	})
}

func TestRegisterRejectsDuplicateEndpoint(t *testing.T) {
	resetRegistryForTest(t)

	Register("rpc+inproc://hub.api", registryTestHandler{})

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate endpoint panic")
		}
		if !strings.Contains(recovered.(error).Error(), "already registered") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	Register("rpc+inproc://hub.api", registryTestHandler{})
}

func TestRegisteredHandlerWorksWithRoundTrip(t *testing.T) {
	resetRegistryForTest(t)

	Register("rpc+inproc://hub.api", registryTestHandler{})
	response, err := RoundTrip("rpc+inproc://hub.api", registryTestRequest(t))
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	if response == nil || response.Error() == nil || response.Error().Code() != ex.OK {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestUnregisterRemovesHandler(t *testing.T) {
	resetRegistryForTest(t)

	Register("rpc+inproc://hub.api", registryTestHandler{})
	Unregister("rpc+inproc://hub.api")

	got, ok := getHandler("rpc+inproc://hub.api")
	if ok {
		t.Fatalf("expected handler to be removed, got %#v", got)
	}
}

func TestRegistrationCleanupIsIdempotentAndDoesNotRemoveReplacement(t *testing.T) {
	resetRegistryForTest(t)

	endpoint := "rpc+inproc://hub.api"
	cleanupFirst := Register(endpoint, registryTestHandler{})
	Unregister(endpoint)
	cleanupSecond := Register(endpoint, registryTestHandler{})

	cleanupFirst()
	cleanupFirst()
	if _, ok := getHandler(endpoint); !ok {
		t.Fatal("stale cleanup removed replacement handler")
	}

	cleanupSecond()
	cleanupSecond()
	if handler, ok := getHandler(endpoint); ok {
		t.Fatalf("cleanup did not remove handler: %#v", handler)
	}
}

func TestRegistrySupportsConcurrentRegistrationsAndLookups(t *testing.T) {
	resetRegistryForTest(t)

	const count = 64
	results := make(chan bool, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		endpoint := fmt.Sprintf("rpc+inproc://concurrent-%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			cleanup := Register(endpoint, registryTestHandler{})
			_, registered := getHandler(endpoint)
			cleanup()
			_, remains := getHandler(endpoint)
			results <- registered && !remains
		}()
	}
	wg.Wait()
	close(results)
	for result := range results {
		if !result {
			t.Fatal("concurrent registration lifecycle failed")
		}
	}
}
