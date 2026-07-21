package client

import (
	"context"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/util/vpre"
)

type _Invoker struct {
	context             meta.Context
	clientApp           meta.App
	logger              *logger.Logger
	serverEndpoint      string
	returnIfSystemError bool

	methodInfo spec.MethodInfo
	arguments  any
	options    *_InvokeOptions

	cancel context.CancelFunc
}

func (c *Client) newInvoker(methodInfo spec.MethodInfo, arguments any, options []InvokeOption) *_Invoker {
	invoker := &_Invoker{
		context:             c.context,
		clientApp:           c.clientApp,
		logger:              c.logger,
		serverEndpoint:      c.serverEndpoint,
		returnIfSystemError: c.returnIfSystemError,
		methodInfo:          methodInfo,
		arguments:           arguments,
		options:             newInvokeOptions(),
	}

	if invoker.arguments == nil {
		invoker.arguments = &spec.EmptyArguments{}
	}
	vpre.CheckNilError(methodInfo.ValidateArguments(invoker.arguments), "arguments validation failed")

	for _, option := range options {
		option.apply(invoker.options)
	}
	vpre.Check(!(invoker.options.context != nil && invoker.options.timeoutSet), "WithContext and WithTimeout cannot be used together")
	return invoker
}

func (i *_Invoker) invoke() (result any, err ex.Error) {
	var rpcResponse spec.Response

	defer i.cleanup()
	rpcRequest := i.buildRequest()
	logSpan := rpclog.Noop()
	if !inproc.IsEndpoint(i.serverEndpoint) || rpclog.IsInprocClientLogEnabled() {
		logSpan = rpclog.StartClientInvoke(i.logger, rpcRequest.Trace(), i.methodInfo, i.serverEndpoint)
	}
	defer func() { logSpan.FinishWithResponse(err, rpcResponse) }()

	rpcResponse, err = i.roundTrip(rpcRequest)
	result, err = i.parseResponse(rpcResponse, err)

	if err != nil && !i.returnIfSystemError && err.Type() == ex.SystemError {
		panic(err)
	}

	return result, err
}

func (i *_Invoker) cleanup() {
	if i.cancel != nil {
		i.cancel()
	}
}

func (i *_Invoker) buildRequest() spec.Request {
	parentContext := i.context.(context.Context)
	if i.options.context != nil {
		parentContext = i.options.context
	}

	reqContext := parentContext
	if i.options.context == nil {
		var cancel context.CancelFunc
		reqContext, cancel = context.WithTimeout(parentContext, i.options.timeout)
		i.cancel = cancel
	}

	return &spec.RequestImpl{
		ContextValue:    reqContext,
		TraceValue:      i.context.Trace().NewChildTrace(),
		ActorValue:      i.context.Actor(),
		InitiatorValue:  i.context.Initiator(),
		ClientValue:     i.clientApp,
		MethodInfoValue: i.methodInfo,
		ArgumentsValue:  i.arguments,
	}
}

func (i *_Invoker) roundTrip(rpcRequest spec.Request) (spec.Response, ex.Error) {
	if inproc.IsEndpoint(i.serverEndpoint) {
		return inproc.RoundTrip(i.serverEndpoint, rpcRequest)
	}
	return http.RoundTrip(i.serverEndpoint, rpcRequest)
}

func (i *_Invoker) parseResponse(rpcResponse spec.Response, err ex.Error) (any, ex.Error) {
	if err != nil {
		return nil, err
	}

	if rpcResponse.Error() != nil && rpcResponse.Error().Code() != ex.OK {
		return nil, rpcResponse.Error()
	}

	result := rpcResponse.Result()
	if err := i.methodInfo.ValidateResult(result); err != nil {
		return nil, ex.New(ex.UnexpectedResponse, err.Error())
	}
	return result, nil
}
