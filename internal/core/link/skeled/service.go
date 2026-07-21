package skeled

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/ex"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
)

func init() {
	rpcspec.Register(_BootServiceSpec)
	rpcspec.Register(_ConfigServiceSpec)
	rpcspec.Register(_EventServiceSpec)
	rpcspec.Register(_RegistryServiceSpec)
	rpcspec.Register(_TaskServiceSpec)
}

// BootServiceServer Link's startup information service, called by the App

// BootService / Spec

var (
	_BootServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "BootService",
		SkelName:          "vine.link.BootService",
		Hash:              "e8b9fae5",
		ServerType:        reflect.TypeFor[BootServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultBootServiceServer](),
		ClientType:        reflect.TypeFor[BootServiceClient](),
		ClientCtor:        NewBootServiceClient,

		ERServerType:        reflect.TypeFor[BootServiceServerER](),
		WrapperERServerCtor: _NewWrapperBootServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultBootServiceServerER](),
		ERClientType:        reflect.TypeFor[BootServiceClientER](),
		ERClientCtor:        NewBootServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_BootServiceGetInfoSpec,
		},
	}
	_BootServiceGetInfoSpec = &rpcspec.MethodSpec{
		Name:                        "GetInfo",
		SkelName:                    "getInfo",
		ArgumentsType:               nil,
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[BootInfo](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			BootServiceClient.GetInfo,
			BootServiceClientER.GetInfo,
			BootServiceServer.GetInfo,
			BootServiceServerER.GetInfo,
		},
	}
)

// BootService / Server

type BootServiceServer interface {
	// GetInfo Get key startup information。
	GetInfo() BootInfo

	mustBeBootServiceServer()
}

// BootService / Server / DefaultServer

type DefaultBootServiceServer struct{}

func (*DefaultBootServiceServer) GetInfo() BootInfo {
	ex.PanicNew(ex.InvalidRequest, "method getInfo is not implemented")
	return BootInfo{}
}

func (*DefaultBootServiceServer) mustBeBootServiceServer() {}

// BootService / ERServer

type BootServiceServerER interface {
	GetInfo() (BootInfo, ex.Error)

	mustBeBootServiceServerER()
}

// BootService / ERServer / WrapperERServer

type _WrapperBootServiceServerER struct {
	DefaultBootServiceServer
	serverImpl BootServiceServer
}

func _NewWrapperBootServiceServerER(serverImpl BootServiceServer) BootServiceServerER {
	return &_WrapperBootServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperBootServiceServerER) server() BootServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultBootServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperBootServiceServerER) GetInfo() (ret BootInfo, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().GetInfo()
	return
}

func (*_WrapperBootServiceServerER) mustBeBootServiceServerER() {}

// BootService / ERServer / DefaultERServer

type DefaultBootServiceServerER struct {
	_WrapperBootServiceServerER
}

// BootService / Client

type BootServiceClient interface {
	// GetInfo Get key startup information。
	GetInfo(_ivOpts ...rpcclient.InvokeOption) BootInfo
}

type _BootServiceClient struct {
	clientER BootServiceClientER
}

func NewBootServiceClient(clientER BootServiceClientER) BootServiceClient {
	return &_BootServiceClient{clientER: clientER}
}

func (client *_BootServiceClient) GetInfo(_ivOpts ...rpcclient.InvokeOption) BootInfo {
	ret, err := client.clientER.GetInfo(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// BootService / ERClient

type BootServiceClientER interface {
	// GetInfo Get key startup information。
	GetInfo(_ivOpts ...rpcclient.InvokeOption) (BootInfo, ex.Error)
}

type _BootServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewBootServiceClientER(rpcClient *rpcclient.Client) BootServiceClientER {
	return &_BootServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_BootServiceClientER) GetInfo(_ivOpts ...rpcclient.InvokeOption) (BootInfo, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_BootServiceGetInfoSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.(BootInfo)
	err, _ := errI.(ex.Error)
	return ret, err
}

// ConfigServiceServer Link's application configuration service, called by the App

// ConfigService / Spec

var (
	_ConfigServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "ConfigService",
		SkelName:          "vine.link.ConfigService",
		Hash:              "420d2cb7",
		ServerType:        reflect.TypeFor[ConfigServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultConfigServiceServer](),
		ClientType:        reflect.TypeFor[ConfigServiceClient](),
		ClientCtor:        NewConfigServiceClient,

		ERServerType:        reflect.TypeFor[ConfigServiceServerER](),
		WrapperERServerCtor: _NewWrapperConfigServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultConfigServiceServerER](),
		ERClientType:        reflect.TypeFor[ConfigServiceClientER](),
		ERClientCtor:        NewConfigServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_ConfigServiceGetEternalSpec,
			_ConfigServiceGetInstantSpec,
		},
	}
	_ConfigServiceGetEternalSpec = &rpcspec.MethodSpec{
		Name:                        "GetEternal",
		SkelName:                    "getEternal",
		ArgumentsType:               reflect.TypeFor[_ConfigServiceGetEternalArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[string](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ConfigServiceClient.GetEternal,
			ConfigServiceClientER.GetEternal,
			ConfigServiceServer.GetEternal,
			ConfigServiceServerER.GetEternal,
		},
	}
	_ConfigServiceGetInstantSpec = &rpcspec.MethodSpec{
		Name:                        "GetInstant",
		SkelName:                    "getInstant",
		ArgumentsType:               reflect.TypeFor[_ConfigServiceGetInstantArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[string](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ConfigServiceClient.GetInstant,
			ConfigServiceClientER.GetInstant,
			ConfigServiceServer.GetInstant,
			ConfigServiceServerER.GetInstant,
		},
	}
)

// ConfigService / Arguments

type _ConfigServiceGetEternalArguments struct {
	Key string `json:"key" arg:"0"`
}

type _ConfigServiceGetInstantArguments struct {
	Key string `json:"key" arg:"0"`
}

// ConfigService / Server

type ConfigServiceServer interface {
	// GetEternal Read Eternal configuration。
	//   @param key - Configuration key
	//   @returns string - Configuration JSON
	GetEternal(key string) string
	// GetInstant Read Instant configuration。
	//   @param key - Configuration key
	//   @returns string - Configuration JSON
	GetInstant(key string) string

	mustBeConfigServiceServer()
}

// ConfigService / Server / DefaultServer

type DefaultConfigServiceServer struct{}

func (*DefaultConfigServiceServer) GetEternal(string) string {
	ex.PanicNew(ex.InvalidRequest, "method getEternal is not implemented")
	return ""
}

func (*DefaultConfigServiceServer) GetInstant(string) string {
	ex.PanicNew(ex.InvalidRequest, "method getInstant is not implemented")
	return ""
}

func (*DefaultConfigServiceServer) mustBeConfigServiceServer() {}

// ConfigService / ERServer

type ConfigServiceServerER interface {
	GetEternal(key string) (string, ex.Error)
	GetInstant(key string) (string, ex.Error)

	mustBeConfigServiceServerER()
}

// ConfigService / ERServer / WrapperERServer

type _WrapperConfigServiceServerER struct {
	DefaultConfigServiceServer
	serverImpl ConfigServiceServer
}

func _NewWrapperConfigServiceServerER(serverImpl ConfigServiceServer) ConfigServiceServerER {
	return &_WrapperConfigServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperConfigServiceServerER) server() ConfigServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultConfigServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperConfigServiceServerER) GetEternal(key string) (ret string, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().GetEternal(key)
	return
}

func (service *_WrapperConfigServiceServerER) GetInstant(key string) (ret string, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().GetInstant(key)
	return
}

func (*_WrapperConfigServiceServerER) mustBeConfigServiceServerER() {}

// ConfigService / ERServer / DefaultERServer

type DefaultConfigServiceServerER struct {
	_WrapperConfigServiceServerER
}

// ConfigService / Client

type ConfigServiceClient interface {
	// GetEternal Read Eternal configuration。
	//   @param key - Configuration key
	//   @returns string - Configuration JSON
	GetEternal(key string, _ivOpts ...rpcclient.InvokeOption) string
	// GetInstant Read Instant configuration。
	//   @param key - Configuration key
	//   @returns string - Configuration JSON
	GetInstant(key string, _ivOpts ...rpcclient.InvokeOption) string
}

type _ConfigServiceClient struct {
	clientER ConfigServiceClientER
}

func NewConfigServiceClient(clientER ConfigServiceClientER) ConfigServiceClient {
	return &_ConfigServiceClient{clientER: clientER}
}

func (client *_ConfigServiceClient) GetEternal(key string, _ivOpts ...rpcclient.InvokeOption) string {
	ret, err := client.clientER.GetEternal(key, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_ConfigServiceClient) GetInstant(key string, _ivOpts ...rpcclient.InvokeOption) string {
	ret, err := client.clientER.GetInstant(key, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// ConfigService / ERClient

type ConfigServiceClientER interface {
	// GetEternal Read Eternal configuration。
	//   @param key - Configuration key
	//   @returns string - Configuration JSON
	GetEternal(key string, _ivOpts ...rpcclient.InvokeOption) (string, ex.Error)
	// GetInstant Read Instant configuration。
	//   @param key - Configuration key
	//   @returns string - Configuration JSON
	GetInstant(key string, _ivOpts ...rpcclient.InvokeOption) (string, ex.Error)
}

type _ConfigServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewConfigServiceClientER(rpcClient *rpcclient.Client) ConfigServiceClientER {
	return &_ConfigServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_ConfigServiceClientER) GetEternal(key string, _ivOpts ...rpcclient.InvokeOption) (string, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_ConfigServiceGetEternalSpec.Info(), &_ConfigServiceGetEternalArguments{
		Key: key,
	}, _ivOpts...)
	ret, _ := retI.(string)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_ConfigServiceClientER) GetInstant(key string, _ivOpts ...rpcclient.InvokeOption) (string, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_ConfigServiceGetInstantSpec.Info(), &_ConfigServiceGetInstantArguments{
		Key: key,
	}, _ivOpts...)
	ret, _ := retI.(string)
	err, _ := errI.(ex.Error)
	return ret, err
}

// EventServiceServer Link's event service, called by App

// EventService / Spec

var (
	_EventServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "EventService",
		SkelName:          "vine.link.EventService",
		Hash:              "f1d171cc",
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
			_EventServiceEmitEventSpec,
		},
	}
	_EventServiceEmitEventSpec = &rpcspec.MethodSpec{
		Name:                        "EmitEvent",
		SkelName:                    "emitEvent",
		ArgumentsType:               reflect.TypeFor[_EventServiceEmitEventArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			EventServiceClient.EmitEvent,
			EventServiceClientER.EmitEvent,
			EventServiceServer.EmitEvent,
			EventServiceServerER.EmitEvent,
		},
	}
)

// EventService / Arguments

type _EventServiceEmitEventArguments struct {
	Emission EventEmission `json:"emission" arg:"0"`
}

// EventService / Server

type EventServiceServer interface {
	// EmitEvent Send event。
	//   @param emission - Event sending information
	EmitEvent(emission EventEmission)

	mustBeEventServiceServer()
}

// EventService / Server / DefaultServer

type DefaultEventServiceServer struct{}

func (*DefaultEventServiceServer) EmitEvent(EventEmission) {
	ex.PanicNew(ex.InvalidRequest, "method emitEvent is not implemented")
}

func (*DefaultEventServiceServer) mustBeEventServiceServer() {}

// EventService / ERServer

type EventServiceServerER interface {
	EmitEvent(emission EventEmission) ex.Error

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

func (service *_WrapperEventServiceServerER) EmitEvent(emission EventEmission) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().EmitEvent(emission)
	return
}

func (*_WrapperEventServiceServerER) mustBeEventServiceServerER() {}

// EventService / ERServer / DefaultERServer

type DefaultEventServiceServerER struct {
	_WrapperEventServiceServerER
}

// EventService / Client

type EventServiceClient interface {
	// EmitEvent Send event。
	//   @param emission - Event sending information
	EmitEvent(emission EventEmission, _ivOpts ...rpcclient.InvokeOption)
}

type _EventServiceClient struct {
	clientER EventServiceClientER
}

func NewEventServiceClient(clientER EventServiceClientER) EventServiceClient {
	return &_EventServiceClient{clientER: clientER}
}

func (client *_EventServiceClient) EmitEvent(emission EventEmission, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.EmitEvent(emission, _ivOpts...)
	ex.PanicIfError(err)
}

// EventService / ERClient

type EventServiceClientER interface {
	// EmitEvent Send event。
	//   @param emission - Event sending information
	EmitEvent(emission EventEmission, _ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _EventServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewEventServiceClientER(rpcClient *rpcclient.Client) EventServiceClientER {
	return &_EventServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_EventServiceClientER) EmitEvent(emission EventEmission, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_EventServiceEmitEventSpec.Info(), &_EventServiceEmitEventArguments{
		Emission: emission,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

// RegistryServiceServer Link's application registration service, called by the App

// RegistryService / Spec

var (
	_RegistryServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "RegistryService",
		SkelName:          "vine.link.RegistryService",
		Hash:              "6f5f2981",
		ServerType:        reflect.TypeFor[RegistryServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultRegistryServiceServer](),
		ClientType:        reflect.TypeFor[RegistryServiceClient](),
		ClientCtor:        NewRegistryServiceClient,

		ERServerType:        reflect.TypeFor[RegistryServiceServerER](),
		WrapperERServerCtor: _NewWrapperRegistryServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultRegistryServiceServerER](),
		ERClientType:        reflect.TypeFor[RegistryServiceClientER](),
		ERClientCtor:        NewRegistryServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_RegistryServiceRegisterSpec,
			_RegistryServiceUnregisterSpec,
		},
	}
	_RegistryServiceRegisterSpec = &rpcspec.MethodSpec{
		Name:          "Register",
		SkelName:      "register",
		ArgumentsType: reflect.TypeFor[_RegistryServiceRegisterArguments](),
		ValidateArguments: func(value any) error {
			args := value.(*_RegistryServiceRegisterArguments)
			if err := (&args.Registration).Validate(rpcspec.JoinPath("arguments", "Registration")); err != nil {
				return err
			}
			return nil
		},
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			RegistryServiceClient.Register,
			RegistryServiceClientER.Register,
			RegistryServiceServer.Register,
			RegistryServiceServerER.Register,
		},
	}
	_RegistryServiceUnregisterSpec = &rpcspec.MethodSpec{
		Name:                        "Unregister",
		SkelName:                    "unregister",
		ArgumentsType:               nil,
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			RegistryServiceClient.Unregister,
			RegistryServiceClientER.Unregister,
			RegistryServiceServer.Unregister,
			RegistryServiceServerER.Unregister,
		},
	}
)

// RegistryService / Arguments

type _RegistryServiceRegisterArguments struct {
	Registration AppRegistration `json:"registration" arg:"0"`
}

// RegistryService / Server

type RegistryServiceServer interface {
	// Register Register the currently running application。
	//   @param registration - Application instance registration information
	Register(registration AppRegistration)
	// Unregister Log out of the currently running application and prepare to exit gracefully. This call may block until the current instance's in-progress work on the Link side is drained, or until the wait times out。
	Unregister()

	mustBeRegistryServiceServer()
}

// RegistryService / Server / DefaultServer

type DefaultRegistryServiceServer struct{}

func (*DefaultRegistryServiceServer) Register(AppRegistration) {
	ex.PanicNew(ex.InvalidRequest, "method register is not implemented")
}

func (*DefaultRegistryServiceServer) Unregister() {
	ex.PanicNew(ex.InvalidRequest, "method unregister is not implemented")
}

func (*DefaultRegistryServiceServer) mustBeRegistryServiceServer() {}

// RegistryService / ERServer

type RegistryServiceServerER interface {
	Register(registration AppRegistration) ex.Error
	Unregister() ex.Error

	mustBeRegistryServiceServerER()
}

// RegistryService / ERServer / WrapperERServer

type _WrapperRegistryServiceServerER struct {
	DefaultRegistryServiceServer
	serverImpl RegistryServiceServer
}

func _NewWrapperRegistryServiceServerER(serverImpl RegistryServiceServer) RegistryServiceServerER {
	return &_WrapperRegistryServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperRegistryServiceServerER) server() RegistryServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultRegistryServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperRegistryServiceServerER) Register(registration AppRegistration) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().Register(registration)
	return
}

func (service *_WrapperRegistryServiceServerER) Unregister() (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().Unregister()
	return
}

func (*_WrapperRegistryServiceServerER) mustBeRegistryServiceServerER() {}

// RegistryService / ERServer / DefaultERServer

type DefaultRegistryServiceServerER struct {
	_WrapperRegistryServiceServerER
}

// RegistryService / Client

type RegistryServiceClient interface {
	// Register Register the currently running application。
	//   @param registration - Application instance registration information
	Register(registration AppRegistration, _ivOpts ...rpcclient.InvokeOption)
	// Unregister Log out of the currently running application and prepare to exit gracefully. This call may block until the current instance's in-progress work on the Link side is drained, or until the wait times out。
	Unregister(_ivOpts ...rpcclient.InvokeOption)
}

type _RegistryServiceClient struct {
	clientER RegistryServiceClientER
}

func NewRegistryServiceClient(clientER RegistryServiceClientER) RegistryServiceClient {
	return &_RegistryServiceClient{clientER: clientER}
}

func (client *_RegistryServiceClient) Register(registration AppRegistration, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.Register(registration, _ivOpts...)
	ex.PanicIfError(err)
}

func (client *_RegistryServiceClient) Unregister(_ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.Unregister(_ivOpts...)
	ex.PanicIfError(err)
}

// RegistryService / ERClient

type RegistryServiceClientER interface {
	// Register Register the currently running application。
	//   @param registration - Application instance registration information
	Register(registration AppRegistration, _ivOpts ...rpcclient.InvokeOption) ex.Error
	// Unregister Log out of the currently running application and prepare to exit gracefully. This call may block until the current instance's in-progress work on the Link side is drained, or until the wait times out。
	Unregister(_ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _RegistryServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewRegistryServiceClientER(rpcClient *rpcclient.Client) RegistryServiceClientER {
	return &_RegistryServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_RegistryServiceClientER) Register(registration AppRegistration, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_RegistryServiceRegisterSpec.Info(), &_RegistryServiceRegisterArguments{
		Registration: registration,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

func (client *_RegistryServiceClientER) Unregister(_ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_RegistryServiceUnregisterSpec.Info(), nil, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

// TaskServiceServer Link's task service, called by the App

// TaskService / Spec

var (
	_TaskServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "TaskService",
		SkelName:          "vine.link.TaskService",
		Hash:              "26a1aef1",
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
			_TaskServiceLaunchTaskSpec,
		},
	}
	_TaskServiceLaunchTaskSpec = &rpcspec.MethodSpec{
		Name:                        "LaunchTask",
		SkelName:                    "launchTask",
		ArgumentsType:               reflect.TypeFor[_TaskServiceLaunchTaskArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			TaskServiceClient.LaunchTask,
			TaskServiceClientER.LaunchTask,
			TaskServiceServer.LaunchTask,
			TaskServiceServerER.LaunchTask,
		},
	}
)

// TaskService / Arguments

type _TaskServiceLaunchTaskArguments struct {
	Launch TaskLaunch `json:"launch" arg:"0"`
}

// TaskService / Server

type TaskServiceServer interface {
	// LaunchTask Start a task。
	//   @param launch - Task trigger information
	LaunchTask(launch TaskLaunch)

	mustBeTaskServiceServer()
}

// TaskService / Server / DefaultServer

type DefaultTaskServiceServer struct{}

func (*DefaultTaskServiceServer) LaunchTask(TaskLaunch) {
	ex.PanicNew(ex.InvalidRequest, "method launchTask is not implemented")
}

func (*DefaultTaskServiceServer) mustBeTaskServiceServer() {}

// TaskService / ERServer

type TaskServiceServerER interface {
	LaunchTask(launch TaskLaunch) ex.Error

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

func (service *_WrapperTaskServiceServerER) LaunchTask(launch TaskLaunch) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().LaunchTask(launch)
	return
}

func (*_WrapperTaskServiceServerER) mustBeTaskServiceServerER() {}

// TaskService / ERServer / DefaultERServer

type DefaultTaskServiceServerER struct {
	_WrapperTaskServiceServerER
}

// TaskService / Client

type TaskServiceClient interface {
	// LaunchTask Start a task。
	//   @param launch - Task trigger information
	LaunchTask(launch TaskLaunch, _ivOpts ...rpcclient.InvokeOption)
}

type _TaskServiceClient struct {
	clientER TaskServiceClientER
}

func NewTaskServiceClient(clientER TaskServiceClientER) TaskServiceClient {
	return &_TaskServiceClient{clientER: clientER}
}

func (client *_TaskServiceClient) LaunchTask(launch TaskLaunch, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.LaunchTask(launch, _ivOpts...)
	ex.PanicIfError(err)
}

// TaskService / ERClient

type TaskServiceClientER interface {
	// LaunchTask Start a task。
	//   @param launch - Task trigger information
	LaunchTask(launch TaskLaunch, _ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _TaskServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewTaskServiceClientER(rpcClient *rpcclient.Client) TaskServiceClientER {
	return &_TaskServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_TaskServiceClientER) LaunchTask(launch TaskLaunch, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_TaskServiceLaunchTaskSpec.Info(), &_TaskServiceLaunchTaskArguments{
		Launch: launch,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}
