package repo

import (
	"cmp"
	"time"

	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vslice"
)

const hubRegistryLeaseKey = "app:leases"
const hubRegistryEphemeralTTL = 180 * time.Second
const hubRegistryLeaseTTL = 30 * time.Second
const hubRegistryLeaseSweepLimit = 100

// Structs

type _AppStatus struct {
	InstanceId      string                            `json:"instanceId"`
	Name            string                            `json:"name"`
	Version         string                            `json:"version"`
	Endpoint        string                            `json:"endpoint"`
	ExpiresAt       time.Time                         `json:"expiresAt"`
	ServiceHandlers []core.ServiceHandlerRegistration `json:"serviceHandlers"`
	WebHandlers     []core.WebHandlerRegistration     `json:"webHandlers"`
	EventListeners  []core.EventListenerRegistration  `json:"eventListeners"`
	TaskRunners     []core.TaskRunnerRegistration     `json:"taskRunners"`
}

type _AppLease struct {
	Name       string `json:"name"`
	InstanceId string `json:"instanceId"`
}

// Repo

type RegistryRepo struct {
	WatchServer *watchserver.Server             `inject:""`
	InprocFlag  *internalapp.InternalInprocFlag `inject:""`
}

func (r *RegistryRepo) SaveAppStatus(status *core.AppStatus) {
	r.saveStatus(status)
}

func (r *RegistryRepo) ListAppStatuses() []*core.AppStatus {
	keys := r.WatchServer.Scan(watched.FormatAppStatusPattern())
	items := make([]*core.AppStatus, 0, len(keys))
	for _, key := range keys {
		value, ok := r.WatchServer.Get(key)
		if !ok {
			continue
		}
		item := toCoreAppStatus(vcode.MustUnmarshalJsonS[*_AppStatus](value))
		items = append(items, item)
	}
	return vslice.SortBy(items, func(a *core.AppStatus, b *core.AppStatus) bool {
		if a.Name != b.Name {
			return cmp.Compare(a.Name, b.Name) < 0
		}
		return cmp.Compare(a.InstanceId, b.InstanceId) < 0
	})
}

func (r *RegistryRepo) GetAppStatus(appName string, instanceId string) (*core.AppStatus, bool) {
	status, ok := r.getAppStatus(appName, instanceId)
	if !ok {
		return nil, false
	}
	return toCoreAppStatus(status), true
}

func (r *RegistryRepo) KeepAppStatus(appName string, instanceId string) bool {
	if r.InprocFlag.Enabled {
		return true
	}
	status, ok := r.getAppStatus(appName, instanceId)
	if !ok {
		return false
	}
	status.ExpiresAt = timeNow().Add(hubRegistryLeaseTTL)
	key := watched.FormatAppStatusKey(appName, instanceId)
	r.WatchServer.SetEphemeral(key, vcode.MustMarshalJsonS(status), hubRegistryEphemeralTTL)
	r.saveAppLease(appName, instanceId)
	return true
}

func (r *RegistryRepo) RemoveAppStatus(appName string, instanceId string) {
	key := watched.FormatAppStatusKey(appName, instanceId)
	r.removeAppLease(appName, instanceId)
	r.WatchServer.DeleteAndNotify(key)
}

func (r *RegistryRepo) SaveRpcServiceRegistration(registration *core.RpcServiceRegistration) {
	watchRegistration := toRpcServiceRegistration(registration)
	key := watched.FormatRpcServiceRegistrationKey(watchRegistration.ServiceName, watchRegistration.AppName, watchRegistration.AppInstanceId)
	value := vcode.MustMarshalJsonS(watchRegistration)
	if r.InprocFlag.Enabled {
		r.WatchServer.SetAndNotify(key, value)
		return
	}
	r.WatchServer.SetEphemeralAndNotify(key, value, hubRegistryEphemeralTTL)
}

func (r *RegistryRepo) GetRpcServiceRegistration(serviceName string, appName string, instanceId string) (*core.RpcServiceRegistration, bool) {
	key := watched.FormatRpcServiceRegistrationKey(serviceName, appName, instanceId)
	value, ok := r.WatchServer.Get(key)
	if !ok {
		return nil, false
	}
	return toCoreRpcServiceRegistration(vcode.MustUnmarshalJsonS[*watched.RpcServiceRegistration](value)), true
}

func (r *RegistryRepo) KeepRpcServiceRegistration(serviceName string, appName string, appInstanceId string) bool {
	if r.InprocFlag.Enabled {
		return true
	}
	key := watched.FormatRpcServiceRegistrationKey(serviceName, appName, appInstanceId)
	return r.WatchServer.KeepEphemeral(key, hubRegistryEphemeralTTL)
}

func (r *RegistryRepo) RemoveRpcServiceRegistration(serviceName string, appName string, appInstanceId string) {
	key := watched.FormatRpcServiceRegistrationKey(serviceName, appName, appInstanceId)
	r.WatchServer.DeleteAndNotify(key)
}

func (r *RegistryRepo) SaveWebRegistration(registration *core.WebRegistration) {
	watchRegistration := toWebRegistration(registration)
	key := watched.FormatWebRegistrationKey(watchRegistration.WebSkelName, watchRegistration.AppName, watchRegistration.AppInstanceId)
	value := vcode.MustMarshalJsonS(watchRegistration)
	if r.InprocFlag.Enabled {
		r.WatchServer.SetAndNotify(key, value)
		return
	}
	r.WatchServer.SetEphemeralAndNotify(key, value, hubRegistryEphemeralTTL)
}

func (r *RegistryRepo) GetWebRegistration(name string, appName string, instanceId string) (*core.WebRegistration, bool) {
	key := watched.FormatWebRegistrationKey(name, appName, instanceId)
	value, ok := r.WatchServer.Get(key)
	if !ok {
		return nil, false
	}
	return toCoreWebRegistration(vcode.MustUnmarshalJsonS[*watched.WebRegistration](value)), true
}

func (r *RegistryRepo) KeepWebRegistration(name string, appName string, appInstanceId string) bool {
	if r.InprocFlag.Enabled {
		return true
	}
	key := watched.FormatWebRegistrationKey(name, appName, appInstanceId)
	return r.WatchServer.KeepEphemeral(key, hubRegistryEphemeralTTL)
}

func (r *RegistryRepo) RemoveWebRegistration(name string, appName string, appInstanceId string) {
	key := watched.FormatWebRegistrationKey(name, appName, appInstanceId)
	r.WatchServer.DeleteAndNotify(key)
}

func (r *RegistryRepo) saveStatus(status *core.AppStatus) {
	statusKey := watched.FormatAppStatusKey(status.Name, status.InstanceId)
	statusValue := _AppStatus{
		InstanceId:      status.InstanceId,
		Name:            status.Name,
		Version:         status.Version,
		Endpoint:        status.Endpoint,
		ServiceHandlers: status.ServiceHandlers,
		WebHandlers:     status.WebHandlers,
		EventListeners:  status.EventListeners,
		TaskRunners:     status.TaskRunners,
	}
	value := vcode.MustMarshalJsonS(statusValue)
	if r.InprocFlag.Enabled {
		r.WatchServer.SetAndNotify(statusKey, value)
		return
	}

	statusValue.ExpiresAt = timeNow().Add(hubRegistryLeaseTTL)
	value = vcode.MustMarshalJsonS(statusValue)
	r.WatchServer.SetEphemeralAndNotify(statusKey, value, hubRegistryEphemeralTTL)
	r.saveAppLease(status.Name, status.InstanceId)
}

func (r *RegistryRepo) PopExpiredAppLeases() []core.AppHeartbeat {
	if r.InprocFlag.Enabled {
		return nil
	}

	members := r.WatchServer.PopExpiredLeases(hubRegistryLeaseKey, hubRegistryLeaseSweepLimit)
	leases := make([]core.AppHeartbeat, 0, len(members))
	for _, member := range members {
		lease := vcode.MustUnmarshalJsonS[*_AppLease](member)
		status, ok := r.getAppStatus(lease.Name, lease.InstanceId)
		if !ok || timeNow().Before(status.ExpiresAt) {
			continue
		}

		leases = append(leases, core.AppHeartbeat{
			Name:       lease.Name,
			InstanceId: lease.InstanceId,
		})
	}
	return leases
}

func (r *RegistryRepo) saveAppLease(appName string, instanceId string) {
	member := vcode.MustMarshalJsonS(_AppLease{Name: appName, InstanceId: instanceId})
	r.WatchServer.KeepLease(hubRegistryLeaseKey, member, hubRegistryLeaseTTL)
}

func (r *RegistryRepo) removeAppLease(appName string, instanceId string) {
	member := vcode.MustMarshalJsonS(_AppLease{Name: appName, InstanceId: instanceId})
	r.WatchServer.RemoveLease(hubRegistryLeaseKey, member)
}

func (r *RegistryRepo) getAppStatus(appName string, instanceId string) (*_AppStatus, bool) {
	value, ok := r.WatchServer.Get(watched.FormatAppStatusKey(appName, instanceId))
	if !ok {
		return nil, false
	}
	return vcode.MustUnmarshalJsonS[*_AppStatus](value), true
}

func toCoreAppStatus(status *_AppStatus) *core.AppStatus {
	return &core.AppStatus{
		InstanceId:      status.InstanceId,
		Name:            status.Name,
		Version:         status.Version,
		Endpoint:        status.Endpoint,
		ExpiresAt:       status.ExpiresAt,
		ServiceHandlers: status.ServiceHandlers,
		WebHandlers:     status.WebHandlers,
		EventListeners:  status.EventListeners,
		TaskRunners:     status.TaskRunners,
	}
}

func toRpcServiceRegistration(registration *core.RpcServiceRegistration) *watched.RpcServiceRegistration {
	return &watched.RpcServiceRegistration{
		Endpoint:       registration.Endpoint,
		ServerIdentity: registration.ServerIdentity,
		ServiceName:    registration.ServiceName,
		Api:            registration.Api,
		AppName:        registration.AppName,
		AppVersion:     registration.AppVersion,
		AppInstanceId:  registration.AppInstanceId,
	}
}

func toCoreRpcServiceRegistration(registration *watched.RpcServiceRegistration) *core.RpcServiceRegistration {
	return &core.RpcServiceRegistration{
		Endpoint:       registration.Endpoint,
		ServerIdentity: registration.ServerIdentity,
		ServiceName:    registration.ServiceName,
		Api:            registration.Api,
		AppName:        registration.AppName,
		AppVersion:     registration.AppVersion,
		AppInstanceId:  registration.AppInstanceId,
	}
}

func toWebRegistration(registration *core.WebRegistration) *watched.WebRegistration {
	return &watched.WebRegistration{
		Endpoint:       registration.Endpoint,
		ServerIdentity: registration.ServerIdentity,
		WebSkelName:    registration.WebSkelName,
		AppName:        registration.AppName,
		AppVersion:     registration.AppVersion,
		AppInstanceId:  registration.AppInstanceId,
	}
}

func toCoreWebRegistration(registration *watched.WebRegistration) *core.WebRegistration {
	return &core.WebRegistration{
		Endpoint:       registration.Endpoint,
		ServerIdentity: registration.ServerIdentity,
		WebSkelName:    registration.WebSkelName,
		AppName:        registration.AppName,
		AppVersion:     registration.AppVersion,
		AppInstanceId:  registration.AppInstanceId,
	}
}
