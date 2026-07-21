package minder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func TestNewAppInstanceClonesRegistrationSlices(t *testing.T) {
	registration := AppRegistration{
		AppInfo:         mustTestMetaApp(),
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
		IngressEndpoint: "http://127.0.0.1:8081",
		ServiceHandlers: []skeled.ServiceHandlerRegistration{{ServiceSkelName: "demo.service.UserService"}},
		WebHandlers:     []skeled.WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
		EventListeners:  []skeled.EventListenerRegistration{{EventSkelName: "demo.event.Created"}},
		TaskRunners:     []skeled.TaskRunnerRegistration{{TaskSkelName: "demo.task.Sync"}},
	}
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{},
		App:                   mustTestMetaApp(),
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: &_RegistryServiceClient{},
	}
	minder.DIInit()

	instance := minder.newAppInstance(registration)
	registration.ServiceHandlers[0].ServiceSkelName = "changed"
	registration.WebHandlers[0].WebSkelName = "changed"
	registration.EventListeners[0].EventSkelName = "changed"
	registration.TaskRunners[0].TaskSkelName = "changed"

	assert.Equal(t, []skeled.ServiceHandlerRegistration{{ServiceSkelName: "demo.service.UserService"}}, instance.ServiceHandlers)
	assert.Equal(t, []skeled.WebHandlerRegistration{{WebSkelName: "default@demo.app"}}, instance.WebHandlers)
	assert.Equal(t, []skeled.EventListenerRegistration{{EventSkelName: "demo.event.Created"}}, instance.EventListeners)
	assert.Equal(t, []skeled.TaskRunnerRegistration{{TaskSkelName: "demo.task.Sync"}}, instance.TaskRunners)
}
