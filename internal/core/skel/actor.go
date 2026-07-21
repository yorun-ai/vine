package skel

type Actor interface {
	Name() string
	SkelName() string
	Vias() []ActorVia

	mustBeActor()
}

type ActorBase struct{}

func (ActorBase) Name() string {
	return ""
}

func (ActorBase) SkelName() string {
	return ""
}

func (ActorBase) Vias() []ActorVia {
	return nil
}

func (ActorBase) mustBeActor() {}

type ActorVia string

const (
	ActorViaClient  ActorVia = "client"
	ActorViaAgent   ActorVia = "agent"
	ActorViaOpenAPI ActorVia = "openapi"
)
