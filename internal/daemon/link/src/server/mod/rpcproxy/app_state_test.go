package rpcproxy

import (
	"sync"
	"testing"
)

func TestOnDrainPublishesImmutableAppState(t *testing.T) {
	proxy := newTestRpcProxy(t, nil)
	localApp := mustMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111111")
	registerLocalApp(proxy, localApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.service.UserService"})

	beforeDrain, ok := proxy.getAppStateByInstanceID(localApp.InstanceId())
	if !ok {
		t.Fatal("expected local app state before drain")
	}
	proxy.OnDrain(beforeDrain.instance)
	afterDrain, ok := proxy.getAppStateByInstanceID(localApp.InstanceId())
	if !ok {
		t.Fatal("expected local app state after drain")
	}

	if beforeDrain == afterDrain {
		t.Fatal("expected drain to publish a new app state")
	}
	if beforeDrain.draining {
		t.Fatal("previously published app state was mutated")
	}
	if !afterDrain.draining {
		t.Fatal("new app state should be draining")
	}
}

func TestAppStateConcurrentReadAndLifecycleUpdates(t *testing.T) {
	proxy := newTestRpcProxy(t, nil)
	localApp := mustMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111111")
	registerLocalApp(proxy, localApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.service.UserService"})
	state, ok := proxy.getAppStateByInstanceID(localApp.InstanceId())
	if !ok {
		t.Fatal("expected local app state")
	}
	instance := state.instance

	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		<-start
		for idx := 0; idx < 1000; idx++ {
			proxy.OnSetup(instance)
			proxy.OnDrain(instance)
		}
	}()
	go func() {
		defer waitGroup.Done()
		<-start
		for idx := 0; idx < 1000; idx++ {
			current, exists := proxy.getAppStateByInstanceID(localApp.InstanceId())
			if !exists {
				continue
			}
			_ = current.instance
			_ = current.appInfo.InstanceId()
			_ = current.serviceEndpoint
			_ = current.draining
			_ = current.hasService("demo.service.UserService")
		}
	}()
	close(start)
	waitGroup.Wait()
}
