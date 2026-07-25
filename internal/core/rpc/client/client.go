package client

import (
	"context"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"
)

type Option struct {
	Context             meta.Context
	ClientApp           meta.App
	Logger              *logger.Logger
	ReturnIfSystemError bool
	ServerEndpoint      string
}

type Client struct {
	context             meta.Context
	clientApp           meta.App
	logger              *logger.Logger
	returnIfSystemError bool
	serverEndpoint      string
}

func New(option Option) *Client {
	vpre.CheckNotNil(option.Context, "rpc client context cannot be nil")
	vpre.CheckNotNil(option.Logger, "rpc client logger cannot be nil")
	return &Client{
		context:             option.Context,
		clientApp:           option.ClientApp,
		logger:              option.Logger,
		serverEndpoint:      option.ServerEndpoint,
		returnIfSystemError: option.ReturnIfSystemError,
	}
}

func (c *Client) Invoke(methodInfo spec.MethodInfo, arguments any, options ...InvokeOption) (any, ex.Error) {
	startedAt := time.Now()
	invoker := c.newInvoker(methodInfo, arguments, options)
	if err := methodInfo.ValidateArguments(invoker.arguments); err != nil {
		logErr := ex.New(ex.InvalidRequest, err.Error())
		rpclog.ClientRejected(c.logger, startedAt, c.context.Trace(), methodInfo, c.serverEndpoint, invoker.arguments, logErr)
		vpre.CheckNilError(err, "arguments validation failed")
	}
	return invoker.invoke()
}

type InvokeOption interface {
	apply(options *_InvokeOptions)
}

type _InvokeOptionFunc func(options *_InvokeOptions)

func (f _InvokeOptionFunc) apply(options *_InvokeOptions) {
	f(options)
}

// WithContext overrides only the parent request context lifecycle.
// It does not change the RPC trace, actor, or initiator metadata sourced from the client.
func WithContext(ctx context.Context) InvokeOption {
	return _InvokeOptionFunc(func(options *_InvokeOptions) {
		options.context = ctx
	})
}

func WithTimeout(duration time.Duration) InvokeOption {
	vpre.Check(duration > 0, "rpc invoke timeout must be greater than 0")
	return _InvokeOptionFunc(func(options *_InvokeOptions) {
		options.timeout = duration
		options.timeoutSet = true
	})
}

type _InvokeOptions struct {
	context    context.Context
	timeout    time.Duration
	timeoutSet bool
}

const defaultRequestTimeout = time.Second * 30

func newInvokeOptions() *_InvokeOptions {
	return &_InvokeOptions{
		context:    nil,
		timeout:    defaultRequestTimeout,
		timeoutSet: false,
	}
}
