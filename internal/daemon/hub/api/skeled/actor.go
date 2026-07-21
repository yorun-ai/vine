package skeled

import (
	"go.yorun.ai/vine/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
)

func init() {
	meta.RegisterActor(meta.ActorSpec{
		Name:     "AdminActor",
		SkelName: "vine.hub.AdminActor",
		Hash:     "e3c26b2d",
	})
}

type AdminActor struct {
	skel.ActorBase
}

func (AdminActor) Name() string {
	return "AdminActor"
}

func (AdminActor) SkelName() string {
	return "vine.hub.AdminActor"
}

func (AdminActor) Vias() []skel.ActorVia {
	return []skel.ActorVia{
		skel.ActorViaClient,
	}
}
