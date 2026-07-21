package spec

import (
	"time"

	"go.yorun.ai/vine/internal/core/meta"
)

type Context interface {
	meta.Context
	Emitter() meta.App
	EmittedAt() time.Time
}

type ContextImpl struct {
	meta.ContextImpl
	EmitterValue   meta.App
	EmittedAtValue time.Time
}

func (c *ContextImpl) Emitter() meta.App {
	return c.EmitterValue
}

func (c *ContextImpl) EmittedAt() time.Time {
	return c.EmittedAtValue
}
