package rpc

import (
	"context"
	"reflect"
	"time"

	"go.yorun.ai/vine/core/di"
	"go.yorun.ai/vine/core/meta"
	internalclient "go.yorun.ai/vine/internal/core/rpc/client"
	internalrpclog "go.yorun.ai/vine/internal/core/rpc/log"
	internalserver "go.yorun.ai/vine/internal/core/rpc/server"
	internalspec "go.yorun.ai/vine/internal/core/rpc/spec"
)

// ClientOption configures an Rpc client and its transport.
type ClientOption = internalclient.Option

// Client invokes registered Rpc methods.
type Client = internalclient.Client

// InvokeOption configures one Rpc invocation.
type InvokeOption = internalclient.InvokeOption

// ServerOption configures an Rpc server.
type ServerOption = internalserver.Option

// Executor invokes a registered Rpc method.
type Executor = internalserver.Executor

// Server receives Rpc requests and dispatches them to an Executor.
type Server = internalserver.Server

// Context carries metadata for one Rpc invocation.
type Context = internalspec.Context

// ServiceSpecType describes whether a service provides a client, server, or both.
type ServiceSpecType = internalspec.ServiceSpecType

// ServiceSpec describes a generated Rpc service contract.
type ServiceSpec = internalspec.ServiceSpec

// MethodSpec describes one method in a ServiceSpec.
type MethodSpec = internalspec.MethodSpec

// ServiceInfo is runtime metadata derived from a ServiceSpec.
type ServiceInfo = internalspec.ServiceInfo

// MethodInfo is runtime metadata for one registered method.
type MethodInfo = internalspec.MethodInfo

// Request is the transport envelope for an Rpc request.
type Request = internalspec.Request

// Response is the transport envelope for an Rpc response.
type Response = internalspec.Response

// CheckValueNotNil validates that a generated request value is not nil.
var CheckValueNotNil = internalspec.CheckValueNotNil

// JoinPath appends a field name to a generated validation path.
var JoinPath = internalspec.JoinPath

// JoinIndex appends a list index to a generated validation path.
var JoinIndex = internalspec.JoinIndex

// JoinMapKey appends a map key to a generated validation path.
var JoinMapKey = internalspec.JoinMapKey

const (
	// ServiceSpecTypeClient identifies a client-only service contract.
	ServiceSpecTypeClient = internalspec.ServiceSpecTypeClient
	// ServiceSpecTypeServer identifies a server-only service contract.
	ServiceSpecTypeServer = internalspec.ServiceSpecTypeServer
	// ServiceSpecTypeBoth identifies a contract providing both client and server metadata.
	ServiceSpecTypeBoth = internalspec.ServiceSpecTypeBoth
)

// NewClient creates an Rpc client from option.
func NewClient(option ClientOption) *Client {
	return internalclient.New(option)
}

// WithContext overrides the context used by one invocation.
func WithContext(ctx context.Context) InvokeOption {
	return internalclient.WithContext(ctx)
}

// WithTimeout sets the maximum duration of one invocation.
func WithTimeout(duration time.Duration) InvokeOption {
	return internalclient.WithTimeout(duration)
}

// NewContext creates an Rpc context from Go and Vine call metadata.
func NewContext(ctx context.Context, trace meta.Trace, client meta.App, initiator meta.Initiator, actor meta.Actor) Context {
	return internalspec.NewContext(ctx, trace, client, initiator, actor)
}

// NewServer creates an Rpc server from option.
func NewServer(option ServerOption) *Server {
	return internalserver.New(option)
}

// NewContainerExecutor creates an Executor backed by a DI container and filter chain.
func NewContainerExecutor(filterTypes []reflect.Type, bindAppliers []di.BindApplier) Executor {
	return internalserver.NewContainerExecutor(filterTypes, bindAppliers)
}

// NewDefaultExecutor creates the framework's default Rpc executor.
func NewDefaultExecutor() Executor {
	return internalserver.NewDefaultExecutor()
}

// Register adds serviceSpec to the process-wide Rpc registry.
func Register(serviceSpec *ServiceSpec) {
	internalspec.Register(serviceSpec)
}

// MuteSuccessLog suppresses successful invocation logs for method.
func MuteSuccessLog(method any) {
	internalrpclog.MuteSuccessLog(method)
}
