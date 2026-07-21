package meta

import "context"

type Context interface {
	context.Context

	Trace() Trace
	Initiator() Initiator
	Actor() Actor
}

type ContextImpl struct {
	context.Context
	TraceValue     Trace
	InitiatorValue Initiator
	ActorValue     Actor
}

func NewContext(ctx context.Context, trace Trace, initiator Initiator, actor Actor) Context {
	return &ContextImpl{
		Context:        ctx,
		TraceValue:     trace,
		InitiatorValue: initiator,
		ActorValue:     actor,
	}
}

func (c *ContextImpl) Trace() Trace {
	return c.TraceValue
}

func (c *ContextImpl) Initiator() Initiator {
	return c.InitiatorValue
}

func (c *ContextImpl) Actor() Actor {
	return c.ActorValue
}
