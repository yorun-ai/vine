package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type _AppStatusServiceRegistryRepo struct {
	statuses []*core.AppStatus
}

func (*_AppStatusServiceRegistryRepo) SaveAppStatus(*core.AppStatus) {}

func (r *_AppStatusServiceRegistryRepo) ListAppStatuses() []*core.AppStatus {
	return r.statuses
}

func (*_AppStatusServiceRegistryRepo) GetAppStatus(string, string) (*core.AppStatus, bool) {
	return nil, false
}

func (*_AppStatusServiceRegistryRepo) KeepAppStatus(string, string) bool {
	return true
}

func (*_AppStatusServiceRegistryRepo) RemoveAppStatus(string, string) {}

func (*_AppStatusServiceRegistryRepo) PopExpiredAppLeases() []core.AppHeartbeat {
	return nil
}

func (*_AppStatusServiceRegistryRepo) SaveRpcServiceRegistration(*core.RpcServiceRegistration) {
}

func (*_AppStatusServiceRegistryRepo) GetRpcServiceRegistration(string, string, string) (*core.RpcServiceRegistration, bool) {
	return nil, false
}

func (*_AppStatusServiceRegistryRepo) KeepRpcServiceRegistration(string, string, string) bool {
	return true
}

func (*_AppStatusServiceRegistryRepo) RemoveRpcServiceRegistration(string, string, string) {
}

func (*_AppStatusServiceRegistryRepo) SaveWebRegistration(*core.WebRegistration) {}

func (*_AppStatusServiceRegistryRepo) GetWebRegistration(string, string, string) (*core.WebRegistration, bool) {
	return nil, false
}

func (*_AppStatusServiceRegistryRepo) KeepWebRegistration(string, string, string) bool {
	return true
}

func (*_AppStatusServiceRegistryRepo) RemoveWebRegistration(string, string, string) {}

func TestAppStatusServiceListReturnsEmptyList(t *testing.T) {
	service := &AppStatusServiceServerImpl{
		RegistryRepo: &_AppStatusServiceRegistryRepo{},
	}

	items := service.List()

	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestAppStatusServiceList(t *testing.T) {
	service := &AppStatusServiceServerImpl{
		RegistryRepo: &_AppStatusServiceRegistryRepo{
			statuses: []*core.AppStatus{{
				Name:       "demo.booker",
				InstanceId: "instance-1",
				Version:    "v1.0.0",
				Endpoint:   "http://127.0.0.1:7001",
				ServiceHandlers: []core.ServiceHandlerRegistration{{
					ServiceSkelName: "demo.booker.CatalogService",
					SchemaHash:      "12345678",
					Endpoint:        "http://127.0.0.1:7001/rpc",
				}},
				WebHandlers: []core.WebHandlerRegistration{{
					WebSkelName: "demo.booker.BookerPortalWeb",
					SchemaHash:  "23456789",
					Endpoint:    "http://127.0.0.1:7001/web",
				}},
				EventListeners: []core.EventListenerRegistration{{
					EventSkelName: "demo.booker.BookCreatedEvent",
					SchemaHash:    "34567890",
					TimeoutMs:     3000,
					Concurrency:   2,
				}},
				TaskRunners: []core.TaskRunnerRegistration{{
					TaskSkelName: "demo.booker.SyncInventoryTask",
					SchemaHash:   "45678901",
					TimeoutMs:    5000,
					Concurrency:  1,
					NoRetry:      true,
				}},
			}},
		},
	}

	items := service.List()

	require.Len(t, items, 1)
	assert.Equal(t, "demo.booker", items[0].Name)
	assert.Equal(t, "instance-1", items[0].InstanceId)
	assert.Equal(t, "v1.0.0", items[0].Version)
	require.Len(t, items[0].ServiceHandlers, 1)
	assert.Equal(t, "demo.booker.CatalogService", items[0].ServiceHandlers[0].ServiceSkelName)
	require.Len(t, items[0].WebHandlers, 1)
	assert.Equal(t, "demo.booker.BookerPortalWeb", items[0].WebHandlers[0].WebSkelName)
	require.Len(t, items[0].EventListeners, 1)
	assert.Equal(t, "demo.booker.BookCreatedEvent", items[0].EventListeners[0].EventSkelName)
	require.Len(t, items[0].TaskRunners, 1)
	assert.Equal(t, "demo.booker.SyncInventoryTask", items[0].TaskRunners[0].TaskSkelName)
}

func TestAppStatusServiceListReturnsEmptyRegistrationLists(t *testing.T) {
	service := &AppStatusServiceServerImpl{
		RegistryRepo: &_AppStatusServiceRegistryRepo{
			statuses: []*core.AppStatus{{
				Name:       "demo.user",
				InstanceId: "instance-1",
			}},
		},
	}

	items := service.List()

	require.Len(t, items, 1)
	assert.NotNil(t, items[0].ServiceHandlers)
	assert.Empty(t, items[0].ServiceHandlers)
	assert.NotNil(t, items[0].WebHandlers)
	assert.Empty(t, items[0].WebHandlers)
	assert.NotNil(t, items[0].EventListeners)
	assert.Empty(t, items[0].EventListeners)
	assert.NotNil(t, items[0].TaskRunners)
	assert.Empty(t, items[0].TaskRunners)
}

func TestAppStatusServiceListSortsStatusesAndRegistrations(t *testing.T) {
	service := &AppStatusServiceServerImpl{
		RegistryRepo: &_AppStatusServiceRegistryRepo{
			statuses: []*core.AppStatus{{
				Name:       "demo.user",
				InstanceId: "instance-2",
			}, {
				Name:       "demo.booker",
				InstanceId: "instance-2",
				ServiceHandlers: []core.ServiceHandlerRegistration{{
					ServiceSkelName: "demo.booker.ZService",
				}, {
					ServiceSkelName: "demo.booker.AService",
				}},
				WebHandlers: []core.WebHandlerRegistration{{
					WebSkelName: "demo.booker.ZWeb",
				}, {
					WebSkelName: "demo.booker.AWeb",
				}},
				EventListeners: []core.EventListenerRegistration{{
					EventSkelName: "demo.booker.ZEvent",
				}, {
					EventSkelName: "demo.booker.AEvent",
				}},
				TaskRunners: []core.TaskRunnerRegistration{{
					TaskSkelName: "demo.booker.ZTask",
				}, {
					TaskSkelName: "demo.booker.ATask",
				}},
			}, {
				Name:       "demo.booker",
				InstanceId: "instance-1",
			}},
		},
	}

	items := service.List()

	require.Len(t, items, 3)
	assert.Equal(t, "demo.booker", items[0].Name)
	assert.Equal(t, "instance-1", items[0].InstanceId)
	assert.Equal(t, "demo.booker", items[1].Name)
	assert.Equal(t, "instance-2", items[1].InstanceId)
	assert.Equal(t, "demo.user", items[2].Name)
	assert.Equal(t, "instance-2", items[2].InstanceId)
	assert.Equal(t, "demo.booker.AService", items[1].ServiceHandlers[0].ServiceSkelName)
	assert.Equal(t, "demo.booker.ZService", items[1].ServiceHandlers[1].ServiceSkelName)
	assert.Equal(t, "demo.booker.AWeb", items[1].WebHandlers[0].WebSkelName)
	assert.Equal(t, "demo.booker.ZWeb", items[1].WebHandlers[1].WebSkelName)
	assert.Equal(t, "demo.booker.AEvent", items[1].EventListeners[0].EventSkelName)
	assert.Equal(t, "demo.booker.ZEvent", items[1].EventListeners[1].EventSkelName)
	assert.Equal(t, "demo.booker.ATask", items[1].TaskRunners[0].TaskSkelName)
	assert.Equal(t, "demo.booker.ZTask", items[1].TaskRunners[1].TaskSkelName)
}
