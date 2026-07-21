package skeled

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/ex"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
)

func init() {
	rpcspec.Register(_ConsoleServiceSpec)
	rpcspec.Register(_EventServiceSpec)
	rpcspec.Register(_TaskServiceSpec)
}

// ConsoleServiceServer App's console service, called by Link

// ConsoleService / Spec

var (
	_ConsoleServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "ConsoleService",
		SkelName:          "vine.app.ConsoleService",
		Hash:              "edd76e05",
		ServerType:        reflect.TypeFor[ConsoleServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultConsoleServiceServer](),
		ClientType:        reflect.TypeFor[ConsoleServiceClient](),
		ClientCtor:        NewConsoleServiceClient,

		ERServerType:        reflect.TypeFor[ConsoleServiceServerER](),
		WrapperERServerCtor: _NewWrapperConsoleServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultConsoleServiceServerER](),
		ERClientType:        reflect.TypeFor[ConsoleServiceClientER](),
		ERClientCtor:        NewConsoleServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_ConsoleServicePingSpec,
		},
	}
	_ConsoleServicePingSpec = &rpcspec.MethodSpec{
		Name:                        "Ping",
		SkelName:                    "ping",
		ArgumentsType:               nil,
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ConsoleServiceClient.Ping,
			ConsoleServiceClientER.Ping,
			ConsoleServiceServer.Ping,
			ConsoleServiceServerER.Ping,
		},
	}
)

// ConsoleService / Server

type ConsoleServiceServer interface {
	// Ping Application health check。
	Ping()

	mustBeConsoleServiceServer()
}

// ConsoleService / Server / DefaultServer

type DefaultConsoleServiceServer struct{}

func (*DefaultConsoleServiceServer) Ping() {
	ex.PanicNew(ex.InvalidRequest, "method ping is not implemented")
}

func (*DefaultConsoleServiceServer) mustBeConsoleServiceServer() {}

// ConsoleService / ERServer

type ConsoleServiceServerER interface {
	Ping() ex.Error

	mustBeConsoleServiceServerER()
}

// ConsoleService / ERServer / WrapperERServer

type _WrapperConsoleServiceServerER struct {
	DefaultConsoleServiceServer
	serverImpl ConsoleServiceServer
}

func _NewWrapperConsoleServiceServerER(serverImpl ConsoleServiceServer) ConsoleServiceServerER {
	return &_WrapperConsoleServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperConsoleServiceServerER) server() ConsoleServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultConsoleServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperConsoleServiceServerER) Ping() (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().Ping()
	return
}

func (*_WrapperConsoleServiceServerER) mustBeConsoleServiceServerER() {}

// ConsoleService / ERServer / DefaultERServer

type DefaultConsoleServiceServerER struct {
	_WrapperConsoleServiceServerER
}

// ConsoleService / Client

type ConsoleServiceClient interface {
	// Ping Application health check。
	Ping(_ivOpts ...rpcclient.InvokeOption)
}

type _ConsoleServiceClient struct {
	clientER ConsoleServiceClientER
}

func NewConsoleServiceClient(clientER ConsoleServiceClientER) ConsoleServiceClient {
	return &_ConsoleServiceClient{clientER: clientER}
}

func (client *_ConsoleServiceClient) Ping(_ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.Ping(_ivOpts...)
	ex.PanicIfError(err)
}

// ConsoleService / ERClient

type ConsoleServiceClientER interface {
	// Ping Application health check。
	Ping(_ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _ConsoleServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewConsoleServiceClientER(rpcClient *rpcclient.Client) ConsoleServiceClientER {
	return &_ConsoleServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_ConsoleServiceClientER) Ping(_ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_ConsoleServicePingSpec.Info(), nil, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

// EventServiceServer App's event processing service, called by Link

// EventService / Spec

var (
	_EventServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "EventService",
		SkelName:          "vine.app.EventService",
		Hash:              "e2f6b7ae",
		ServerType:        reflect.TypeFor[EventServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultEventServiceServer](),
		ClientType:        reflect.TypeFor[EventServiceClient](),
		ClientCtor:        NewEventServiceClient,

		ERServerType:        reflect.TypeFor[EventServiceServerER](),
		WrapperERServerCtor: _NewWrapperEventServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultEventServiceServerER](),
		ERClientType:        reflect.TypeFor[EventServiceClientER](),
		ERClientCtor:        NewEventServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_EventServiceOnEventSpec,
		},
	}
	_EventServiceOnEventSpec = &rpcspec.MethodSpec{
		Name:                        "OnEvent",
		SkelName:                    "onEvent",
		ArgumentsType:               reflect.TypeFor[_EventServiceOnEventArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			EventServiceClient.OnEvent,
			EventServiceClientER.OnEvent,
			EventServiceServer.OnEvent,
			EventServiceServerER.OnEvent,
		},
	}
)

// EventService / Arguments

type _EventServiceOnEventArguments struct {
	On EventOn `json:"on" arg:"0"`
}

// EventService / Server

type EventServiceServer interface {
	// OnEvent Trigger event processing。
	//   @param on - Event handling information
	OnEvent(on EventOn)

	mustBeEventServiceServer()
}

// EventService / Server / DefaultServer

type DefaultEventServiceServer struct{}

func (*DefaultEventServiceServer) OnEvent(EventOn) {
	ex.PanicNew(ex.InvalidRequest, "method onEvent is not implemented")
}

func (*DefaultEventServiceServer) mustBeEventServiceServer() {}

// EventService / ERServer

type EventServiceServerER interface {
	OnEvent(on EventOn) ex.Error

	mustBeEventServiceServerER()
}

// EventService / ERServer / WrapperERServer

type _WrapperEventServiceServerER struct {
	DefaultEventServiceServer
	serverImpl EventServiceServer
}

func _NewWrapperEventServiceServerER(serverImpl EventServiceServer) EventServiceServerER {
	return &_WrapperEventServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperEventServiceServerER) server() EventServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultEventServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperEventServiceServerER) OnEvent(on EventOn) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().OnEvent(on)
	return
}

func (*_WrapperEventServiceServerER) mustBeEventServiceServerER() {}

// EventService / ERServer / DefaultERServer

type DefaultEventServiceServerER struct {
	_WrapperEventServiceServerER
}

// EventService / Client

type EventServiceClient interface {
	// OnEvent Trigger event processing。
	//   @param on - Event handling information
	OnEvent(on EventOn, _ivOpts ...rpcclient.InvokeOption)
}

type _EventServiceClient struct {
	clientER EventServiceClientER
}

func NewEventServiceClient(clientER EventServiceClientER) EventServiceClient {
	return &_EventServiceClient{clientER: clientER}
}

func (client *_EventServiceClient) OnEvent(on EventOn, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.OnEvent(on, _ivOpts...)
	ex.PanicIfError(err)
}

// EventService / ERClient

type EventServiceClientER interface {
	// OnEvent Trigger event processing。
	//   @param on - Event handling information
	OnEvent(on EventOn, _ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _EventServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewEventServiceClientER(rpcClient *rpcclient.Client) EventServiceClientER {
	return &_EventServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_EventServiceClientER) OnEvent(on EventOn, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_EventServiceOnEventSpec.Info(), &_EventServiceOnEventArguments{
		On: on,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

// TaskServiceServer App's task execution service, called by Link

// TaskService / Spec

var (
	_TaskServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "TaskService",
		SkelName:          "vine.app.TaskService",
		Hash:              "56e140ce",
		ServerType:        reflect.TypeFor[TaskServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultTaskServiceServer](),
		ClientType:        reflect.TypeFor[TaskServiceClient](),
		ClientCtor:        NewTaskServiceClient,

		ERServerType:        reflect.TypeFor[TaskServiceServerER](),
		WrapperERServerCtor: _NewWrapperTaskServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultTaskServiceServerER](),
		ERClientType:        reflect.TypeFor[TaskServiceClientER](),
		ERClientCtor:        NewTaskServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_TaskServiceRunTaskSpec,
		},
	}
	_TaskServiceRunTaskSpec = &rpcspec.MethodSpec{
		Name:                        "RunTask",
		SkelName:                    "runTask",
		ArgumentsType:               reflect.TypeFor[_TaskServiceRunTaskArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			TaskServiceClient.RunTask,
			TaskServiceClientER.RunTask,
			TaskServiceServer.RunTask,
			TaskServiceServerER.RunTask,
		},
	}
)

// TaskService / Arguments

type _TaskServiceRunTaskArguments struct {
	Run TaskRun `json:"run" arg:"0"`
}

// TaskService / Server

type TaskServiceServer interface {
	// RunTask Trigger task execution。
	//   @param run - Task execution information
	RunTask(run TaskRun)

	mustBeTaskServiceServer()
}

// TaskService / Server / DefaultServer

type DefaultTaskServiceServer struct{}

func (*DefaultTaskServiceServer) RunTask(TaskRun) {
	ex.PanicNew(ex.InvalidRequest, "method runTask is not implemented")
}

func (*DefaultTaskServiceServer) mustBeTaskServiceServer() {}

// TaskService / ERServer

type TaskServiceServerER interface {
	RunTask(run TaskRun) ex.Error

	mustBeTaskServiceServerER()
}

// TaskService / ERServer / WrapperERServer

type _WrapperTaskServiceServerER struct {
	DefaultTaskServiceServer
	serverImpl TaskServiceServer
}

func _NewWrapperTaskServiceServerER(serverImpl TaskServiceServer) TaskServiceServerER {
	return &_WrapperTaskServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperTaskServiceServerER) server() TaskServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultTaskServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperTaskServiceServerER) RunTask(run TaskRun) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().RunTask(run)
	return
}

func (*_WrapperTaskServiceServerER) mustBeTaskServiceServerER() {}

// TaskService / ERServer / DefaultERServer

type DefaultTaskServiceServerER struct {
	_WrapperTaskServiceServerER
}

// TaskService / Client

type TaskServiceClient interface {
	// RunTask Trigger task execution。
	//   @param run - Task execution information
	RunTask(run TaskRun, _ivOpts ...rpcclient.InvokeOption)
}

type _TaskServiceClient struct {
	clientER TaskServiceClientER
}

func NewTaskServiceClient(clientER TaskServiceClientER) TaskServiceClient {
	return &_TaskServiceClient{clientER: clientER}
}

func (client *_TaskServiceClient) RunTask(run TaskRun, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.RunTask(run, _ivOpts...)
	ex.PanicIfError(err)
}

// TaskService / ERClient

type TaskServiceClientER interface {
	// RunTask Trigger task execution。
	//   @param run - Task execution information
	RunTask(run TaskRun, _ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _TaskServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewTaskServiceClientER(rpcClient *rpcclient.Client) TaskServiceClientER {
	return &_TaskServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_TaskServiceClientER) RunTask(run TaskRun, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_TaskServiceRunTaskSpec.Info(), &_TaskServiceRunTaskArguments{
		Run: run,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}
