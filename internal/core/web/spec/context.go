package spec

import (
	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/core/meta"
)

type Context interface {
	meta.Context

	Gin() *gin.Context
	Route() Route
}

type _Context struct {
	meta.Context

	ginCtx *gin.Context
	route  Route
}

func NewContext(ginCtx *gin.Context, route Route, trace meta.Trace, initiator meta.Initiator, actor meta.Actor) Context {
	return &_Context{
		Context: meta.NewContext(ginCtx.Request.Context(), trace, initiator, actor),
		ginCtx:  ginCtx,
		route:   route,
	}
}

func (c *_Context) Gin() *gin.Context {
	return c.ginCtx
}

func (c *_Context) Route() Route {
	return c.route
}
