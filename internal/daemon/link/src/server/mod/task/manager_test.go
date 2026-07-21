package task

import (
	"context"
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

type _ManagerAppTaskClient struct {
	runTask func(run appskeled.TaskRun) error
}

func (c *_ManagerAppTaskClient) RunTask(run appskeled.TaskRun, _ ...rpcclient.InvokeOption) ex.Error {
	err := c.runTask(run)
	if err == nil {
		return nil
	}
	return ex.New(ex.Internal, err.Error())
}

type _ManagerDispatchHooks struct {
	mutex       sync.Mutex
	runs        []appskeled.TaskRun
	timeout     time.Duration
	callCount   int
	startedChan chan struct{}
	releaseChan chan struct{}
}

func newTestManager(t *testing.T) (*Manager, func()) {
	t.Helper()

	natsModule := &hubnatsserver.NATSServer{
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
		Flag:       &hubflag.Flag{MQEmbeddedNats: true},
	}
	natsModule.DIInit()

	natsClient, conn := newTestNATSClient(t)

	appInfo, err := meta.NewApp("vine.link", "1.0.0", "22222222-2222-2222-2222-222222222222")
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

	cleanup := func() {
		manager.AfterAppStop()
		conn.Close()
		natsModule.AfterAppStop()
	}
	return manager, cleanup
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
