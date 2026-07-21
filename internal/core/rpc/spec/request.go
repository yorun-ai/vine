package spec

import (
	"context"

	"go.yorun.ai/vine/internal/core/meta"
)

type Request interface {
	Context() context.Context
	Cancel()

	Trace() meta.Trace
	Actor() meta.Actor
	Initiator() meta.Initiator

	Client() meta.App

	MethodInfo() MethodInfo
	MethodImpl() MethodImpl

	Arguments() any
	PositionalArguments() []any
}

type RequestImpl struct {
	ContextValue context.Context
	CancelValue  context.CancelFunc

	TraceValue     meta.Trace
	ActorValue     meta.Actor
	InitiatorValue meta.Initiator

	ClientValue meta.App

	MethodInfoValue MethodInfo
	MethodImplValue MethodImpl
	ArgumentsValue  any
}

func (r *RequestImpl) Context() context.Context {
	return r.ContextValue
}

func (r *RequestImpl) Cancel() {
	if r.CancelValue != nil {
		r.CancelValue()
	}
}

func (r *RequestImpl) Trace() meta.Trace {
	return r.TraceValue
}

func (r *RequestImpl) Actor() meta.Actor {
	return r.ActorValue
}

func (r *RequestImpl) Initiator() meta.Initiator {
	return r.InitiatorValue
}

func (r *RequestImpl) Client() meta.App {
	return r.ClientValue
}

func (r *RequestImpl) MethodInfo() MethodInfo {
	return r.MethodInfoValue
}

func (r *RequestImpl) MethodImpl() MethodImpl {
	return r.MethodImplValue
}

func (r *RequestImpl) Arguments() any {
	return r.ArgumentsValue
}

func (r *RequestImpl) PositionalArguments() []any {
	return r.MethodInfo().PositionArguments(r.ArgumentsValue)
}
