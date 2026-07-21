package spec

import (
	"time"

	"go.yorun.ai/vine/internal/core/meta"
)

type Context interface {
	meta.Context
	Launcher() meta.App
	LaunchedAt() time.Time
}

type ContextImpl struct {
	meta.ContextImpl
	LauncherValue   meta.App
	LaunchedAtValue time.Time
}

func (c *ContextImpl) Launcher() meta.App {
	return c.LauncherValue
}

func (c *ContextImpl) LaunchedAt() time.Time {
	return c.LaunchedAtValue
}
