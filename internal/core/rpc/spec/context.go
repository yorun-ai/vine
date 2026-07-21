package spec

import (
	"context"

	"go.yorun.ai/vine/internal/core/meta"
)

type Context interface {
	meta.Context
	Client() meta.App
}

type ContextImpl struct {
	meta.ContextImpl
	ClientValue meta.App
}

func NewContext(ctx context.Context, trace meta.Trace, client meta.App, initiator meta.Initiator, actor meta.Actor) Context {
	return &ContextImpl{
		ContextImpl: meta.ContextImpl{
			Context:        ctx,
			TraceValue:     trace,
			InitiatorValue: initiator,
			ActorValue:     actor,
		},
		ClientValue: client,
	}
}

func (c *ContextImpl) Client() meta.App {
	return c.ClientValue
}
