package impl

import (
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/event"
)

type EventServiceServerImpl struct {
	skeled.DefaultEventServiceServer

	Manager *event.Manager `inject:""`
}

func (s *EventServiceServerImpl) EmitEvent(emission skeled.EventEmission) {
	s.Manager.EmitEvent(emission)
}
