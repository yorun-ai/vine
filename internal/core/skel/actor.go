package skel

import "go.yorun.ai/skel/descriptor"

type Actor interface {
	Name() string
	SkelName() string
	Vias() []descriptor.ActorViaKind

	mustBeActor()
}

type ActorBase struct{}

func (ActorBase) Name() string {
	return ""
}

func (ActorBase) SkelName() string {
	return ""
}

func (ActorBase) Vias() []descriptor.ActorViaKind {
	return nil
}

func (ActorBase) mustBeActor() {}
