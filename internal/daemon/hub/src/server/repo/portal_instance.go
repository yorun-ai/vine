package repo

import (
	"time"

	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

const hubPortalRegistryLeaseKey = "portal:leases"
const hubPortalRegistryEphemeralTTL = 180 * time.Second
const hubPortalRegistryLeaseTTL = 30 * time.Second
const hubPortalRegistryLeaseSweepLimit = 100

// Structs

type _PortalInstance struct {
	InstanceId string    `json:"instanceId"`
	Version    string    `json:"version"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type _PortalLease struct {
	InstanceId string `json:"instanceId"`
}

// Repo

// PortalInstanceRepo stores Portal instance liveness in the Hub watch
// store. Like application registrations, Portal instances live only in Hub
// memory and are re-registered by Portal after a Hub restart.
type PortalInstanceRepo struct {
	WatchServer *watchserver.Server             `inject:""`
	InprocFlag  *internalapp.InternalInprocFlag `inject:""`
}

func (r *PortalInstanceRepo) Save(instance *core.PortalInstance) {
	key := watched.FormatPortalInstanceKey(instance.InstanceId)
	value := _PortalInstance{
		InstanceId: instance.InstanceId,
		Version:    instance.Version,
	}
	if r.InprocFlag.Enabled {
		r.WatchServer.SetAndNotify(key, vcode.MustMarshalJsonS(value))
		return
	}

	value.ExpiresAt = timeNow().Add(hubPortalRegistryLeaseTTL)
	r.WatchServer.SetEphemeralAndNotify(key, vcode.MustMarshalJsonS(value), hubPortalRegistryEphemeralTTL)
	r.savePortalLease(instance.InstanceId)
}

func (r *PortalInstanceRepo) List() []*core.PortalInstance {
	keys := r.WatchServer.Scan(watched.FormatPortalInstancePattern())
	instances := make([]*core.PortalInstance, 0, len(keys))
	for _, key := range keys {
		value, ok := r.WatchServer.Get(key)
		if !ok {
			continue
		}
		instances = append(instances, toCorePortalInstance(vcode.MustUnmarshalJsonS[*_PortalInstance](value)))
	}
	return instances
}

func (r *PortalInstanceRepo) GetById(instanceId string) (*core.PortalInstance, bool) {
	key := watched.FormatPortalInstanceKey(instanceId)
	value, ok := r.WatchServer.Get(key)
	if !ok {
		return nil, false
	}
	return toCorePortalInstance(vcode.MustUnmarshalJsonS[*_PortalInstance](value)), true
}

func (r *PortalInstanceRepo) Keep(instanceId string) bool {
	if r.InprocFlag.Enabled {
		return true
	}

	instance, ok := r.getPortalInstance(instanceId)
	if !ok {
		return false
	}
	instance.ExpiresAt = timeNow().Add(hubPortalRegistryLeaseTTL)
	key := watched.FormatPortalInstanceKey(instanceId)
	// Refresh in place: a live instance must not publish a change event.
	r.WatchServer.SetEphemeral(key, vcode.MustMarshalJsonS(instance), hubPortalRegistryEphemeralTTL)
	r.savePortalLease(instanceId)
	return true
}

func (r *PortalInstanceRepo) Remove(instanceId string) {
	key := watched.FormatPortalInstanceKey(instanceId)
	r.removePortalLease(instanceId)
	r.WatchServer.DeleteAndNotify(key)
}

func (r *PortalInstanceRepo) PopExpiredLeases() []string {
	if r.InprocFlag.Enabled {
		return nil
	}

	members := r.WatchServer.PopExpiredLeases(hubPortalRegistryLeaseKey, hubPortalRegistryLeaseSweepLimit)
	instanceIds := make([]string, 0, len(members))
	for _, member := range members {
		lease := vcode.MustUnmarshalJsonS[*_PortalLease](member)
		instance, ok := r.getPortalInstance(lease.InstanceId)
		if !ok || timeNow().Before(instance.ExpiresAt) {
			continue
		}
		instanceIds = append(instanceIds, lease.InstanceId)
	}
	return instanceIds
}

func (r *PortalInstanceRepo) getPortalInstance(instanceId string) (*_PortalInstance, bool) {
	key := watched.FormatPortalInstanceKey(instanceId)
	value, ok := r.WatchServer.Get(key)
	if !ok {
		return nil, false
	}
	return vcode.MustUnmarshalJsonS[*_PortalInstance](value), true
}

func (r *PortalInstanceRepo) savePortalLease(instanceId string) {
	member := vcode.MustMarshalJsonS(_PortalLease{InstanceId: instanceId})
	r.WatchServer.KeepLease(hubPortalRegistryLeaseKey, member, hubPortalRegistryLeaseTTL)
}

func (r *PortalInstanceRepo) removePortalLease(instanceId string) {
	member := vcode.MustMarshalJsonS(_PortalLease{InstanceId: instanceId})
	r.WatchServer.RemoveLease(hubPortalRegistryLeaseKey, member)
}

func toCorePortalInstance(instance *_PortalInstance) *core.PortalInstance {
	return &core.PortalInstance{
		InstanceId: instance.InstanceId,
		Version:    instance.Version,
		ExpiresAt:  instance.ExpiresAt,
	}
}
