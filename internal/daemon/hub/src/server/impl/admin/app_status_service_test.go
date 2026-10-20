package admin

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

func TestAppStatusServiceListSortsStatusesAndRegistrations(t *testing.T) {
	service := &AppStatusApiServiceServerImpl{
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
