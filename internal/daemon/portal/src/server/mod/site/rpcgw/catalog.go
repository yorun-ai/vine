package rpcgw

import (
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/desc"
)

// apiSnapshot pairs descriptors with the site's current API exposure policy.
func (g *RpcGateway) apiSnapshot() (desc.Snapshot, map[string]*skeldesc.Service) {
	g.mutex.Lock()
	actor := g.actorSkelName
	configured := make([]string, 0, len(g.watchers))
	for name := range g.watchers {
		configured = append(configured, name)
	}
	g.mutex.Unlock()
	snapshot := g.desc.Snapshot()
	visible := map[string]*skeldesc.Service{}
	if snapshot.Actors[actor] != nil {
		for _, name := range configured {
			service := snapshot.Services[name]
			if service != nil && service.Api && skel.HasActorAudience(service.Audiences, actor) {
				visible[name] = service
			}
		}
	}
	return snapshot, visible
}
