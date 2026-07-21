package minder

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type _EndpointMutator struct{}

func (*_EndpointMutator) OnSetup(instance *AppInstance) {
	for idx := range instance.HubServiceHandlers {
		instance.HubServiceHandlers[idx].Endpoint = testRpcProxyInEndpointOf(instance.IngressEndpoint, instance.AppInfo.InstanceId())
	}
	for idx := range instance.HubWebHandlers {
		webSkelName := instance.HubWebHandlers[idx].WebSkelName
		instance.HubWebHandlers[idx].Endpoint = testWebProxyInEndpointOf(instance.IngressEndpoint, instance.AppInfo.InstanceId(), webSkelName)
	}
}

func (*_EndpointMutator) OnDrain(*AppInstance)   {}
func (*_EndpointMutator) OnDestroy(*AppInstance) {}

func testRpcProxyInEndpointOf(endpoint string, instanceID string) string {
	return endpoint + "/rpc/proxy/in/" + instanceID
}

func testWebProxyInEndpointOf(endpoint string, instanceID string, webSkelName string) string {
	return endpoint + "/web/proxy/in/" + instanceID + "/" + webSkelName
}

func TestRegisterHubInstancePublishesRegistration(t *testing.T) {
	appInfo := mustTestMetaApp()
	client := &_RegistryServiceClient{}
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{},
		App:                   appInfo,
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: client,
	}
	minder.DIInit()
	minder.AddMutator(&_EndpointMutator{})
	instance := minder.newAppInstance(AppRegistration{
		AppInfo:         appInfo,
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
		IngressEndpoint: "http://127.0.0.1:8081",
		ServiceHandlers: []skeled.ServiceHandlerRegistration{{ServiceSkelName: "demo.service.UserService"}},
		WebHandlers:     []skeled.WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
		EventListeners:  []skeled.EventListenerRegistration{{EventSkelName: "demo.event.Created"}},
		TaskRunners:     []skeled.TaskRunnerRegistration{{TaskSkelName: "demo.task.Sync"}},
	})
	minder.beforeRegistration(instance)

	instance.registerHubInstance()

	client.mutex.Lock()
	defer client.mutex.Unlock()
	assert.Len(t, client.registrations, 1)
	assert.Equal(t, hubskeled.AppRegistration{
		Name:       appInfo.Name(),
		InstanceId: skel.NewUUID(uuid.MustParse(appInfo.InstanceId())),
		Version:    appInfo.Version(),
		Endpoint:   "http://127.0.0.1:8081",
		ServiceHandlers: []hubskeled.ServiceHandlerRegistration{{
			ServiceSkelName: "demo.service.UserService",
			Endpoint:        testRpcProxyInEndpointOf("http://127.0.0.1:8081", appInfo.InstanceId()),
		}},
		WebHandlers: []hubskeled.WebHandlerRegistration{{
			WebSkelName: "default@demo.app",
			Endpoint:    testWebProxyInEndpointOf("http://127.0.0.1:8081", appInfo.InstanceId(), "default@demo.app"),
		}},
		EventListeners: []hubskeled.EventListenerRegistration{{EventSkelName: "demo.event.Created"}},
		TaskRunners:    []hubskeled.TaskRunnerRegistration{{TaskSkelName: "demo.task.Sync", CronSchedulers: []hubskeled.TaskRunnerCronScheduler{}}},
	}, client.registrations[0])
}

func TestUnregisterHubInstancePublishesUnregister(t *testing.T) {
	appInfo := mustTestMetaApp()
	client := &_RegistryServiceClient{}
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{},
		App:                   appInfo,
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: client,
	}
	minder.DIInit()
	instance := minder.newAppInstance(AppRegistration{AppInfo: appInfo})

	instance.unregisterHubInstance()

	client.mutex.Lock()
	defer client.mutex.Unlock()
	assert.Equal(t, []skel.UUID{skel.NewUUID(uuid.MustParse(appInfo.InstanceId()))}, client.unregistered)
}
