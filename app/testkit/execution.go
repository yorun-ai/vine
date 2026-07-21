package testkit

import (
	"context"
	"reflect"

	"go.yorun.ai/vine/core/logger"
	"go.yorun.ai/vine/core/meta"
	"go.yorun.ai/vine/core/rpc"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"
)

// ExecutionOption configures metadata for calls made by an Execution.
type ExecutionOption struct {
	// Context controls cancellation and deadlines for the execution.
	Context context.Context
	// Trace identifies the distributed trace; a new initial trace is used when nil.
	Trace meta.Trace
	// Initiator describes the original caller, when one is available.
	Initiator meta.Initiator
	// Actor is the calling identity; an absent actor is used when nil.
	Actor meta.Actor
}

// Execution represents one test call context against a Runtime.
type Execution struct {
	runtime   *Runtime
	context   context.Context
	trace     meta.Trace
	initiator meta.Initiator
	actor     meta.Actor
}

// NewExecution creates a call execution associated with r.
func (r *Runtime) NewExecution(option ExecutionOption) *Execution {
	ctx := option.Context
	if ctx == nil {
		ctx = context.Background()
	}
	trace := option.Trace
	if trace == nil {
		trace = meta.InitialTrace()
	}
	actor := option.Actor
	if actor == nil {
		actor = meta.NewAbsentActor()
	}
	return &Execution{
		runtime:   r,
		context:   ctx,
		trace:     trace,
		initiator: option.Initiator,
		actor:     actor,
	}
}

// NewClient constructs a generated ordinary Rpc client bound to execution.
func NewClient[C any](execution *Execution) C {
	clientType := reflect.TypeFor[C]()
	serviceInfo, ok := rpcspec.GetServiceInfoByClientType(clientType)
	vpre.Check(ok, "rpc service client %s is not registered", clientType)
	vpre.CheckNotNil(serviceInfo.ClientCtor(), "rpc service client ctor is not registered")
	vpre.CheckNotNil(serviceInfo.ERClientCtor(), "rpc service er client ctor is not registered")

	erClient := reflect.ValueOf(serviceInfo.ERClientCtor()).Call([]reflect.Value{
		reflect.ValueOf(execution.newClient()),
	})[0]
	client := reflect.ValueOf(serviceInfo.ClientCtor()).Call([]reflect.Value{erClient})[0]
	return client.Interface().(C)
}

// NewClientER constructs a generated error-returning Rpc client bound to execution.
func NewClientER[C any](execution *Execution) C {
	clientType := reflect.TypeFor[C]()
	serviceInfo, ok := rpcspec.GetServiceInfoByERClientType(clientType)
	vpre.Check(ok, "rpc service er client %s is not registered", clientType)
	vpre.CheckNotNil(serviceInfo.ERClientCtor(), "rpc service er client ctor is not registered")

	erClient := reflect.ValueOf(serviceInfo.ERClientCtor()).Call([]reflect.Value{
		reflect.ValueOf(execution.newClient()),
	})[0]
	return erClient.Interface().(C)
}

func (e *Execution) newClient() *rpc.Client {
	return rpc.NewClient(rpc.ClientOption{
		Context:        rpc.NewContext(e.context, e.trace, e.runtime.clientApp, e.initiator, e.actor),
		ClientApp:      e.runtime.clientApp,
		Logger:         logger.NewLogger(logger.GlobalOption()),
		ServerEndpoint: rpcProxyOutEndpoint,
	})
}
