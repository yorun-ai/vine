package server

import (
	"net/http"
	"reflect"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	httptrans "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/util/vpre"
)

type Option struct {
	App meta.App
	// LogicalAppName is the stable ApplicationSpec name used for scoped lifecycle logging.
	LogicalAppName string
	// Logger overrides dynamic App and Rpc-server lifecycle logging when non-nil.
	Logger         *logger.Logger
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
	log      *logger.Logger
	infraLog *logger.Logger
}

func New(opt Option) *Server {
	logger.FreezePayloadPolicies()
	server := &Server{
		opt:      &opt,
		executor: opt.Executor,
	}
	if opt.Logger != nil {
		server.log = opt.Logger
		server.infraLog = opt.Logger
	} else {
		appName := opt.LogicalAppName
		if appName == "" && opt.App != nil {
			appName = opt.App.Name()
		}
		server.log = logger.NewScopedLogger(logger.Scope{AppName: appName, Subsystem: logger.SubsystemRpcServer})
		server.infraLog = logger.NewGlobalLogger()
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
	var logErr ex.Error

	logSpan := rpclog.Noop()
	defer func() { logSpan.FinishServer(logErr, result) }()

	defer func() {
		if reErr := recover(); reErr != nil {
			logErr = ex.RecoverExecution(reErr)
		}

		if logErr == nil {
			logErr = ex.NewOK()
		}

		responseErr := logErr
		if responseErr.Code().IsUnresponsive() {
			responseErr = ex.NewInternal()
		}

		response = &spec.ResponseImpl{
			ServerValue: s.opt.App,
			MethodValue: rpcRequest.MethodInfo(),
			ResultValue: result,
			ErrorValue:  responseErr,
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
	logArguments := rpcRequest.Arguments()
	if !rpcRequest.MethodInfo().HasArguments() {
		logArguments = new(spec.EmptyArguments)
	}
	logSpan = rpclog.StartServerHandle(s.log, trace, rpcRequest.MethodInfo(), rpcRequest.Client(), s.opt.App, logArguments)
	arguments := rpcRequest.PositionalArguments()
	methodImpl := rpcRequest.MethodImpl()

	result, logErr = s.executor.Execute(rpcContext, methodImpl, arguments)
	return
}

// Rpc Handler

func (s *Server) RpcHandler() spec.RpcHandler {
	return spec.RpcHandlerFunc(s.serveRpc)
}

func (s *Server) serveRpc(rpcRequest spec.Request) spec.Response {
	startedAt := time.Now()
	methodInfo := rpcRequest.MethodInfo()
	rpcRequest.(*spec.RequestImpl).ArgumentsValue = spec.CloneInprocRequestArguments(rpcRequest.Arguments(), methodInfo)
	if err := methodInfo.ValidateArguments(rpcRequest.Arguments()); err != nil {
		rejectedErr := ex.New(ex.InvalidRequest, err.Error())
		rpclog.ServerRejected(s.log, startedAt, rpcRequest.Trace(), methodInfo, rpcRequest.Client(), s.opt.App, rejectedErr)
		return &spec.ResponseImpl{
			ServerValue: s.opt.App,
			MethodValue: methodInfo,
			ErrorValue:  rejectedErr,
		}
	}

	methodImpl, err := s.implDict.GetMethodImplByInfo(rpcRequest.MethodInfo())
	if err != nil {
		rejectedErr := ex.New(ex.InvalidRequest, err.Error())
		rpclog.ServerRejected(s.log, startedAt, rpcRequest.Trace(), methodInfo, rpcRequest.Client(), s.opt.App, rejectedErr)
		return &spec.ResponseImpl{
			ServerValue: s.opt.App,
			MethodValue: rpcRequest.MethodInfo(),
			ErrorValue:  rejectedErr,
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
	startedAt := time.Now()
	rpcRequest, diagnostic, err := httptrans.DecodeRequestWithDiagnostic(r)
	if err != nil {
		rejectedErr := ex.New(ex.InvalidRequest, err.Error())
		rpclog.ServerRejected(s.log, startedAt, diagnostic.Trace, diagnostic.Method, diagnostic.Client, s.opt.App, rejectedErr,
			diagnostic.ServiceSkelName, diagnostic.MethodSkelName)
		_ = httptrans.WriteRequestErrorResponse(w, r, s.opt.App, rejectedErr)
		return
	}
	defer rpcRequest.Cancel()

	methodImpl, err := s.implDict.GetMethodImplByInfo(rpcRequest.MethodInfo())
	if err != nil {
		rejectedErr := ex.New(ex.InvalidRequest, err.Error())
		rpclog.ServerRejected(s.log, startedAt, rpcRequest.Trace(), rpcRequest.MethodInfo(), rpcRequest.Client(), s.opt.App, rejectedErr)
		_ = httptrans.WriteRequestErrorResponse(w, r, s.opt.App, rejectedErr)
		return
	}

	rpcRequest.(*spec.RequestImpl).MethodImplValue = methodImpl
	rpcResponse := s.handle(rpcRequest)
	if err := httptrans.WriteResponse(w, r, rpcResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.infraLog.Error("response body write failed", "error", err)
	}
}
