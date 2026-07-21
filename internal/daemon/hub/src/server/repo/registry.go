package repo

import (
	"cmp"
	"time"

	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
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

type RedisRegistryRepo struct {
	RedisServer *redisserver.Server             `inject:""`
	InprocFlag  *internalapp.InternalInprocFlag `inject:""`
}

func (r *RedisRegistryRepo) SaveAppStatus(status *core.AppStatus) {
	r.saveStatus(status)
}

func (r *RedisRegistryRepo) ListAppStatuses() []*core.AppStatus {
	keys := r.RedisServer.Scan(redised.FormatAppStatusPattern())
	items := make([]*core.AppStatus, 0, len(keys))
	for _, key := range keys {
		value, ok := r.RedisServer.Get(key)
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

func (r *RedisRegistryRepo) GetAppStatus(appName string, instanceId string) (*core.AppStatus, bool) {
	status, ok := r.getAppStatus(appName, instanceId)
	if !ok {
		return nil, false
	}
	return toCoreAppStatus(status), true
}

func (r *RedisRegistryRepo) KeepAppStatus(appName string, instanceId string) bool {
	if r.InprocFlag.Enabled {
		return true
	}
	status, ok := r.getAppStatus(appName, instanceId)
	if !ok {
		return false
	}
	status.ExpiresAt = timeNow().Add(hubRegistryLeaseTTL)
	key := redised.FormatAppStatusKey(appName, instanceId)
	r.RedisServer.SetEphemeral(key, vcode.MustMarshalJsonS(status), hubRegistryEphemeralTTL)
	r.saveAppLease(appName, instanceId)
	return true
}

func (r *RedisRegistryRepo) RemoveAppStatus(appName string, instanceId string) {
	key := redised.FormatAppStatusKey(appName, instanceId)
	r.removeAppLease(appName, instanceId)
	r.RedisServer.DeleteAndNotify(key)
}

func (r *RedisRegistryRepo) SaveRpcServiceRegistration(registration *core.RpcServiceRegistration) {
	redisRegistration := toRpcServiceRegistration(registration)
	key := redised.FormatRpcServiceRegistrationKey(redisRegistration.ServiceName, redisRegistration.AppName, redisRegistration.AppInstanceId)
	value := vcode.MustMarshalJsonS(redisRegistration)
	if r.InprocFlag.Enabled {
		r.RedisServer.SetAndNotify(key, value)
		return
	}
	r.RedisServer.SetEphemeralAndNotify(key, value, hubRegistryEphemeralTTL)
}

func (r *RedisRegistryRepo) GetRpcServiceRegistration(serviceName string, appName string, instanceId string) (*core.RpcServiceRegistration, bool) {
	key := redised.FormatRpcServiceRegistrationKey(serviceName, appName, instanceId)
	value, ok := r.RedisServer.Get(key)
	if !ok {
		return nil, false
	}
	return toCoreRpcServiceRegistration(vcode.MustUnmarshalJsonS[*redised.RpcServiceRegistration](value)), true
}

func (r *RedisRegistryRepo) KeepRpcServiceRegistration(serviceName string, appName string, appInstanceId string) bool {
	if r.InprocFlag.Enabled {
		return true
	}
	key := redised.FormatRpcServiceRegistrationKey(serviceName, appName, appInstanceId)
	return r.RedisServer.KeepEphemeral(key, hubRegistryEphemeralTTL)
}

func (r *RedisRegistryRepo) RemoveRpcServiceRegistration(serviceName string, appName string, appInstanceId string) {
	key := redised.FormatRpcServiceRegistrationKey(serviceName, appName, appInstanceId)
	r.RedisServer.DeleteAndNotify(key)
}

func (r *RedisRegistryRepo) SaveWebRegistration(registration *core.WebRegistration) {
	redisRegistration := toWebRegistration(registration)
	key := redised.FormatWebRegistrationKey(redisRegistration.WebSkelName, redisRegistration.AppName, redisRegistration.AppInstanceId)
	value := vcode.MustMarshalJsonS(redisRegistration)
	if r.InprocFlag.Enabled {
		r.RedisServer.SetAndNotify(key, value)
		return
	}
	r.RedisServer.SetEphemeralAndNotify(key, value, hubRegistryEphemeralTTL)
}

func (r *RedisRegistryRepo) GetWebRegistration(name string, appName string, instanceId string) (*core.WebRegistration, bool) {
	key := redised.FormatWebRegistrationKey(name, appName, instanceId)
	value, ok := r.RedisServer.Get(key)
	if !ok {
		return nil, false
	}
	return toCoreWebRegistration(vcode.MustUnmarshalJsonS[*redised.WebRegistration](value)), true
}

func (r *RedisRegistryRepo) KeepWebRegistration(name string, appName string, appInstanceId string) bool {
	if r.InprocFlag.Enabled {
		return true
	}
	key := redised.FormatWebRegistrationKey(name, appName, appInstanceId)
	return r.RedisServer.KeepEphemeral(key, hubRegistryEphemeralTTL)
}

func (r *RedisRegistryRepo) RemoveWebRegistration(name string, appName string, appInstanceId string) {
	key := redised.FormatWebRegistrationKey(name, appName, appInstanceId)
	r.RedisServer.DeleteAndNotify(key)
}

func (r *RedisRegistryRepo) saveStatus(status *core.AppStatus) {
	statusKey := redised.FormatAppStatusKey(status.Name, status.InstanceId)
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
		r.RedisServer.SetAndNotify(statusKey, value)
		return
	}

	statusValue.ExpiresAt = timeNow().Add(hubRegistryLeaseTTL)
	value = vcode.MustMarshalJsonS(statusValue)
	r.RedisServer.SetEphemeralAndNotify(statusKey, value, hubRegistryEphemeralTTL)
	r.saveAppLease(status.Name, status.InstanceId)
}

func (r *RedisRegistryRepo) PopExpiredAppLeases() []core.AppHeartbeat {
	if r.InprocFlag.Enabled {
		return nil
	}

	members := r.RedisServer.PopExpiredLeases(hubRegistryLeaseKey, hubRegistryLeaseSweepLimit)
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

func (r *RedisRegistryRepo) saveAppLease(appName string, instanceId string) {
	member := vcode.MustMarshalJsonS(_AppLease{Name: appName, InstanceId: instanceId})
	r.RedisServer.KeepLease(hubRegistryLeaseKey, member, hubRegistryLeaseTTL)
}

func (r *RedisRegistryRepo) removeAppLease(appName string, instanceId string) {
	member := vcode.MustMarshalJsonS(_AppLease{Name: appName, InstanceId: instanceId})
	r.RedisServer.RemoveLease(hubRegistryLeaseKey, member)
}

func (r *RedisRegistryRepo) getAppStatus(appName string, instanceId string) (*_AppStatus, bool) {
	value, ok := r.RedisServer.Get(redised.FormatAppStatusKey(appName, instanceId))
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

func toRpcServiceRegistration(registration *core.RpcServiceRegistration) *redised.RpcServiceRegistration {
	return &redised.RpcServiceRegistration{
		Endpoint:      registration.Endpoint,
		ServiceName:   registration.ServiceName,
		AppName:       registration.AppName,
		AppVersion:    registration.AppVersion,
		AppInstanceId: registration.AppInstanceId,
	}
}

func toCoreRpcServiceRegistration(registration *redised.RpcServiceRegistration) *core.RpcServiceRegistration {
	return &core.RpcServiceRegistration{
		Endpoint:      registration.Endpoint,
		ServiceName:   registration.ServiceName,
		AppName:       registration.AppName,
		AppVersion:    registration.AppVersion,
		AppInstanceId: registration.AppInstanceId,
	}
}

func toWebRegistration(registration *core.WebRegistration) *redised.WebRegistration {
	return &redised.WebRegistration{
		Endpoint:      registration.Endpoint,
		WebSkelName:   registration.WebSkelName,
		AppName:       registration.AppName,
		AppVersion:    registration.AppVersion,
		AppInstanceId: registration.AppInstanceId,
	}
}

func toCoreWebRegistration(registration *redised.WebRegistration) *core.WebRegistration {
	return &core.WebRegistration{
		Endpoint:      registration.Endpoint,
		WebSkelName:   registration.WebSkelName,
		AppName:       registration.AppName,
		AppVersion:    registration.AppVersion,
		AppInstanceId: registration.AppInstanceId,
	}
}
