package event

import (
	"context"
	"fmt"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	hubnatsserver "go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	linknats "go.yorun.ai/vine/internal/daemon/link/src/server/comp/nats"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

type _ManagerRegistryServiceClient struct{}

func (*_ManagerRegistryServiceClient) Register(hubskeled.AppRegistration, ...rpcclient.InvokeOption) {
}
func (*_ManagerRegistryServiceClient) Unregister(string, skel.UUID, ...rpcclient.InvokeOption) {
}
func (*_ManagerRegistryServiceClient) Heartbeat(hubskeled.AppStatus, ...rpcclient.InvokeOption) bool {
	return true
}

type _ManagerAppEventClient struct {
	onEvent func(on appskeled.EventOn) error
}

func (c *_ManagerAppEventClient) OnEvent(on appskeled.EventOn, _ ...rpcclient.InvokeOption) ex.Error {
	err := c.onEvent(on)
	if err == nil {
		return nil
	}
	return ex.New(ex.Internal, err.Error())
}

type _ManagerDispatchHooks struct {
	mutex       sync.Mutex
	events      []appskeled.EventOn
	timeout     time.Duration
	callCount   int
	startedChan chan struct{}
	releaseChan chan struct{}
}

func newTestManagers(t *testing.T, count int) ([]*Manager, func()) {
	t.Helper()

	natsModule := &hubnatsserver.NATSServer{
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag:       &hubflag.Flag{MQEmbeddedNats: true},
	}
	natsModule.DIInit()

	managers := make([]*Manager, 0, count)
	conns := make([]*gonats.Conn, 0, count)
	for index := 0; index < count; index++ {
		natsClient, conn := newTestNATSClient(t)
		conns = append(conns, conn)

		appInfo, err := meta.NewApp("vine.link", "1.0.0", fmt.Sprintf("22222222-2222-2222-2222-%012d", index+1))
		require.NoError(t, err)
		appMinder := &minder.AppMinder{
			Context:               context.Background(),
			Flag:                  &flag.Flag{HubInprocMode: true},
			App:                   appInfo,
			InprocFlag:            &internalapp.InternalInprocFlag{Enabled: true},
			RegistryServiceClient: &_ManagerRegistryServiceClient{},
		}
		appMinder.DIInit()

		manager := &Manager{
			Context:    context.Background(),
			App:        appInfo,
			NATSClient: natsClient,
			AppMinder:  appMinder,
		}
		manager.DIInit()
		managers = append(managers, manager)
	}

	cleanup := func() {
		for _, manager := range managers {
			manager.AfterAppStop()
		}
		for _, conn := range conns {
			conn.Close()
		}
		natsModule.AfterAppStop()
	}
	return managers, cleanup
}

func newTestManager(t *testing.T) (*Manager, func()) {
	t.Helper()
	managers, cleanup := newTestManagers(t, 1)
	return managers[0], cleanup
}

func newTestNATSClient(t *testing.T) (*linknats.Client, *gonats.Conn) {
	t.Helper()

	conn := hubnats.ConnectInproc()
	var err error
	js, err := jetstream.New(conn)
	require.NoError(t, err)

	client := new(linknats.Client)
	setUnexportedField(t, reflect.ValueOf(client).Elem().FieldByName("_Client").FieldByName("conn"), conn)
	setUnexportedField(t, reflect.ValueOf(client).Elem().FieldByName("_Client").FieldByName("jetStream"), js)
	setUnexportedField(t, reflect.ValueOf(client).Elem().FieldByName("_Client").FieldByName("ensuredStream"), map[string]struct{}{})
	consumersField := reflect.ValueOf(client).Elem().FieldByName("_Client").FieldByName("consumers")
	setUnexportedFieldValue(t, consumersField, reflect.MakeMap(consumersField.Type()))
	return client, conn
}

func setUnexportedField(t *testing.T, field reflect.Value, value any) {
	t.Helper()
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func setUnexportedFieldValue(t *testing.T, field reflect.Value, value reflect.Value) {
	t.Helper()
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(value)
}
