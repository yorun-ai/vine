package server

import (
	"net/http"
	"reflect"
	"runtime/debug"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	httptrans "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/util/vpre"
)

var rpcServerLogger = logger.NewLogger(logger.GlobalOption())

type Option struct {
	App            meta.App
	MuteVerboseLog bool
	HandlerTypes   []reflect.Type
	Executor       Executor
}

type Executor interface {
	Init(infoDict spec.ImplDict)
	Execute(rpcContext spec.Context, methodImpl spec.MethodImpl, arguments []any) (any, ex.Error)
}

type Server struct {
	opt *Option

	implDict *spec.ImplDict
	executor Executor
}

func New(opt Option) *Server {
	server := &Server{
		opt:      &opt,
		executor: opt.Executor,
	}
	server.init()
	return server
}

func (s *Server) init() {
	vpre.Check(len(s.opt.HandlerTypes) > 0, "no rpc handler type found")
	s.implDict = spec.NewImplDict()
	for _, handlerType := range s.opt.HandlerTypes {
		s.implDict.Add(handlerType)
	}
	if s.executor == nil {
		s.executor = NewDefaultExecutor()
	}
	s.executor.Init(*s.implDict)
}

func (s *Server) GetServiceInfos() []spec.ServiceInfo {
	var serviceInfos []spec.ServiceInfo
	s.implDict.IterateServiceImpl(func(info spec.ServiceImpl) {
		serviceInfos = append(serviceInfos, info.Info())
	})
	return serviceInfos
}

func (s *Server) handle(rpcRequest spec.Request) (response spec.Response) {
	var trace meta.Trace
	var result any
	var err ex.Error

	logSpan := rpclog.Noop()
	defer func() { logSpan.Finish(err) }()

	defer func() {
		if reErr := recover(); reErr != nil {
			switch casted := reErr.(type) {
			case ex.Error:
				err = casted
			default:
				rpcServerLogger.Error("rpc server recovered panic",
					"panic", reErr,
					"stack", string(debug.Stack()),
					"rpcMethod", rpcRequest.MethodInfo().Name(),
					"rpcMethodSkel", rpcRequest.MethodInfo().SkelName(),
					"clientName", rpcRequest.Client().Name(),
					"clientVersion", rpcRequest.Client().Version(),
					"clientInstanceId", rpcRequest.Client().InstanceId(),
					"serverName", s.opt.App.Name(),
					"serverVersion", s.opt.App.Version(),
					"serverInstanceId", s.opt.App.InstanceId(),
				)
				err = ex.NewInternal()
			}
		}

		if err == nil {
			err = ex.NewOK()
		}

		if err.Code().IsUnresponsive() {
			err = ex.NewInternal()
		}

		response = &spec.ResponseImpl{
			ServerValue: s.opt.App,
			MethodValue: rpcRequest.MethodInfo(),
			ResultValue: result,
			ErrorValue:  err,
		}
	}()

	trace = rpcRequest.Trace().NewChildTrace()
	rpcContext := &spec.ContextImpl{
		ContextImpl: meta.ContextImpl{
			Context:        rpcRequest.Context(),
			TraceValue:     trace,
			InitiatorValue: rpcRequest.Initiator(),
			ActorValue:     rpcRequest.Actor(),
		},
		ClientValue: rpcRequest.Client(),
	}
	logSpan = rpclog.StartServerHandle(rpcServerLogger, trace, rpcRequest.MethodInfo(), rpcRequest.Client(), s.opt.App)
	arguments := rpcRequest.PositionalArguments()
	methodImpl := rpcRequest.MethodImpl()

	result, err = s.executor.Execute(rpcContext, methodImpl, arguments)
	return
}

// Rpc Handler

func (s *Server) RpcHandler() spec.RpcHandler {
	return spec.RpcHandlerFunc(s.serveRpc)
}

func (s *Server) serveRpc(rpcRequest spec.Request) spec.Response {
	methodInfo := rpcRequest.MethodInfo()
	rpcRequest.(*spec.RequestImpl).ArgumentsValue = spec.CloneInprocRequestArguments(rpcRequest.Arguments(), methodInfo)
	if err := methodInfo.ValidateArguments(rpcRequest.Arguments()); err != nil {
		return &spec.ResponseImpl{
			ServerValue: s.opt.App,
			MethodValue: methodInfo,
			ErrorValue:  ex.New(ex.InvalidRequest, err.Error()),
		}
	}

	methodImpl, err := s.implDict.GetMethodImplByInfo(rpcRequest.MethodInfo())
	if err != nil {
		return &spec.ResponseImpl{
			ServerValue: s.opt.App,
			MethodValue: rpcRequest.MethodInfo(),
			ErrorValue:  ex.New(ex.InvalidRequest, err.Error()),
		}
	}

	rpcRequest.(*spec.RequestImpl).MethodImplValue = methodImpl
	rpcResponse := s.handle(rpcRequest)

	rpcResponse.(*spec.ResponseImpl).ResultValue = spec.CloneInprocResponseResult(rpcResponse.Result(), methodInfo)
	return rpcResponse
}

// HTTP Handler

func (s *Server) HTTPHandler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	rpcRequest, err := httptrans.DecodeRequest(r)
	if err != nil {
		_ = httptrans.WriteRequestErrorResponse(w, r, s.opt.App, ex.New(ex.InvalidRequest, err.Error()))
		return
	}
	defer rpcRequest.Cancel()

	methodImpl, err := s.implDict.GetMethodImplByInfo(rpcRequest.MethodInfo())
	if err != nil {
		_ = httptrans.WriteRequestErrorResponse(w, r, s.opt.App, ex.New(ex.InvalidRequest, err.Error()))
		return
	}

	rpcRequest.(*spec.RequestImpl).MethodImplValue = methodImpl
	rpcResponse := s.handle(rpcRequest)
	if err := httptrans.WriteResponse(w, r, rpcResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		rpcServerLogger.Error("response body write failed", "error", err)
	}
}
