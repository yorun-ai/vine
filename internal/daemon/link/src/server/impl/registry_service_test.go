package impl

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/ingress"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

type _LinkRegistryRegistryServiceClient struct {
	registrations []hubskeled.AppRegistration
	unregistered  []skel.UUID
}

func (c *_LinkRegistryRegistryServiceClient) Register(registration hubskeled.AppRegistration, _ivOpts ...client.InvokeOption) {
	c.registrations = append(c.registrations, registration)
}

func (c *_LinkRegistryRegistryServiceClient) Unregister(_ string, instanceId skel.UUID, _ivOpts ...client.InvokeOption) {
	c.unregistered = append(c.unregistered, instanceId)
}

func (*_LinkRegistryRegistryServiceClient) Heartbeat(hubskeled.AppStatus, ...client.InvokeOption) bool {
	return true
}

func mustLinkRegistryTestApp() meta.App {
	app, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	if err != nil {
		panic(err)
	}
	return app
}

func newLinkRegistryTestIngress() *ingress.Ingress {
	return &ingress.Ingress{}
}

func newTestAppMinder(t *testing.T, app meta.App, client *_LinkRegistryRegistryServiceClient) *minder.AppMinder {
	t.Helper()
	appMinder := &minder.AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{HubInprocMode: true},
		App:                   app,
		InprocFlag:            &internalapp.InternalInprocFlag{},
		RegistryServiceClient: client,
	}
	appMinder.DIInit()
	return appMinder
}

func TestRegistryServiceRegisterStartsHeartbeat(t *testing.T) {
	app, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	assert.NoError(t, err)

	client := &_LinkRegistryRegistryServiceClient{}
	appMinder := newTestAppMinder(t, app, client)
	ing := newLinkRegistryTestIngress()
	service := &RegistryServiceServerImpl{
		AppMinder: appMinder,
		Ingress:   ing,
		Context:   newLinkServiceSpecContext(t, app),
	}

	service.Register(skeled.AppRegistration{
		ConsoleEndpoint:   "http://127.0.0.1:8080/console",
		ServiceEndpoint:   "http://127.0.0.1:8080/rpc/invoke",
		WebEndpointPrefix: "http://127.0.0.1:8080/web/access",
		EventEndpoint:     "http://127.0.0.1:8080/event",
		TaskEndpoint:      "http://127.0.0.1:8080/task",
		ServiceHandlers: []skeled.ServiceHandlerRegistration{{
			ServiceSkelName: "demo.service.UserService",
		}},
		WebHandlers: []skeled.WebHandlerRegistration{{
			WebSkelName: "default@demo.app",
		}},
	})

	assert.Len(t, client.registrations, 1)
	assert.Equal(t, skel.NewUUID(uuid.MustParse(app.InstanceId())), client.registrations[0].InstanceId)
	assert.Equal(t, ing.Endpoint(), client.registrations[0].Endpoint)
	assert.Equal(t, []hubskeled.ServiceHandlerRegistration{{
		ServiceSkelName: "demo.service.UserService",
		Endpoint:        "",
	}}, client.registrations[0].ServiceHandlers)
	assert.Equal(t, []hubskeled.WebHandlerRegistration{{
		WebSkelName: "default@demo.app",
		Endpoint:    "",
	}}, client.registrations[0].WebHandlers)
	assert.Empty(t, client.registrations[0].EventListeners)
	assert.Empty(t, client.registrations[0].TaskRunners)
	appMinder.AfterAppStop()
}

func TestRegistryServiceUnregisterStopsHeartbeat(t *testing.T) {
	app, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	assert.NoError(t, err)

	client := &_LinkRegistryRegistryServiceClient{}
	appMinder := newTestAppMinder(t, app, client)
	ing := newLinkRegistryTestIngress()
	appMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:         app,
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
		IngressEndpoint: ing.Endpoint(),
		ServiceHandlers: []skeled.ServiceHandlerRegistration{{
			ServiceSkelName: "demo.service.UserService",
		}},
		WebHandlers: []skeled.WebHandlerRegistration{{
			WebSkelName: "default@demo.app",
		}},
	})
	service := &RegistryServiceServerImpl{
		AppMinder: appMinder,
		Ingress:   ing,
		Context:   newLinkServiceSpecContext(t, app),
	}

	service.Unregister()

	assert.Equal(t, []skel.UUID{skel.NewUUID(uuid.MustParse(app.InstanceId()))}, client.unregistered)
}

func newLinkServiceSpecContext(t *testing.T, app meta.App) spec.Context {
	t.Helper()

	trace := meta.InitialTrace()
	actor := meta.NewAbsentActor()
	return spec.NewContext(context.Background(), trace, app, nil, actor)
}
