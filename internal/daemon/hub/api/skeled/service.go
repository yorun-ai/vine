package skeled

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/ex"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/skel"
)

func init() {
	rpcspec.Register(_AppConfigServiceSpec)
	rpcspec.Register(_AppStatusServiceSpec)
	rpcspec.Register(_EventDebugServiceSpec)
	rpcspec.Register(_InfoServiceSpec)
	rpcspec.Register(_MaintenanceServiceSpec)
	rpcspec.Register(_PortalCertServiceSpec)
	rpcspec.Register(_PortalEntryServiceSpec)
	rpcspec.Register(_PortalRuleServiceSpec)
	rpcspec.Register(_PortalSiteServiceSpec)
	rpcspec.Register(_RegistryServiceSpec)
	rpcspec.Register(_ServiceDebugServiceSpec)
	rpcspec.Register(_SkeletonServiceSpec)
	rpcspec.Register(_TaskDebugServiceSpec)
}

// AppConfigServiceServer Hub's application configuration service, called by Client

// AppConfigService / Spec

var (
	_AppConfigServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "AppConfigService",
		SkelName:          "vine.hub.AppConfigService",
		Hash:              "358424b9",
		ServerType:        reflect.TypeFor[AppConfigServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultAppConfigServiceServer](),
		ClientType:        reflect.TypeFor[AppConfigServiceClient](),
		ClientCtor:        NewAppConfigServiceClient,

		ERServerType:        reflect.TypeFor[AppConfigServiceServerER](),
		WrapperERServerCtor: _NewWrapperAppConfigServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultAppConfigServiceServerER](),
		ERClientType:        reflect.TypeFor[AppConfigServiceClientER](),
		ERClientCtor:        NewAppConfigServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_AppConfigServiceListSpec,
			_AppConfigServiceGetSpec,
			_AppConfigServiceUpdateSpec,
			_AppConfigServiceCreateSpec,
			_AppConfigServiceRemoveSpec,
		},
	}
	_AppConfigServiceListSpec = &rpcspec.MethodSpec{
		Name:              "List",
		SkelName:          "list",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]AppConfigItem](),
		ValidateResult: func(value any) error {
			ret := value.([]AppConfigItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			AppConfigServiceClient.List,
			AppConfigServiceClientER.List,
			AppConfigServiceServer.List,
			AppConfigServiceServerER.List,
		},
	}
	_AppConfigServiceGetSpec = &rpcspec.MethodSpec{
		Name:              "Get",
		SkelName:          "get",
		ArgumentsType:     reflect.TypeFor[_AppConfigServiceGetArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[AppConfigItem](),
		ValidateResult: func(value any) error {
			ret := value.(AppConfigItem)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			AppConfigServiceClient.Get,
			AppConfigServiceClientER.Get,
			AppConfigServiceServer.Get,
			AppConfigServiceServerER.Get,
		},
	}
	_AppConfigServiceUpdateSpec = &rpcspec.MethodSpec{
		Name:              "Update",
		SkelName:          "update",
		ArgumentsType:     reflect.TypeFor[_AppConfigServiceUpdateArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[AppConfigItem](),
		ValidateResult: func(value any) error {
			ret := value.(AppConfigItem)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			AppConfigServiceClient.Update,
			AppConfigServiceClientER.Update,
			AppConfigServiceServer.Update,
			AppConfigServiceServerER.Update,
		},
	}
	_AppConfigServiceCreateSpec = &rpcspec.MethodSpec{
		Name:              "Create",
		SkelName:          "create",
		ArgumentsType:     reflect.TypeFor[_AppConfigServiceCreateArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[AppConfigItem](),
		ValidateResult: func(value any) error {
			ret := value.(AppConfigItem)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			AppConfigServiceClient.Create,
			AppConfigServiceClientER.Create,
			AppConfigServiceServer.Create,
			AppConfigServiceServerER.Create,
		},
	}
	_AppConfigServiceRemoveSpec = &rpcspec.MethodSpec{
		Name:                        "Remove",
		SkelName:                    "remove",
		ArgumentsType:               reflect.TypeFor[_AppConfigServiceRemoveArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[bool](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			AppConfigServiceClient.Remove,
			AppConfigServiceClientER.Remove,
			AppConfigServiceServer.Remove,
			AppConfigServiceServerER.Remove,
		},
	}
)

// AppConfigService / Arguments

type _AppConfigServiceGetArguments struct {
	Id int `json:"id" arg:"0"`
}

type _AppConfigServiceUpdateArguments struct {
	Id     int             `json:"id" arg:"0"`
	Update AppConfigUpdate `json:"update" arg:"1"`
}

type _AppConfigServiceCreateArguments struct {
	Creation AppConfigCreation `json:"creation" arg:"0"`
}

type _AppConfigServiceRemoveArguments struct {
	Id int `json:"id" arg:"0"`
}

// AppConfigService / Server

type AppConfigServiceServer interface {
	// List List configuration items。
	//   @returns []AppConfigItem - Configuration item list
	List() []AppConfigItem
	// Get Read configuration。
	//   @param id - Configuration ID
	//   @returns AppConfigItem - Configuration items
	Get(id int) AppConfigItem
	// Update Modify configuration。
	//   @param id - Configuration ID
	//   @param update - Configuration update parameters
	//   @returns AppConfigItem - Configuration items
	Update(id int, update AppConfigUpdate) AppConfigItem
	// Create Create configuration。
	//   @param creation - Configuration creation parameters
	//   @returns AppConfigItem - Configuration items
	Create(creation AppConfigCreation) AppConfigItem
	// Remove Delete unused configuration。
	//   @param id - Configuration ID
	//   @returns bool - Whether deletion succeeded
	Remove(id int) bool

	mustBeAppConfigServiceServer()
}

// AppConfigService / Server / DefaultServer

type DefaultAppConfigServiceServer struct{}

func (*DefaultAppConfigServiceServer) List() []AppConfigItem {
	ex.PanicNew(ex.InvalidRequest, "method list is not implemented")
	return []AppConfigItem{}
}

func (*DefaultAppConfigServiceServer) Get(int) AppConfigItem {
	ex.PanicNew(ex.InvalidRequest, "method get is not implemented")
	return AppConfigItem{}
}

func (*DefaultAppConfigServiceServer) Update(int, AppConfigUpdate) AppConfigItem {
	ex.PanicNew(ex.InvalidRequest, "method update is not implemented")
	return AppConfigItem{}
}

func (*DefaultAppConfigServiceServer) Create(AppConfigCreation) AppConfigItem {
	ex.PanicNew(ex.InvalidRequest, "method create is not implemented")
	return AppConfigItem{}
}

func (*DefaultAppConfigServiceServer) Remove(int) bool {
	ex.PanicNew(ex.InvalidRequest, "method remove is not implemented")
	return false
}

func (*DefaultAppConfigServiceServer) mustBeAppConfigServiceServer() {}

// AppConfigService / ERServer

type AppConfigServiceServerER interface {
	List() ([]AppConfigItem, ex.Error)
	Get(id int) (AppConfigItem, ex.Error)
	Update(id int, update AppConfigUpdate) (AppConfigItem, ex.Error)
	Create(creation AppConfigCreation) (AppConfigItem, ex.Error)
	Remove(id int) (bool, ex.Error)

	mustBeAppConfigServiceServerER()
}

// AppConfigService / ERServer / WrapperERServer

type _WrapperAppConfigServiceServerER struct {
	DefaultAppConfigServiceServer
	serverImpl AppConfigServiceServer
}

func _NewWrapperAppConfigServiceServerER(serverImpl AppConfigServiceServer) AppConfigServiceServerER {
	return &_WrapperAppConfigServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperAppConfigServiceServerER) server() AppConfigServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultAppConfigServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperAppConfigServiceServerER) List() (ret []AppConfigItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().List()
	return
}

func (service *_WrapperAppConfigServiceServerER) Get(id int) (ret AppConfigItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Get(id)
	return
}

func (service *_WrapperAppConfigServiceServerER) Update(id int, update AppConfigUpdate) (ret AppConfigItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Update(id, update)
	return
}

func (service *_WrapperAppConfigServiceServerER) Create(creation AppConfigCreation) (ret AppConfigItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Create(creation)
	return
}

func (service *_WrapperAppConfigServiceServerER) Remove(id int) (ret bool, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Remove(id)
	return
}

func (*_WrapperAppConfigServiceServerER) mustBeAppConfigServiceServerER() {}

// AppConfigService / ERServer / DefaultERServer

type DefaultAppConfigServiceServerER struct {
	_WrapperAppConfigServiceServerER
}

// AppConfigService / Client

type AppConfigServiceClient interface {
	// List List configuration items。
	//   @returns []AppConfigItem - Configuration item list
	List(_ivOpts ...rpcclient.InvokeOption) []AppConfigItem
	// Get Read configuration。
	//   @param id - Configuration ID
	//   @returns AppConfigItem - Configuration items
	Get(id int, _ivOpts ...rpcclient.InvokeOption) AppConfigItem
	// Update Modify configuration。
	//   @param id - Configuration ID
	//   @param update - Configuration update parameters
	//   @returns AppConfigItem - Configuration items
	Update(id int, update AppConfigUpdate, _ivOpts ...rpcclient.InvokeOption) AppConfigItem
	// Create Create configuration。
	//   @param creation - Configuration creation parameters
	//   @returns AppConfigItem - Configuration items
	Create(creation AppConfigCreation, _ivOpts ...rpcclient.InvokeOption) AppConfigItem
	// Remove Delete unused configuration。
	//   @param id - Configuration ID
	//   @returns bool - Whether deletion succeeded
	Remove(id int, _ivOpts ...rpcclient.InvokeOption) bool
}

type _AppConfigServiceClient struct {
	clientER AppConfigServiceClientER
}

func NewAppConfigServiceClient(clientER AppConfigServiceClientER) AppConfigServiceClient {
	return &_AppConfigServiceClient{clientER: clientER}
}

func (client *_AppConfigServiceClient) List(_ivOpts ...rpcclient.InvokeOption) []AppConfigItem {
	ret, err := client.clientER.List(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_AppConfigServiceClient) Get(id int, _ivOpts ...rpcclient.InvokeOption) AppConfigItem {
	ret, err := client.clientER.Get(id, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_AppConfigServiceClient) Update(id int, update AppConfigUpdate, _ivOpts ...rpcclient.InvokeOption) AppConfigItem {
	ret, err := client.clientER.Update(id, update, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_AppConfigServiceClient) Create(creation AppConfigCreation, _ivOpts ...rpcclient.InvokeOption) AppConfigItem {
	ret, err := client.clientER.Create(creation, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_AppConfigServiceClient) Remove(id int, _ivOpts ...rpcclient.InvokeOption) bool {
	ret, err := client.clientER.Remove(id, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// AppConfigService / ERClient

type AppConfigServiceClientER interface {
	// List List configuration items。
	//   @returns []AppConfigItem - Configuration item list
	List(_ivOpts ...rpcclient.InvokeOption) ([]AppConfigItem, ex.Error)
	// Get Read configuration。
	//   @param id - Configuration ID
	//   @returns AppConfigItem - Configuration items
	Get(id int, _ivOpts ...rpcclient.InvokeOption) (AppConfigItem, ex.Error)
	// Update Modify configuration。
	//   @param id - Configuration ID
	//   @param update - Configuration update parameters
	//   @returns AppConfigItem - Configuration items
	Update(id int, update AppConfigUpdate, _ivOpts ...rpcclient.InvokeOption) (AppConfigItem, ex.Error)
	// Create Create configuration。
	//   @param creation - Configuration creation parameters
	//   @returns AppConfigItem - Configuration items
	Create(creation AppConfigCreation, _ivOpts ...rpcclient.InvokeOption) (AppConfigItem, ex.Error)
	// Remove Delete unused configuration。
	//   @param id - Configuration ID
	//   @returns bool - Whether deletion succeeded
	Remove(id int, _ivOpts ...rpcclient.InvokeOption) (bool, ex.Error)
}

type _AppConfigServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewAppConfigServiceClientER(rpcClient *rpcclient.Client) AppConfigServiceClientER {
	return &_AppConfigServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_AppConfigServiceClientER) List(_ivOpts ...rpcclient.InvokeOption) ([]AppConfigItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_AppConfigServiceListSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]AppConfigItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_AppConfigServiceClientER) Get(id int, _ivOpts ...rpcclient.InvokeOption) (AppConfigItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_AppConfigServiceGetSpec.Info(), &_AppConfigServiceGetArguments{
		Id: id,
	}, _ivOpts...)
	ret, _ := retI.(AppConfigItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_AppConfigServiceClientER) Update(id int, update AppConfigUpdate, _ivOpts ...rpcclient.InvokeOption) (AppConfigItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_AppConfigServiceUpdateSpec.Info(), &_AppConfigServiceUpdateArguments{
		Id:     id,
		Update: update,
	}, _ivOpts...)
	ret, _ := retI.(AppConfigItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_AppConfigServiceClientER) Create(creation AppConfigCreation, _ivOpts ...rpcclient.InvokeOption) (AppConfigItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_AppConfigServiceCreateSpec.Info(), &_AppConfigServiceCreateArguments{
		Creation: creation,
	}, _ivOpts...)
	ret, _ := retI.(AppConfigItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_AppConfigServiceClientER) Remove(id int, _ivOpts ...rpcclient.InvokeOption) (bool, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_AppConfigServiceRemoveSpec.Info(), &_AppConfigServiceRemoveArguments{
		Id: id,
	}, _ivOpts...)
	ret, _ := retI.(bool)
	err, _ := errI.(ex.Error)
	return ret, err
}

// AppStatusServiceServer Hub Dashboard's application status service

// AppStatusService / Spec

var (
	_AppStatusServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "AppStatusService",
		SkelName:          "vine.hub.AppStatusService",
		Hash:              "2d56f3ea",
		ServerType:        reflect.TypeFor[AppStatusServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultAppStatusServiceServer](),
		ClientType:        reflect.TypeFor[AppStatusServiceClient](),
		ClientCtor:        NewAppStatusServiceClient,

		ERServerType:        reflect.TypeFor[AppStatusServiceServerER](),
		WrapperERServerCtor: _NewWrapperAppStatusServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultAppStatusServiceServerER](),
		ERClientType:        reflect.TypeFor[AppStatusServiceClientER](),
		ERClientCtor:        NewAppStatusServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_AppStatusServiceListSpec,
		},
	}
	_AppStatusServiceListSpec = &rpcspec.MethodSpec{
		Name:              "List",
		SkelName:          "list",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]AppStatusView](),
		ValidateResult: func(value any) error {
			ret := value.([]AppStatusView)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			AppStatusServiceClient.List,
			AppStatusServiceClientER.List,
			AppStatusServiceServer.List,
			AppStatusServiceServerER.List,
		},
	}
)

// AppStatusService / Server

type AppStatusServiceServer interface {
	// List List application instance statuses currently stored in Redis。
	List() []AppStatusView

	mustBeAppStatusServiceServer()
}

// AppStatusService / Server / DefaultServer

type DefaultAppStatusServiceServer struct{}

func (*DefaultAppStatusServiceServer) List() []AppStatusView {
	ex.PanicNew(ex.InvalidRequest, "method list is not implemented")
	return []AppStatusView{}
}

func (*DefaultAppStatusServiceServer) mustBeAppStatusServiceServer() {}

// AppStatusService / ERServer

type AppStatusServiceServerER interface {
	List() ([]AppStatusView, ex.Error)

	mustBeAppStatusServiceServerER()
}

// AppStatusService / ERServer / WrapperERServer

type _WrapperAppStatusServiceServerER struct {
	DefaultAppStatusServiceServer
	serverImpl AppStatusServiceServer
}

func _NewWrapperAppStatusServiceServerER(serverImpl AppStatusServiceServer) AppStatusServiceServerER {
	return &_WrapperAppStatusServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperAppStatusServiceServerER) server() AppStatusServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultAppStatusServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperAppStatusServiceServerER) List() (ret []AppStatusView, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().List()
	return
}

func (*_WrapperAppStatusServiceServerER) mustBeAppStatusServiceServerER() {}

// AppStatusService / ERServer / DefaultERServer

type DefaultAppStatusServiceServerER struct {
	_WrapperAppStatusServiceServerER
}

// AppStatusService / Client

type AppStatusServiceClient interface {
	// List List application instance statuses currently stored in Redis。
	List(_ivOpts ...rpcclient.InvokeOption) []AppStatusView
}

type _AppStatusServiceClient struct {
	clientER AppStatusServiceClientER
}

func NewAppStatusServiceClient(clientER AppStatusServiceClientER) AppStatusServiceClient {
	return &_AppStatusServiceClient{clientER: clientER}
}

func (client *_AppStatusServiceClient) List(_ivOpts ...rpcclient.InvokeOption) []AppStatusView {
	ret, err := client.clientER.List(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// AppStatusService / ERClient

type AppStatusServiceClientER interface {
	// List List application instance statuses currently stored in Redis。
	List(_ivOpts ...rpcclient.InvokeOption) ([]AppStatusView, ex.Error)
}

type _AppStatusServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewAppStatusServiceClientER(rpcClient *rpcclient.Client) AppStatusServiceClientER {
	return &_AppStatusServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_AppStatusServiceClientER) List(_ivOpts ...rpcclient.InvokeOption) ([]AppStatusView, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_AppStatusServiceListSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]AppStatusView)
	err, _ := errI.(ex.Error)
	return ret, err
}

// EventDebugServiceServer Hub Dashboard Event Debugging Service

// EventDebugService / Spec

var (
	_EventDebugServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "EventDebugService",
		SkelName:          "vine.hub.EventDebugService",
		Hash:              "7ca4a329",
		ServerType:        reflect.TypeFor[EventDebugServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultEventDebugServiceServer](),
		ClientType:        reflect.TypeFor[EventDebugServiceClient](),
		ClientCtor:        NewEventDebugServiceClient,

		ERServerType:        reflect.TypeFor[EventDebugServiceServerER](),
		WrapperERServerCtor: _NewWrapperEventDebugServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultEventDebugServiceServerER](),
		ERClientType:        reflect.TypeFor[EventDebugServiceClientER](),
		ERClientCtor:        NewEventDebugServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_EventDebugServiceListEventsSpec,
			_EventDebugServiceBuildDefaultEmitRequestSpec,
			_EventDebugServiceEmitEventSpec,
		},
	}
	_EventDebugServiceListEventsSpec = &rpcspec.MethodSpec{
		Name:              "ListEvents",
		SkelName:          "listEvents",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]EventDebugEventItem](),
		ValidateResult: func(value any) error {
			ret := value.([]EventDebugEventItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			EventDebugServiceClient.ListEvents,
			EventDebugServiceClientER.ListEvents,
			EventDebugServiceServer.ListEvents,
			EventDebugServiceServerER.ListEvents,
		},
	}
	_EventDebugServiceBuildDefaultEmitRequestSpec = &rpcspec.MethodSpec{
		Name:                        "BuildDefaultEmitRequest",
		SkelName:                    "buildDefaultEmitRequest",
		ArgumentsType:               reflect.TypeFor[_EventDebugServiceBuildDefaultEmitRequestArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[EventDebugDefaultEmitRequest](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			EventDebugServiceClient.BuildDefaultEmitRequest,
			EventDebugServiceClientER.BuildDefaultEmitRequest,
			EventDebugServiceServer.BuildDefaultEmitRequest,
			EventDebugServiceServerER.BuildDefaultEmitRequest,
		},
	}
	_EventDebugServiceEmitEventSpec = &rpcspec.MethodSpec{
		Name:                        "EmitEvent",
		SkelName:                    "emitEvent",
		ArgumentsType:               reflect.TypeFor[_EventDebugServiceEmitEventArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			EventDebugServiceClient.EmitEvent,
			EventDebugServiceClientER.EmitEvent,
			EventDebugServiceServer.EmitEvent,
			EventDebugServiceServerER.EmitEvent,
		},
	}
)

// EventDebugService / Arguments

type _EventDebugServiceBuildDefaultEmitRequestArguments struct {
	EventSkelName string `json:"eventSkelName" arg:"0"`
	SchemaHash    string `json:"schemaHash" arg:"1"`
}

type _EventDebugServiceEmitEventArguments struct {
	Request EventDebugEmitRequest `json:"request" arg:"0"`
}

// EventDebugService / Server

type EventDebugServiceServer interface {
	// ListEvents List the events monitored by the application instance。
	ListEvents() []EventDebugEventItem
	// BuildDefaultEmitRequest Generate a default Event send request。
	//   @param eventSkelName - Event Skel name
	//   @param schemaHash - Event schema hash
	BuildDefaultEmitRequest(eventSkelName string, schemaHash string) EventDebugDefaultEmitRequest
	// EmitEvent Send Event。
	//   @param request - Debug send request
	EmitEvent(request EventDebugEmitRequest)

	mustBeEventDebugServiceServer()
}

// EventDebugService / Server / DefaultServer

type DefaultEventDebugServiceServer struct{}

func (*DefaultEventDebugServiceServer) ListEvents() []EventDebugEventItem {
	ex.PanicNew(ex.InvalidRequest, "method listEvents is not implemented")
	return []EventDebugEventItem{}
}

func (*DefaultEventDebugServiceServer) BuildDefaultEmitRequest(string, string) EventDebugDefaultEmitRequest {
	ex.PanicNew(ex.InvalidRequest, "method buildDefaultEmitRequest is not implemented")
	return EventDebugDefaultEmitRequest{}
}

func (*DefaultEventDebugServiceServer) EmitEvent(EventDebugEmitRequest) {
	ex.PanicNew(ex.InvalidRequest, "method emitEvent is not implemented")
}

func (*DefaultEventDebugServiceServer) mustBeEventDebugServiceServer() {}

// EventDebugService / ERServer

type EventDebugServiceServerER interface {
	ListEvents() ([]EventDebugEventItem, ex.Error)
	BuildDefaultEmitRequest(eventSkelName string, schemaHash string) (EventDebugDefaultEmitRequest, ex.Error)
	EmitEvent(request EventDebugEmitRequest) ex.Error

	mustBeEventDebugServiceServerER()
}

// EventDebugService / ERServer / WrapperERServer

type _WrapperEventDebugServiceServerER struct {
	DefaultEventDebugServiceServer
	serverImpl EventDebugServiceServer
}

func _NewWrapperEventDebugServiceServerER(serverImpl EventDebugServiceServer) EventDebugServiceServerER {
	return &_WrapperEventDebugServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperEventDebugServiceServerER) server() EventDebugServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultEventDebugServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperEventDebugServiceServerER) ListEvents() (ret []EventDebugEventItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListEvents()
	return
}

func (service *_WrapperEventDebugServiceServerER) BuildDefaultEmitRequest(eventSkelName string, schemaHash string) (ret EventDebugDefaultEmitRequest, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().BuildDefaultEmitRequest(eventSkelName, schemaHash)
	return
}

func (service *_WrapperEventDebugServiceServerER) EmitEvent(request EventDebugEmitRequest) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().EmitEvent(request)
	return
}

func (*_WrapperEventDebugServiceServerER) mustBeEventDebugServiceServerER() {}

// EventDebugService / ERServer / DefaultERServer

type DefaultEventDebugServiceServerER struct {
	_WrapperEventDebugServiceServerER
}

// EventDebugService / Client

type EventDebugServiceClient interface {
	// ListEvents List the events monitored by the application instance。
	ListEvents(_ivOpts ...rpcclient.InvokeOption) []EventDebugEventItem
	// BuildDefaultEmitRequest Generate a default Event send request。
	//   @param eventSkelName - Event Skel name
	//   @param schemaHash - Event schema hash
	BuildDefaultEmitRequest(eventSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) EventDebugDefaultEmitRequest
	// EmitEvent Send Event。
	//   @param request - Debug send request
	EmitEvent(request EventDebugEmitRequest, _ivOpts ...rpcclient.InvokeOption)
}

type _EventDebugServiceClient struct {
	clientER EventDebugServiceClientER
}

func NewEventDebugServiceClient(clientER EventDebugServiceClientER) EventDebugServiceClient {
	return &_EventDebugServiceClient{clientER: clientER}
}

func (client *_EventDebugServiceClient) ListEvents(_ivOpts ...rpcclient.InvokeOption) []EventDebugEventItem {
	ret, err := client.clientER.ListEvents(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_EventDebugServiceClient) BuildDefaultEmitRequest(eventSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) EventDebugDefaultEmitRequest {
	ret, err := client.clientER.BuildDefaultEmitRequest(eventSkelName, schemaHash, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_EventDebugServiceClient) EmitEvent(request EventDebugEmitRequest, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.EmitEvent(request, _ivOpts...)
	ex.PanicIfError(err)
}

// EventDebugService / ERClient

type EventDebugServiceClientER interface {
	// ListEvents List the events monitored by the application instance。
	ListEvents(_ivOpts ...rpcclient.InvokeOption) ([]EventDebugEventItem, ex.Error)
	// BuildDefaultEmitRequest Generate a default Event send request。
	//   @param eventSkelName - Event Skel name
	//   @param schemaHash - Event schema hash
	BuildDefaultEmitRequest(eventSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) (EventDebugDefaultEmitRequest, ex.Error)
	// EmitEvent Send Event。
	//   @param request - Debug send request
	EmitEvent(request EventDebugEmitRequest, _ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _EventDebugServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewEventDebugServiceClientER(rpcClient *rpcclient.Client) EventDebugServiceClientER {
	return &_EventDebugServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_EventDebugServiceClientER) ListEvents(_ivOpts ...rpcclient.InvokeOption) ([]EventDebugEventItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_EventDebugServiceListEventsSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]EventDebugEventItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_EventDebugServiceClientER) BuildDefaultEmitRequest(eventSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) (EventDebugDefaultEmitRequest, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_EventDebugServiceBuildDefaultEmitRequestSpec.Info(), &_EventDebugServiceBuildDefaultEmitRequestArguments{
		EventSkelName: eventSkelName,
		SchemaHash:    schemaHash,
	}, _ivOpts...)
	ret, _ := retI.(EventDebugDefaultEmitRequest)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_EventDebugServiceClientER) EmitEvent(request EventDebugEmitRequest, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_EventDebugServiceEmitEventSpec.Info(), &_EventDebugServiceEmitEventArguments{
		Request: request,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

// InfoServiceServer Hub's information service, called by Link

// InfoService / Spec

var (
	_InfoServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "InfoService",
		SkelName:          "vine.hub.InfoService",
		Hash:              "b28553d7",
		ServerType:        reflect.TypeFor[InfoServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultInfoServiceServer](),
		ClientType:        reflect.TypeFor[InfoServiceClient](),
		ClientCtor:        NewInfoServiceClient,

		ERServerType:        reflect.TypeFor[InfoServiceServerER](),
		WrapperERServerCtor: _NewWrapperInfoServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultInfoServiceServerER](),
		ERClientType:        reflect.TypeFor[InfoServiceClientER](),
		ERClientCtor:        NewInfoServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_InfoServiceGetInfoSpec,
		},
	}
	_InfoServiceGetInfoSpec = &rpcspec.MethodSpec{
		Name:                        "GetInfo",
		SkelName:                    "getInfo",
		ArgumentsType:               nil,
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[Info](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			InfoServiceClient.GetInfo,
			InfoServiceClientER.GetInfo,
			InfoServiceServer.GetInfo,
			InfoServiceServerER.GetInfo,
		},
	}
)

// InfoService / Server

type InfoServiceServer interface {
	// GetInfo Read Hub information。
	//   @returns Info - Hub information
	GetInfo() Info

	mustBeInfoServiceServer()
}

// InfoService / Server / DefaultServer

type DefaultInfoServiceServer struct{}

func (*DefaultInfoServiceServer) GetInfo() Info {
	ex.PanicNew(ex.InvalidRequest, "method getInfo is not implemented")
	return Info{}
}

func (*DefaultInfoServiceServer) mustBeInfoServiceServer() {}

// InfoService / ERServer

type InfoServiceServerER interface {
	GetInfo() (Info, ex.Error)

	mustBeInfoServiceServerER()
}

// InfoService / ERServer / WrapperERServer

type _WrapperInfoServiceServerER struct {
	DefaultInfoServiceServer
	serverImpl InfoServiceServer
}

func _NewWrapperInfoServiceServerER(serverImpl InfoServiceServer) InfoServiceServerER {
	return &_WrapperInfoServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperInfoServiceServerER) server() InfoServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultInfoServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperInfoServiceServerER) GetInfo() (ret Info, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().GetInfo()
	return
}

func (*_WrapperInfoServiceServerER) mustBeInfoServiceServerER() {}

// InfoService / ERServer / DefaultERServer

type DefaultInfoServiceServerER struct {
	_WrapperInfoServiceServerER
}

// InfoService / Client

type InfoServiceClient interface {
	// GetInfo Read Hub information。
	//   @returns Info - Hub information
	GetInfo(_ivOpts ...rpcclient.InvokeOption) Info
}

type _InfoServiceClient struct {
	clientER InfoServiceClientER
}

func NewInfoServiceClient(clientER InfoServiceClientER) InfoServiceClient {
	return &_InfoServiceClient{clientER: clientER}
}

func (client *_InfoServiceClient) GetInfo(_ivOpts ...rpcclient.InvokeOption) Info {
	ret, err := client.clientER.GetInfo(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// InfoService / ERClient

type InfoServiceClientER interface {
	// GetInfo Read Hub information。
	//   @returns Info - Hub information
	GetInfo(_ivOpts ...rpcclient.InvokeOption) (Info, ex.Error)
}

type _InfoServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewInfoServiceClientER(rpcClient *rpcclient.Client) InfoServiceClientER {
	return &_InfoServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_InfoServiceClientER) GetInfo(_ivOpts ...rpcclient.InvokeOption) (Info, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_InfoServiceGetInfoSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.(Info)
	err, _ := errI.(ex.Error)
	return ret, err
}

// MaintenanceServiceServer Hub maintenance service

// MaintenanceService / Spec

var (
	_MaintenanceServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "MaintenanceService",
		SkelName:          "vine.hub.MaintenanceService",
		Hash:              "689cac11",
		ServerType:        reflect.TypeFor[MaintenanceServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultMaintenanceServiceServer](),
		ClientType:        reflect.TypeFor[MaintenanceServiceClient](),
		ClientCtor:        NewMaintenanceServiceClient,

		ERServerType:        reflect.TypeFor[MaintenanceServiceServerER](),
		WrapperERServerCtor: _NewWrapperMaintenanceServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultMaintenanceServiceServerER](),
		ERClientType:        reflect.TypeFor[MaintenanceServiceClientER](),
		ERClientCtor:        NewMaintenanceServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_MaintenanceServicePreviewSeedYamlSpec,
			_MaintenanceServiceApplySeedYamlSpec,
		},
	}
	_MaintenanceServicePreviewSeedYamlSpec = &rpcspec.MethodSpec{
		Name:              "PreviewSeedYaml",
		SkelName:          "previewSeedYaml",
		ArgumentsType:     reflect.TypeFor[_MaintenanceServicePreviewSeedYamlArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[SeedPreview](),
		ValidateResult: func(value any) error {
			ret := value.(SeedPreview)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			MaintenanceServiceClient.PreviewSeedYaml,
			MaintenanceServiceClientER.PreviewSeedYaml,
			MaintenanceServiceServer.PreviewSeedYaml,
			MaintenanceServiceServerER.PreviewSeedYaml,
		},
	}
	_MaintenanceServiceApplySeedYamlSpec = &rpcspec.MethodSpec{
		Name:          "ApplySeedYaml",
		SkelName:      "applySeedYaml",
		ArgumentsType: reflect.TypeFor[_MaintenanceServiceApplySeedYamlArguments](),
		ValidateArguments: func(value any) error {
			args := value.(*_MaintenanceServiceApplySeedYamlArguments)
			if err := rpcspec.CheckValueNotNil(args.Selections, rpcspec.JoinPath("arguments", "Selections")); err != nil {
				return err
			}
			return nil
		},
		ResultType: reflect.TypeFor[SeedPreview](),
		ValidateResult: func(value any) error {
			ret := value.(SeedPreview)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			MaintenanceServiceClient.ApplySeedYaml,
			MaintenanceServiceClientER.ApplySeedYaml,
			MaintenanceServiceServer.ApplySeedYaml,
			MaintenanceServiceServerER.ApplySeedYaml,
		},
	}
)

// MaintenanceService / Arguments

type _MaintenanceServicePreviewSeedYamlArguments struct {
	Content string `json:"content" arg:"0"`
}

type _MaintenanceServiceApplySeedYamlArguments struct {
	Content    string              `json:"content" arg:"0"`
	Selections []SeedItemSelection `json:"selections" arg:"1"`
}

// MaintenanceService / Server

type MaintenanceServiceServer interface {
	// PreviewSeedYaml Preview Seed YAML differences。
	//   @param content - Seed YAML content
	//   @returns SeedPreview - Seed preview
	PreviewSeedYaml(content string) SeedPreview
	// ApplySeedYaml Apply Seed YAML entity updates。
	//   @param content - Seed YAML content
	//   @param selections - Entity to update
	//   @returns SeedPreview - Updated Seed preview
	ApplySeedYaml(content string, selections []SeedItemSelection) SeedPreview

	mustBeMaintenanceServiceServer()
}

// MaintenanceService / Server / DefaultServer

type DefaultMaintenanceServiceServer struct{}

func (*DefaultMaintenanceServiceServer) PreviewSeedYaml(string) SeedPreview {
	ex.PanicNew(ex.InvalidRequest, "method previewSeedYaml is not implemented")
	return SeedPreview{}
}

func (*DefaultMaintenanceServiceServer) ApplySeedYaml(string, []SeedItemSelection) SeedPreview {
	ex.PanicNew(ex.InvalidRequest, "method applySeedYaml is not implemented")
	return SeedPreview{}
}

func (*DefaultMaintenanceServiceServer) mustBeMaintenanceServiceServer() {}

// MaintenanceService / ERServer

type MaintenanceServiceServerER interface {
	PreviewSeedYaml(content string) (SeedPreview, ex.Error)
	ApplySeedYaml(content string, selections []SeedItemSelection) (SeedPreview, ex.Error)

	mustBeMaintenanceServiceServerER()
}

// MaintenanceService / ERServer / WrapperERServer

type _WrapperMaintenanceServiceServerER struct {
	DefaultMaintenanceServiceServer
	serverImpl MaintenanceServiceServer
}

func _NewWrapperMaintenanceServiceServerER(serverImpl MaintenanceServiceServer) MaintenanceServiceServerER {
	return &_WrapperMaintenanceServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperMaintenanceServiceServerER) server() MaintenanceServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultMaintenanceServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperMaintenanceServiceServerER) PreviewSeedYaml(content string) (ret SeedPreview, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().PreviewSeedYaml(content)
	return
}

func (service *_WrapperMaintenanceServiceServerER) ApplySeedYaml(content string, selections []SeedItemSelection) (ret SeedPreview, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ApplySeedYaml(content, selections)
	return
}

func (*_WrapperMaintenanceServiceServerER) mustBeMaintenanceServiceServerER() {}

// MaintenanceService / ERServer / DefaultERServer

type DefaultMaintenanceServiceServerER struct {
	_WrapperMaintenanceServiceServerER
}

// MaintenanceService / Client

type MaintenanceServiceClient interface {
	// PreviewSeedYaml Preview Seed YAML differences。
	//   @param content - Seed YAML content
	//   @returns SeedPreview - Seed preview
	PreviewSeedYaml(content string, _ivOpts ...rpcclient.InvokeOption) SeedPreview
	// ApplySeedYaml Apply Seed YAML entity updates。
	//   @param content - Seed YAML content
	//   @param selections - Entity to update
	//   @returns SeedPreview - Updated Seed preview
	ApplySeedYaml(content string, selections []SeedItemSelection, _ivOpts ...rpcclient.InvokeOption) SeedPreview
}

type _MaintenanceServiceClient struct {
	clientER MaintenanceServiceClientER
}

func NewMaintenanceServiceClient(clientER MaintenanceServiceClientER) MaintenanceServiceClient {
	return &_MaintenanceServiceClient{clientER: clientER}
}

func (client *_MaintenanceServiceClient) PreviewSeedYaml(content string, _ivOpts ...rpcclient.InvokeOption) SeedPreview {
	ret, err := client.clientER.PreviewSeedYaml(content, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_MaintenanceServiceClient) ApplySeedYaml(content string, selections []SeedItemSelection, _ivOpts ...rpcclient.InvokeOption) SeedPreview {
	ret, err := client.clientER.ApplySeedYaml(content, selections, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// MaintenanceService / ERClient

type MaintenanceServiceClientER interface {
	// PreviewSeedYaml Preview Seed YAML differences。
	//   @param content - Seed YAML content
	//   @returns SeedPreview - Seed preview
	PreviewSeedYaml(content string, _ivOpts ...rpcclient.InvokeOption) (SeedPreview, ex.Error)
	// ApplySeedYaml Apply Seed YAML entity updates。
	//   @param content - Seed YAML content
	//   @param selections - Entity to update
	//   @returns SeedPreview - Updated Seed preview
	ApplySeedYaml(content string, selections []SeedItemSelection, _ivOpts ...rpcclient.InvokeOption) (SeedPreview, ex.Error)
}

type _MaintenanceServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewMaintenanceServiceClientER(rpcClient *rpcclient.Client) MaintenanceServiceClientER {
	return &_MaintenanceServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_MaintenanceServiceClientER) PreviewSeedYaml(content string, _ivOpts ...rpcclient.InvokeOption) (SeedPreview, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_MaintenanceServicePreviewSeedYamlSpec.Info(), &_MaintenanceServicePreviewSeedYamlArguments{
		Content: content,
	}, _ivOpts...)
	ret, _ := retI.(SeedPreview)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_MaintenanceServiceClientER) ApplySeedYaml(content string, selections []SeedItemSelection, _ivOpts ...rpcclient.InvokeOption) (SeedPreview, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_MaintenanceServiceApplySeedYamlSpec.Info(), &_MaintenanceServiceApplySeedYamlArguments{
		Content:    content,
		Selections: selections,
	}, _ivOpts...)
	ret, _ := retI.(SeedPreview)
	err, _ := errI.(ex.Error)
	return ret, err
}

// PortalCertServiceServer Hub's Portal site certificate service, called by the Portal management client

// PortalCertService / Spec

var (
	_PortalCertServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "PortalCertService",
		SkelName:          "vine.hub.PortalCertService",
		Hash:              "6bacbfdb",
		ServerType:        reflect.TypeFor[PortalCertServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultPortalCertServiceServer](),
		ClientType:        reflect.TypeFor[PortalCertServiceClient](),
		ClientCtor:        NewPortalCertServiceClient,

		ERServerType:        reflect.TypeFor[PortalCertServiceServerER](),
		WrapperERServerCtor: _NewWrapperPortalCertServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultPortalCertServiceServerER](),
		ERClientType:        reflect.TypeFor[PortalCertServiceClientER](),
		ERClientCtor:        NewPortalCertServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_PortalCertServiceListSpec,
			_PortalCertServiceGetSpec,
			_PortalCertServiceCreateSpec,
			_PortalCertServiceUpdateSpec,
			_PortalCertServiceRemoveSpec,
		},
	}
	_PortalCertServiceListSpec = &rpcspec.MethodSpec{
		Name:              "List",
		SkelName:          "list",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]PortalCert](),
		ValidateResult: func(value any) error {
			ret := value.([]PortalCert)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalCertServiceClient.List,
			PortalCertServiceClientER.List,
			PortalCertServiceServer.List,
			PortalCertServiceServerER.List,
		},
	}
	_PortalCertServiceGetSpec = &rpcspec.MethodSpec{
		Name:              "Get",
		SkelName:          "get",
		ArgumentsType:     reflect.TypeFor[_PortalCertServiceGetArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[PortalCert](),
		ValidateResult: func(value any) error {
			ret := value.(PortalCert)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalCertServiceClient.Get,
			PortalCertServiceClientER.Get,
			PortalCertServiceServer.Get,
			PortalCertServiceServerER.Get,
		},
	}
	_PortalCertServiceCreateSpec = &rpcspec.MethodSpec{
		Name:              "Create",
		SkelName:          "create",
		ArgumentsType:     reflect.TypeFor[_PortalCertServiceCreateArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[PortalCert](),
		ValidateResult: func(value any) error {
			ret := value.(PortalCert)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalCertServiceClient.Create,
			PortalCertServiceClientER.Create,
			PortalCertServiceServer.Create,
			PortalCertServiceServerER.Create,
		},
	}
	_PortalCertServiceUpdateSpec = &rpcspec.MethodSpec{
		Name:              "Update",
		SkelName:          "update",
		ArgumentsType:     reflect.TypeFor[_PortalCertServiceUpdateArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[PortalCert](),
		ValidateResult: func(value any) error {
			ret := value.(PortalCert)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalCertServiceClient.Update,
			PortalCertServiceClientER.Update,
			PortalCertServiceServer.Update,
			PortalCertServiceServerER.Update,
		},
	}
	_PortalCertServiceRemoveSpec = &rpcspec.MethodSpec{
		Name:                        "Remove",
		SkelName:                    "remove",
		ArgumentsType:               reflect.TypeFor[_PortalCertServiceRemoveArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalCertServiceClient.Remove,
			PortalCertServiceClientER.Remove,
			PortalCertServiceServer.Remove,
			PortalCertServiceServerER.Remove,
		},
	}
)

// PortalCertService / Arguments

type _PortalCertServiceGetArguments struct {
	Id int `json:"id" arg:"0"`
}

type _PortalCertServiceCreateArguments struct {
	Creation PortalCertCreation `json:"creation" arg:"0"`
}

type _PortalCertServiceUpdateArguments struct {
	Id     int              `json:"id" arg:"0"`
	Update PortalCertUpdate `json:"update" arg:"1"`
}

type _PortalCertServiceRemoveArguments struct {
	Id int `json:"id" arg:"0"`
}

// PortalCertService / Server

type PortalCertServiceServer interface {
	// List List Portal site certificates。
	//   @returns []PortalCert - Portal site certificate list
	List() []PortalCert
	// Get Read the Portal site certificate。
	//   @param id - Certificate ID
	//   @returns PortalCert - Portal site certificate
	Get(id int) PortalCert
	// Create Create Portal site certificate。
	//   @param creation - Portal site certificate creation parameters
	//   @returns PortalCert - Portal site certificate
	Create(creation PortalCertCreation) PortalCert
	// Update Modify Portal site certificate。
	//   @param id - Certificate ID
	//   @param update - Portal site certificate update parameters
	//   @returns PortalCert - Portal site certificate
	Update(id int, update PortalCertUpdate) PortalCert
	// Remove Delete Portal site certificate。
	//   @param id - Certificate ID
	Remove(id int)

	mustBePortalCertServiceServer()
}

// PortalCertService / Server / DefaultServer

type DefaultPortalCertServiceServer struct{}

func (*DefaultPortalCertServiceServer) List() []PortalCert {
	ex.PanicNew(ex.InvalidRequest, "method list is not implemented")
	return []PortalCert{}
}

func (*DefaultPortalCertServiceServer) Get(int) PortalCert {
	ex.PanicNew(ex.InvalidRequest, "method get is not implemented")
	return PortalCert{}
}

func (*DefaultPortalCertServiceServer) Create(PortalCertCreation) PortalCert {
	ex.PanicNew(ex.InvalidRequest, "method create is not implemented")
	return PortalCert{}
}

func (*DefaultPortalCertServiceServer) Update(int, PortalCertUpdate) PortalCert {
	ex.PanicNew(ex.InvalidRequest, "method update is not implemented")
	return PortalCert{}
}

func (*DefaultPortalCertServiceServer) Remove(int) {
	ex.PanicNew(ex.InvalidRequest, "method remove is not implemented")
}

func (*DefaultPortalCertServiceServer) mustBePortalCertServiceServer() {}

// PortalCertService / ERServer

type PortalCertServiceServerER interface {
	List() ([]PortalCert, ex.Error)
	Get(id int) (PortalCert, ex.Error)
	Create(creation PortalCertCreation) (PortalCert, ex.Error)
	Update(id int, update PortalCertUpdate) (PortalCert, ex.Error)
	Remove(id int) ex.Error

	mustBePortalCertServiceServerER()
}

// PortalCertService / ERServer / WrapperERServer

type _WrapperPortalCertServiceServerER struct {
	DefaultPortalCertServiceServer
	serverImpl PortalCertServiceServer
}

func _NewWrapperPortalCertServiceServerER(serverImpl PortalCertServiceServer) PortalCertServiceServerER {
	return &_WrapperPortalCertServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperPortalCertServiceServerER) server() PortalCertServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultPortalCertServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperPortalCertServiceServerER) List() (ret []PortalCert, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().List()
	return
}

func (service *_WrapperPortalCertServiceServerER) Get(id int) (ret PortalCert, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Get(id)
	return
}

func (service *_WrapperPortalCertServiceServerER) Create(creation PortalCertCreation) (ret PortalCert, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Create(creation)
	return
}

func (service *_WrapperPortalCertServiceServerER) Update(id int, update PortalCertUpdate) (ret PortalCert, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Update(id, update)
	return
}

func (service *_WrapperPortalCertServiceServerER) Remove(id int) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().Remove(id)
	return
}

func (*_WrapperPortalCertServiceServerER) mustBePortalCertServiceServerER() {}

// PortalCertService / ERServer / DefaultERServer

type DefaultPortalCertServiceServerER struct {
	_WrapperPortalCertServiceServerER
}

// PortalCertService / Client

type PortalCertServiceClient interface {
	// List List Portal site certificates。
	//   @returns []PortalCert - Portal site certificate list
	List(_ivOpts ...rpcclient.InvokeOption) []PortalCert
	// Get Read the Portal site certificate。
	//   @param id - Certificate ID
	//   @returns PortalCert - Portal site certificate
	Get(id int, _ivOpts ...rpcclient.InvokeOption) PortalCert
	// Create Create Portal site certificate。
	//   @param creation - Portal site certificate creation parameters
	//   @returns PortalCert - Portal site certificate
	Create(creation PortalCertCreation, _ivOpts ...rpcclient.InvokeOption) PortalCert
	// Update Modify Portal site certificate。
	//   @param id - Certificate ID
	//   @param update - Portal site certificate update parameters
	//   @returns PortalCert - Portal site certificate
	Update(id int, update PortalCertUpdate, _ivOpts ...rpcclient.InvokeOption) PortalCert
	// Remove Delete Portal site certificate。
	//   @param id - Certificate ID
	Remove(id int, _ivOpts ...rpcclient.InvokeOption)
}

type _PortalCertServiceClient struct {
	clientER PortalCertServiceClientER
}

func NewPortalCertServiceClient(clientER PortalCertServiceClientER) PortalCertServiceClient {
	return &_PortalCertServiceClient{clientER: clientER}
}

func (client *_PortalCertServiceClient) List(_ivOpts ...rpcclient.InvokeOption) []PortalCert {
	ret, err := client.clientER.List(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalCertServiceClient) Get(id int, _ivOpts ...rpcclient.InvokeOption) PortalCert {
	ret, err := client.clientER.Get(id, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalCertServiceClient) Create(creation PortalCertCreation, _ivOpts ...rpcclient.InvokeOption) PortalCert {
	ret, err := client.clientER.Create(creation, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalCertServiceClient) Update(id int, update PortalCertUpdate, _ivOpts ...rpcclient.InvokeOption) PortalCert {
	ret, err := client.clientER.Update(id, update, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalCertServiceClient) Remove(id int, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.Remove(id, _ivOpts...)
	ex.PanicIfError(err)
}

// PortalCertService / ERClient

type PortalCertServiceClientER interface {
	// List List Portal site certificates。
	//   @returns []PortalCert - Portal site certificate list
	List(_ivOpts ...rpcclient.InvokeOption) ([]PortalCert, ex.Error)
	// Get Read the Portal site certificate。
	//   @param id - Certificate ID
	//   @returns PortalCert - Portal site certificate
	Get(id int, _ivOpts ...rpcclient.InvokeOption) (PortalCert, ex.Error)
	// Create Create Portal site certificate。
	//   @param creation - Portal site certificate creation parameters
	//   @returns PortalCert - Portal site certificate
	Create(creation PortalCertCreation, _ivOpts ...rpcclient.InvokeOption) (PortalCert, ex.Error)
	// Update Modify Portal site certificate。
	//   @param id - Certificate ID
	//   @param update - Portal site certificate update parameters
	//   @returns PortalCert - Portal site certificate
	Update(id int, update PortalCertUpdate, _ivOpts ...rpcclient.InvokeOption) (PortalCert, ex.Error)
	// Remove Delete Portal site certificate。
	//   @param id - Certificate ID
	Remove(id int, _ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _PortalCertServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewPortalCertServiceClientER(rpcClient *rpcclient.Client) PortalCertServiceClientER {
	return &_PortalCertServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_PortalCertServiceClientER) List(_ivOpts ...rpcclient.InvokeOption) ([]PortalCert, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalCertServiceListSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]PortalCert)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalCertServiceClientER) Get(id int, _ivOpts ...rpcclient.InvokeOption) (PortalCert, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalCertServiceGetSpec.Info(), &_PortalCertServiceGetArguments{
		Id: id,
	}, _ivOpts...)
	ret, _ := retI.(PortalCert)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalCertServiceClientER) Create(creation PortalCertCreation, _ivOpts ...rpcclient.InvokeOption) (PortalCert, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalCertServiceCreateSpec.Info(), &_PortalCertServiceCreateArguments{
		Creation: creation,
	}, _ivOpts...)
	ret, _ := retI.(PortalCert)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalCertServiceClientER) Update(id int, update PortalCertUpdate, _ivOpts ...rpcclient.InvokeOption) (PortalCert, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalCertServiceUpdateSpec.Info(), &_PortalCertServiceUpdateArguments{
		Id:     id,
		Update: update,
	}, _ivOpts...)
	ret, _ := retI.(PortalCert)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalCertServiceClientER) Remove(id int, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_PortalCertServiceRemoveSpec.Info(), &_PortalCertServiceRemoveArguments{
		Id: id,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

// PortalEntryServiceServer Hub's Portal access entry service, called by the Portal management client

// PortalEntryService / Spec

var (
	_PortalEntryServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "PortalEntryService",
		SkelName:          "vine.hub.PortalEntryService",
		Hash:              "0abee2e7",
		ServerType:        reflect.TypeFor[PortalEntryServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultPortalEntryServiceServer](),
		ClientType:        reflect.TypeFor[PortalEntryServiceClient](),
		ClientCtor:        NewPortalEntryServiceClient,

		ERServerType:        reflect.TypeFor[PortalEntryServiceServerER](),
		WrapperERServerCtor: _NewWrapperPortalEntryServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultPortalEntryServiceServerER](),
		ERClientType:        reflect.TypeFor[PortalEntryServiceClientER](),
		ERClientCtor:        NewPortalEntryServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_PortalEntryServiceListSpec,
			_PortalEntryServiceUpdateAccessSpec,
		},
	}
	_PortalEntryServiceListSpec = &rpcspec.MethodSpec{
		Name:              "List",
		SkelName:          "list",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]PortalEntry](),
		ValidateResult: func(value any) error {
			ret := value.([]PortalEntry)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalEntryServiceClient.List,
			PortalEntryServiceClientER.List,
			PortalEntryServiceServer.List,
			PortalEntryServiceServerER.List,
		},
	}
	_PortalEntryServiceUpdateAccessSpec = &rpcspec.MethodSpec{
		Name:              "UpdateAccess",
		SkelName:          "updateAccess",
		ArgumentsType:     reflect.TypeFor[_PortalEntryServiceUpdateAccessArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[PortalEntry](),
		ValidateResult: func(value any) error {
			ret := value.(PortalEntry)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalEntryServiceClient.UpdateAccess,
			PortalEntryServiceClientER.UpdateAccess,
			PortalEntryServiceServer.UpdateAccess,
			PortalEntryServiceServerER.UpdateAccess,
		},
	}
)

// PortalEntryService / Arguments

type _PortalEntryServiceUpdateAccessArguments struct {
	Scheme string                  `json:"scheme" arg:"0"`
	Host   string                  `json:"host" arg:"1"`
	Port   int                     `json:"port" arg:"2"`
	Update PortalEntryAccessUpdate `json:"update" arg:"3"`
}

// PortalEntryService / Server

type PortalEntryServiceServer interface {
	// List List Portal access entries。
	//   @returns []PortalEntry - Portal access entry list
	List() []PortalEntry
	// UpdateAccess Modify Portal access configuration。
	//   @param scheme - Entry protocol
	//   @param host - Match Host, empty string means no restriction
	//   @param port - Entry port
	//   @param update - Portal access entry configuration update parameters
	//   @returns PortalEntry - Portal access entry
	UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate) PortalEntry

	mustBePortalEntryServiceServer()
}

// PortalEntryService / Server / DefaultServer

type DefaultPortalEntryServiceServer struct{}

func (*DefaultPortalEntryServiceServer) List() []PortalEntry {
	ex.PanicNew(ex.InvalidRequest, "method list is not implemented")
	return []PortalEntry{}
}

func (*DefaultPortalEntryServiceServer) UpdateAccess(string, string, int, PortalEntryAccessUpdate) PortalEntry {
	ex.PanicNew(ex.InvalidRequest, "method updateAccess is not implemented")
	return PortalEntry{}
}

func (*DefaultPortalEntryServiceServer) mustBePortalEntryServiceServer() {}

// PortalEntryService / ERServer

type PortalEntryServiceServerER interface {
	List() ([]PortalEntry, ex.Error)
	UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate) (PortalEntry, ex.Error)

	mustBePortalEntryServiceServerER()
}

// PortalEntryService / ERServer / WrapperERServer

type _WrapperPortalEntryServiceServerER struct {
	DefaultPortalEntryServiceServer
	serverImpl PortalEntryServiceServer
}

func _NewWrapperPortalEntryServiceServerER(serverImpl PortalEntryServiceServer) PortalEntryServiceServerER {
	return &_WrapperPortalEntryServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperPortalEntryServiceServerER) server() PortalEntryServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultPortalEntryServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperPortalEntryServiceServerER) List() (ret []PortalEntry, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().List()
	return
}

func (service *_WrapperPortalEntryServiceServerER) UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate) (ret PortalEntry, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().UpdateAccess(scheme, host, port, update)
	return
}

func (*_WrapperPortalEntryServiceServerER) mustBePortalEntryServiceServerER() {}

// PortalEntryService / ERServer / DefaultERServer

type DefaultPortalEntryServiceServerER struct {
	_WrapperPortalEntryServiceServerER
}

// PortalEntryService / Client

type PortalEntryServiceClient interface {
	// List List Portal access entries。
	//   @returns []PortalEntry - Portal access entry list
	List(_ivOpts ...rpcclient.InvokeOption) []PortalEntry
	// UpdateAccess Modify Portal access configuration。
	//   @param scheme - Entry protocol
	//   @param host - Match Host, empty string means no restriction
	//   @param port - Entry port
	//   @param update - Portal access entry configuration update parameters
	//   @returns PortalEntry - Portal access entry
	UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate, _ivOpts ...rpcclient.InvokeOption) PortalEntry
}

type _PortalEntryServiceClient struct {
	clientER PortalEntryServiceClientER
}

func NewPortalEntryServiceClient(clientER PortalEntryServiceClientER) PortalEntryServiceClient {
	return &_PortalEntryServiceClient{clientER: clientER}
}

func (client *_PortalEntryServiceClient) List(_ivOpts ...rpcclient.InvokeOption) []PortalEntry {
	ret, err := client.clientER.List(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalEntryServiceClient) UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate, _ivOpts ...rpcclient.InvokeOption) PortalEntry {
	ret, err := client.clientER.UpdateAccess(scheme, host, port, update, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// PortalEntryService / ERClient

type PortalEntryServiceClientER interface {
	// List List Portal access entries。
	//   @returns []PortalEntry - Portal access entry list
	List(_ivOpts ...rpcclient.InvokeOption) ([]PortalEntry, ex.Error)
	// UpdateAccess Modify Portal access configuration。
	//   @param scheme - Entry protocol
	//   @param host - Match Host, empty string means no restriction
	//   @param port - Entry port
	//   @param update - Portal access entry configuration update parameters
	//   @returns PortalEntry - Portal access entry
	UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate, _ivOpts ...rpcclient.InvokeOption) (PortalEntry, ex.Error)
}

type _PortalEntryServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewPortalEntryServiceClientER(rpcClient *rpcclient.Client) PortalEntryServiceClientER {
	return &_PortalEntryServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_PortalEntryServiceClientER) List(_ivOpts ...rpcclient.InvokeOption) ([]PortalEntry, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalEntryServiceListSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]PortalEntry)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalEntryServiceClientER) UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate, _ivOpts ...rpcclient.InvokeOption) (PortalEntry, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalEntryServiceUpdateAccessSpec.Info(), &_PortalEntryServiceUpdateAccessArguments{
		Scheme: scheme,
		Host:   host,
		Port:   port,
		Update: update,
	}, _ivOpts...)
	ret, _ := retI.(PortalEntry)
	err, _ := errI.(ex.Error)
	return ret, err
}

// PortalRuleServiceServer Hub's Portal entry rule service, called by the Portal management client

// PortalRuleService / Spec

var (
	_PortalRuleServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "PortalRuleService",
		SkelName:          "vine.hub.PortalRuleService",
		Hash:              "5232897d",
		ServerType:        reflect.TypeFor[PortalRuleServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultPortalRuleServiceServer](),
		ClientType:        reflect.TypeFor[PortalRuleServiceClient](),
		ClientCtor:        NewPortalRuleServiceClient,

		ERServerType:        reflect.TypeFor[PortalRuleServiceServerER](),
		WrapperERServerCtor: _NewWrapperPortalRuleServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultPortalRuleServiceServerER](),
		ERClientType:        reflect.TypeFor[PortalRuleServiceClientER](),
		ERClientCtor:        NewPortalRuleServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_PortalRuleServiceListSpec,
			_PortalRuleServiceGetSpec,
			_PortalRuleServiceCreateSpec,
			_PortalRuleServiceUpdateSpec,
			_PortalRuleServiceRemoveSpec,
			_PortalRuleServiceGetDashboardAccessSpec,
			_PortalRuleServiceUpdateDashboardAccessSpec,
		},
	}
	_PortalRuleServiceListSpec = &rpcspec.MethodSpec{
		Name:              "List",
		SkelName:          "list",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]PortalRule](),
		ValidateResult: func(value any) error {
			ret := value.([]PortalRule)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalRuleServiceClient.List,
			PortalRuleServiceClientER.List,
			PortalRuleServiceServer.List,
			PortalRuleServiceServerER.List,
		},
	}
	_PortalRuleServiceGetSpec = &rpcspec.MethodSpec{
		Name:                        "Get",
		SkelName:                    "get",
		ArgumentsType:               reflect.TypeFor[_PortalRuleServiceGetArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[PortalRule](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalRuleServiceClient.Get,
			PortalRuleServiceClientER.Get,
			PortalRuleServiceServer.Get,
			PortalRuleServiceServerER.Get,
		},
	}
	_PortalRuleServiceCreateSpec = &rpcspec.MethodSpec{
		Name:                        "Create",
		SkelName:                    "create",
		ArgumentsType:               reflect.TypeFor[_PortalRuleServiceCreateArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[PortalRule](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalRuleServiceClient.Create,
			PortalRuleServiceClientER.Create,
			PortalRuleServiceServer.Create,
			PortalRuleServiceServerER.Create,
		},
	}
	_PortalRuleServiceUpdateSpec = &rpcspec.MethodSpec{
		Name:                        "Update",
		SkelName:                    "update",
		ArgumentsType:               reflect.TypeFor[_PortalRuleServiceUpdateArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[PortalRule](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalRuleServiceClient.Update,
			PortalRuleServiceClientER.Update,
			PortalRuleServiceServer.Update,
			PortalRuleServiceServerER.Update,
		},
	}
	_PortalRuleServiceRemoveSpec = &rpcspec.MethodSpec{
		Name:                        "Remove",
		SkelName:                    "remove",
		ArgumentsType:               reflect.TypeFor[_PortalRuleServiceRemoveArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalRuleServiceClient.Remove,
			PortalRuleServiceClientER.Remove,
			PortalRuleServiceServer.Remove,
			PortalRuleServiceServerER.Remove,
		},
	}
	_PortalRuleServiceGetDashboardAccessSpec = &rpcspec.MethodSpec{
		Name:                        "GetDashboardAccess",
		SkelName:                    "getDashboardAccess",
		ArgumentsType:               nil,
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[PortalDashboardAccess](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalRuleServiceClient.GetDashboardAccess,
			PortalRuleServiceClientER.GetDashboardAccess,
			PortalRuleServiceServer.GetDashboardAccess,
			PortalRuleServiceServerER.GetDashboardAccess,
		},
	}
	_PortalRuleServiceUpdateDashboardAccessSpec = &rpcspec.MethodSpec{
		Name:              "UpdateDashboardAccess",
		SkelName:          "updateDashboardAccess",
		ArgumentsType:     reflect.TypeFor[_PortalRuleServiceUpdateDashboardAccessArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]PortalRule](),
		ValidateResult: func(value any) error {
			ret := value.([]PortalRule)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalRuleServiceClient.UpdateDashboardAccess,
			PortalRuleServiceClientER.UpdateDashboardAccess,
			PortalRuleServiceServer.UpdateDashboardAccess,
			PortalRuleServiceServerER.UpdateDashboardAccess,
		},
	}
)

// PortalRuleService / Arguments

type _PortalRuleServiceGetArguments struct {
	Id int `json:"id" arg:"0"`
}

type _PortalRuleServiceCreateArguments struct {
	Creation PortalRuleCreation `json:"creation" arg:"0"`
}

type _PortalRuleServiceUpdateArguments struct {
	Id     int              `json:"id" arg:"0"`
	Update PortalRuleUpdate `json:"update" arg:"1"`
}

type _PortalRuleServiceRemoveArguments struct {
	Id int `json:"id" arg:"0"`
}

type _PortalRuleServiceUpdateDashboardAccessArguments struct {
	Scheme     string `json:"scheme" arg:"0"`
	Host       string `json:"host" arg:"1"`
	Port       int    `json:"port" arg:"2"`
	PathPrefix string `json:"pathPrefix" arg:"3"`
}

// PortalRuleService / Server

type PortalRuleServiceServer interface {
	// List List Portal entry rules。
	//   @returns []PortalRule - Portal entry rule list
	List() []PortalRule
	// Get Read Portal entry rules。
	//   @param id - Rule ID
	//   @returns PortalRule - Portal entry rules
	Get(id int) PortalRule
	// Create Create Portal entry rules。
	//   @param creation - Portal entry rule creation parameters
	//   @returns PortalRule - Portal entry rules
	Create(creation PortalRuleCreation) PortalRule
	// Update Modify Portal entry rules。
	//   @param id - Rule ID
	//   @param update - Portal entry rule update parameters
	//   @returns PortalRule - Portal entry rules
	Update(id int, update PortalRuleUpdate) PortalRule
	// Remove Delete Portal entry rules。
	//   @param id - Rule ID
	Remove(id int)
	// GetDashboardAccess Get the Hub Dashboard access entry。
	//   @returns PortalDashboardAccess - Hub Dashboard access entry
	GetDashboardAccess() PortalDashboardAccess
	// UpdateDashboardAccess Modify Hub Dashboard access entry。
	//   @param scheme - Hub Dashboard entry protocol
	//   @param host - Hub Dashboard entry host
	//   @param port - Hub Dashboard entry port
	//   @param pathPrefix - Hub Dashboard entry path prefix
	//   @returns []PortalRule - Hub Dashboard entry rules
	UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string) []PortalRule

	mustBePortalRuleServiceServer()
}

// PortalRuleService / Server / DefaultServer

type DefaultPortalRuleServiceServer struct{}

func (*DefaultPortalRuleServiceServer) List() []PortalRule {
	ex.PanicNew(ex.InvalidRequest, "method list is not implemented")
	return []PortalRule{}
}

func (*DefaultPortalRuleServiceServer) Get(int) PortalRule {
	ex.PanicNew(ex.InvalidRequest, "method get is not implemented")
	return PortalRule{}
}

func (*DefaultPortalRuleServiceServer) Create(PortalRuleCreation) PortalRule {
	ex.PanicNew(ex.InvalidRequest, "method create is not implemented")
	return PortalRule{}
}

func (*DefaultPortalRuleServiceServer) Update(int, PortalRuleUpdate) PortalRule {
	ex.PanicNew(ex.InvalidRequest, "method update is not implemented")
	return PortalRule{}
}

func (*DefaultPortalRuleServiceServer) Remove(int) {
	ex.PanicNew(ex.InvalidRequest, "method remove is not implemented")
}

func (*DefaultPortalRuleServiceServer) GetDashboardAccess() PortalDashboardAccess {
	ex.PanicNew(ex.InvalidRequest, "method getDashboardAccess is not implemented")
	return PortalDashboardAccess{}
}

func (*DefaultPortalRuleServiceServer) UpdateDashboardAccess(string, string, int, string) []PortalRule {
	ex.PanicNew(ex.InvalidRequest, "method updateDashboardAccess is not implemented")
	return []PortalRule{}
}

func (*DefaultPortalRuleServiceServer) mustBePortalRuleServiceServer() {}

// PortalRuleService / ERServer

type PortalRuleServiceServerER interface {
	List() ([]PortalRule, ex.Error)
	Get(id int) (PortalRule, ex.Error)
	Create(creation PortalRuleCreation) (PortalRule, ex.Error)
	Update(id int, update PortalRuleUpdate) (PortalRule, ex.Error)
	Remove(id int) ex.Error
	GetDashboardAccess() (PortalDashboardAccess, ex.Error)
	UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string) ([]PortalRule, ex.Error)

	mustBePortalRuleServiceServerER()
}

// PortalRuleService / ERServer / WrapperERServer

type _WrapperPortalRuleServiceServerER struct {
	DefaultPortalRuleServiceServer
	serverImpl PortalRuleServiceServer
}

func _NewWrapperPortalRuleServiceServerER(serverImpl PortalRuleServiceServer) PortalRuleServiceServerER {
	return &_WrapperPortalRuleServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperPortalRuleServiceServerER) server() PortalRuleServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultPortalRuleServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperPortalRuleServiceServerER) List() (ret []PortalRule, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().List()
	return
}

func (service *_WrapperPortalRuleServiceServerER) Get(id int) (ret PortalRule, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Get(id)
	return
}

func (service *_WrapperPortalRuleServiceServerER) Create(creation PortalRuleCreation) (ret PortalRule, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Create(creation)
	return
}

func (service *_WrapperPortalRuleServiceServerER) Update(id int, update PortalRuleUpdate) (ret PortalRule, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Update(id, update)
	return
}

func (service *_WrapperPortalRuleServiceServerER) Remove(id int) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().Remove(id)
	return
}

func (service *_WrapperPortalRuleServiceServerER) GetDashboardAccess() (ret PortalDashboardAccess, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().GetDashboardAccess()
	return
}

func (service *_WrapperPortalRuleServiceServerER) UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string) (ret []PortalRule, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().UpdateDashboardAccess(scheme, host, port, pathPrefix)
	return
}

func (*_WrapperPortalRuleServiceServerER) mustBePortalRuleServiceServerER() {}

// PortalRuleService / ERServer / DefaultERServer

type DefaultPortalRuleServiceServerER struct {
	_WrapperPortalRuleServiceServerER
}

// PortalRuleService / Client

type PortalRuleServiceClient interface {
	// List List Portal entry rules。
	//   @returns []PortalRule - Portal entry rule list
	List(_ivOpts ...rpcclient.InvokeOption) []PortalRule
	// Get Read Portal entry rules。
	//   @param id - Rule ID
	//   @returns PortalRule - Portal entry rules
	Get(id int, _ivOpts ...rpcclient.InvokeOption) PortalRule
	// Create Create Portal entry rules。
	//   @param creation - Portal entry rule creation parameters
	//   @returns PortalRule - Portal entry rules
	Create(creation PortalRuleCreation, _ivOpts ...rpcclient.InvokeOption) PortalRule
	// Update Modify Portal entry rules。
	//   @param id - Rule ID
	//   @param update - Portal entry rule update parameters
	//   @returns PortalRule - Portal entry rules
	Update(id int, update PortalRuleUpdate, _ivOpts ...rpcclient.InvokeOption) PortalRule
	// Remove Delete Portal entry rules。
	//   @param id - Rule ID
	Remove(id int, _ivOpts ...rpcclient.InvokeOption)
	// GetDashboardAccess Get the Hub Dashboard access entry。
	//   @returns PortalDashboardAccess - Hub Dashboard access entry
	GetDashboardAccess(_ivOpts ...rpcclient.InvokeOption) PortalDashboardAccess
	// UpdateDashboardAccess Modify Hub Dashboard access entry。
	//   @param scheme - Hub Dashboard entry protocol
	//   @param host - Hub Dashboard entry host
	//   @param port - Hub Dashboard entry port
	//   @param pathPrefix - Hub Dashboard entry path prefix
	//   @returns []PortalRule - Hub Dashboard entry rules
	UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string, _ivOpts ...rpcclient.InvokeOption) []PortalRule
}

type _PortalRuleServiceClient struct {
	clientER PortalRuleServiceClientER
}

func NewPortalRuleServiceClient(clientER PortalRuleServiceClientER) PortalRuleServiceClient {
	return &_PortalRuleServiceClient{clientER: clientER}
}

func (client *_PortalRuleServiceClient) List(_ivOpts ...rpcclient.InvokeOption) []PortalRule {
	ret, err := client.clientER.List(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalRuleServiceClient) Get(id int, _ivOpts ...rpcclient.InvokeOption) PortalRule {
	ret, err := client.clientER.Get(id, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalRuleServiceClient) Create(creation PortalRuleCreation, _ivOpts ...rpcclient.InvokeOption) PortalRule {
	ret, err := client.clientER.Create(creation, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalRuleServiceClient) Update(id int, update PortalRuleUpdate, _ivOpts ...rpcclient.InvokeOption) PortalRule {
	ret, err := client.clientER.Update(id, update, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalRuleServiceClient) Remove(id int, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.Remove(id, _ivOpts...)
	ex.PanicIfError(err)
}

func (client *_PortalRuleServiceClient) GetDashboardAccess(_ivOpts ...rpcclient.InvokeOption) PortalDashboardAccess {
	ret, err := client.clientER.GetDashboardAccess(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalRuleServiceClient) UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string, _ivOpts ...rpcclient.InvokeOption) []PortalRule {
	ret, err := client.clientER.UpdateDashboardAccess(scheme, host, port, pathPrefix, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// PortalRuleService / ERClient

type PortalRuleServiceClientER interface {
	// List List Portal entry rules。
	//   @returns []PortalRule - Portal entry rule list
	List(_ivOpts ...rpcclient.InvokeOption) ([]PortalRule, ex.Error)
	// Get Read Portal entry rules。
	//   @param id - Rule ID
	//   @returns PortalRule - Portal entry rules
	Get(id int, _ivOpts ...rpcclient.InvokeOption) (PortalRule, ex.Error)
	// Create Create Portal entry rules。
	//   @param creation - Portal entry rule creation parameters
	//   @returns PortalRule - Portal entry rules
	Create(creation PortalRuleCreation, _ivOpts ...rpcclient.InvokeOption) (PortalRule, ex.Error)
	// Update Modify Portal entry rules。
	//   @param id - Rule ID
	//   @param update - Portal entry rule update parameters
	//   @returns PortalRule - Portal entry rules
	Update(id int, update PortalRuleUpdate, _ivOpts ...rpcclient.InvokeOption) (PortalRule, ex.Error)
	// Remove Delete Portal entry rules。
	//   @param id - Rule ID
	Remove(id int, _ivOpts ...rpcclient.InvokeOption) ex.Error
	// GetDashboardAccess Get the Hub Dashboard access entry。
	//   @returns PortalDashboardAccess - Hub Dashboard access entry
	GetDashboardAccess(_ivOpts ...rpcclient.InvokeOption) (PortalDashboardAccess, ex.Error)
	// UpdateDashboardAccess Modify Hub Dashboard access entry。
	//   @param scheme - Hub Dashboard entry protocol
	//   @param host - Hub Dashboard entry host
	//   @param port - Hub Dashboard entry port
	//   @param pathPrefix - Hub Dashboard entry path prefix
	//   @returns []PortalRule - Hub Dashboard entry rules
	UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string, _ivOpts ...rpcclient.InvokeOption) ([]PortalRule, ex.Error)
}

type _PortalRuleServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewPortalRuleServiceClientER(rpcClient *rpcclient.Client) PortalRuleServiceClientER {
	return &_PortalRuleServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_PortalRuleServiceClientER) List(_ivOpts ...rpcclient.InvokeOption) ([]PortalRule, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalRuleServiceListSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]PortalRule)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalRuleServiceClientER) Get(id int, _ivOpts ...rpcclient.InvokeOption) (PortalRule, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalRuleServiceGetSpec.Info(), &_PortalRuleServiceGetArguments{
		Id: id,
	}, _ivOpts...)
	ret, _ := retI.(PortalRule)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalRuleServiceClientER) Create(creation PortalRuleCreation, _ivOpts ...rpcclient.InvokeOption) (PortalRule, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalRuleServiceCreateSpec.Info(), &_PortalRuleServiceCreateArguments{
		Creation: creation,
	}, _ivOpts...)
	ret, _ := retI.(PortalRule)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalRuleServiceClientER) Update(id int, update PortalRuleUpdate, _ivOpts ...rpcclient.InvokeOption) (PortalRule, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalRuleServiceUpdateSpec.Info(), &_PortalRuleServiceUpdateArguments{
		Id:     id,
		Update: update,
	}, _ivOpts...)
	ret, _ := retI.(PortalRule)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalRuleServiceClientER) Remove(id int, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_PortalRuleServiceRemoveSpec.Info(), &_PortalRuleServiceRemoveArguments{
		Id: id,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

func (client *_PortalRuleServiceClientER) GetDashboardAccess(_ivOpts ...rpcclient.InvokeOption) (PortalDashboardAccess, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalRuleServiceGetDashboardAccessSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.(PortalDashboardAccess)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalRuleServiceClientER) UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string, _ivOpts ...rpcclient.InvokeOption) ([]PortalRule, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalRuleServiceUpdateDashboardAccessSpec.Info(), &_PortalRuleServiceUpdateDashboardAccessArguments{
		Scheme:     scheme,
		Host:       host,
		Port:       port,
		PathPrefix: pathPrefix,
	}, _ivOpts...)
	ret, _ := retI.([]PortalRule)
	err, _ := errI.(ex.Error)
	return ret, err
}

// PortalSiteServiceServer Hub's Portal target site service, called by the Portal management client

// PortalSiteService / Spec

var (
	_PortalSiteServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "PortalSiteService",
		SkelName:          "vine.hub.PortalSiteService",
		Hash:              "a4d7f889",
		ServerType:        reflect.TypeFor[PortalSiteServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultPortalSiteServiceServer](),
		ClientType:        reflect.TypeFor[PortalSiteServiceClient](),
		ClientCtor:        NewPortalSiteServiceClient,

		ERServerType:        reflect.TypeFor[PortalSiteServiceServerER](),
		WrapperERServerCtor: _NewWrapperPortalSiteServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultPortalSiteServiceServerER](),
		ERClientType:        reflect.TypeFor[PortalSiteServiceClientER](),
		ERClientCtor:        NewPortalSiteServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_PortalSiteServiceListSpec,
			_PortalSiteServiceListOptionsSpec,
			_PortalSiteServiceGetSpec,
			_PortalSiteServiceCreateSpec,
			_PortalSiteServiceUpdateSpec,
			_PortalSiteServiceRemoveSpec,
		},
	}
	_PortalSiteServiceListSpec = &rpcspec.MethodSpec{
		Name:              "List",
		SkelName:          "list",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]PortalSite](),
		ValidateResult: func(value any) error {
			ret := value.([]PortalSite)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalSiteServiceClient.List,
			PortalSiteServiceClientER.List,
			PortalSiteServiceServer.List,
			PortalSiteServiceServerER.List,
		},
	}
	_PortalSiteServiceListOptionsSpec = &rpcspec.MethodSpec{
		Name:              "ListOptions",
		SkelName:          "listOptions",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[PortalSiteOptions](),
		ValidateResult: func(value any) error {
			ret := value.(PortalSiteOptions)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalSiteServiceClient.ListOptions,
			PortalSiteServiceClientER.ListOptions,
			PortalSiteServiceServer.ListOptions,
			PortalSiteServiceServerER.ListOptions,
		},
	}
	_PortalSiteServiceGetSpec = &rpcspec.MethodSpec{
		Name:              "Get",
		SkelName:          "get",
		ArgumentsType:     reflect.TypeFor[_PortalSiteServiceGetArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[PortalSite](),
		ValidateResult: func(value any) error {
			ret := value.(PortalSite)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalSiteServiceClient.Get,
			PortalSiteServiceClientER.Get,
			PortalSiteServiceServer.Get,
			PortalSiteServiceServerER.Get,
		},
	}
	_PortalSiteServiceCreateSpec = &rpcspec.MethodSpec{
		Name:          "Create",
		SkelName:      "create",
		ArgumentsType: reflect.TypeFor[_PortalSiteServiceCreateArguments](),
		ValidateArguments: func(value any) error {
			args := value.(*_PortalSiteServiceCreateArguments)
			if err := (&args.Creation).Validate(rpcspec.JoinPath("arguments", "Creation")); err != nil {
				return err
			}
			return nil
		},
		ResultType: reflect.TypeFor[PortalSite](),
		ValidateResult: func(value any) error {
			ret := value.(PortalSite)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalSiteServiceClient.Create,
			PortalSiteServiceClientER.Create,
			PortalSiteServiceServer.Create,
			PortalSiteServiceServerER.Create,
		},
	}
	_PortalSiteServiceUpdateSpec = &rpcspec.MethodSpec{
		Name:          "Update",
		SkelName:      "update",
		ArgumentsType: reflect.TypeFor[_PortalSiteServiceUpdateArguments](),
		ValidateArguments: func(value any) error {
			args := value.(*_PortalSiteServiceUpdateArguments)
			if err := (&args.Update).Validate(rpcspec.JoinPath("arguments", "Update")); err != nil {
				return err
			}
			return nil
		},
		ResultType: reflect.TypeFor[PortalSite](),
		ValidateResult: func(value any) error {
			ret := value.(PortalSite)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalSiteServiceClient.Update,
			PortalSiteServiceClientER.Update,
			PortalSiteServiceServer.Update,
			PortalSiteServiceServerER.Update,
		},
	}
	_PortalSiteServiceRemoveSpec = &rpcspec.MethodSpec{
		Name:                        "Remove",
		SkelName:                    "remove",
		ArgumentsType:               reflect.TypeFor[_PortalSiteServiceRemoveArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			PortalSiteServiceClient.Remove,
			PortalSiteServiceClientER.Remove,
			PortalSiteServiceServer.Remove,
			PortalSiteServiceServerER.Remove,
		},
	}
)

// PortalSiteService / Arguments

type _PortalSiteServiceGetArguments struct {
	Id int `json:"id" arg:"0"`
}

type _PortalSiteServiceCreateArguments struct {
	Creation PortalSiteCreation `json:"creation" arg:"0"`
}

type _PortalSiteServiceUpdateArguments struct {
	Id     int              `json:"id" arg:"0"`
	Update PortalSiteUpdate `json:"update" arg:"1"`
}

type _PortalSiteServiceRemoveArguments struct {
	Id int `json:"id" arg:"0"`
}

// PortalSiteService / Server

type PortalSiteServiceServer interface {
	// List List Portal target sites。
	//   @returns []PortalSite - Portal target site list
	List() []PortalSite
	// ListOptions List Portal target site form options。
	//   @returns PortalSiteOptions - Portal target site form options
	ListOptions() PortalSiteOptions
	// Get Read the Portal target site。
	//   @param id - Target site id
	//   @returns PortalSite - Portal target site
	Get(id int) PortalSite
	// Create Create Portal target site。
	//   @param creation - Portal target site creation parameters
	//   @returns PortalSite - Portal target site
	Create(creation PortalSiteCreation) PortalSite
	// Update Modify Portal target site。
	//   @param id - Target site id
	//   @param update - Portal target site update parameters
	//   @returns PortalSite - Portal target site
	Update(id int, update PortalSiteUpdate) PortalSite
	// Remove Delete Portal target site。
	//   @param id - Target site id
	Remove(id int)

	mustBePortalSiteServiceServer()
}

// PortalSiteService / Server / DefaultServer

type DefaultPortalSiteServiceServer struct{}

func (*DefaultPortalSiteServiceServer) List() []PortalSite {
	ex.PanicNew(ex.InvalidRequest, "method list is not implemented")
	return []PortalSite{}
}

func (*DefaultPortalSiteServiceServer) ListOptions() PortalSiteOptions {
	ex.PanicNew(ex.InvalidRequest, "method listOptions is not implemented")
	return PortalSiteOptions{}
}

func (*DefaultPortalSiteServiceServer) Get(int) PortalSite {
	ex.PanicNew(ex.InvalidRequest, "method get is not implemented")
	return PortalSite{}
}

func (*DefaultPortalSiteServiceServer) Create(PortalSiteCreation) PortalSite {
	ex.PanicNew(ex.InvalidRequest, "method create is not implemented")
	return PortalSite{}
}

func (*DefaultPortalSiteServiceServer) Update(int, PortalSiteUpdate) PortalSite {
	ex.PanicNew(ex.InvalidRequest, "method update is not implemented")
	return PortalSite{}
}

func (*DefaultPortalSiteServiceServer) Remove(int) {
	ex.PanicNew(ex.InvalidRequest, "method remove is not implemented")
}

func (*DefaultPortalSiteServiceServer) mustBePortalSiteServiceServer() {}

// PortalSiteService / ERServer

type PortalSiteServiceServerER interface {
	List() ([]PortalSite, ex.Error)
	ListOptions() (PortalSiteOptions, ex.Error)
	Get(id int) (PortalSite, ex.Error)
	Create(creation PortalSiteCreation) (PortalSite, ex.Error)
	Update(id int, update PortalSiteUpdate) (PortalSite, ex.Error)
	Remove(id int) ex.Error

	mustBePortalSiteServiceServerER()
}

// PortalSiteService / ERServer / WrapperERServer

type _WrapperPortalSiteServiceServerER struct {
	DefaultPortalSiteServiceServer
	serverImpl PortalSiteServiceServer
}

func _NewWrapperPortalSiteServiceServerER(serverImpl PortalSiteServiceServer) PortalSiteServiceServerER {
	return &_WrapperPortalSiteServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperPortalSiteServiceServerER) server() PortalSiteServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultPortalSiteServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperPortalSiteServiceServerER) List() (ret []PortalSite, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().List()
	return
}

func (service *_WrapperPortalSiteServiceServerER) ListOptions() (ret PortalSiteOptions, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListOptions()
	return
}

func (service *_WrapperPortalSiteServiceServerER) Get(id int) (ret PortalSite, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Get(id)
	return
}

func (service *_WrapperPortalSiteServiceServerER) Create(creation PortalSiteCreation) (ret PortalSite, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Create(creation)
	return
}

func (service *_WrapperPortalSiteServiceServerER) Update(id int, update PortalSiteUpdate) (ret PortalSite, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Update(id, update)
	return
}

func (service *_WrapperPortalSiteServiceServerER) Remove(id int) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().Remove(id)
	return
}

func (*_WrapperPortalSiteServiceServerER) mustBePortalSiteServiceServerER() {}

// PortalSiteService / ERServer / DefaultERServer

type DefaultPortalSiteServiceServerER struct {
	_WrapperPortalSiteServiceServerER
}

// PortalSiteService / Client

type PortalSiteServiceClient interface {
	// List List Portal target sites。
	//   @returns []PortalSite - Portal target site list
	List(_ivOpts ...rpcclient.InvokeOption) []PortalSite
	// ListOptions List Portal target site form options。
	//   @returns PortalSiteOptions - Portal target site form options
	ListOptions(_ivOpts ...rpcclient.InvokeOption) PortalSiteOptions
	// Get Read the Portal target site。
	//   @param id - Target site id
	//   @returns PortalSite - Portal target site
	Get(id int, _ivOpts ...rpcclient.InvokeOption) PortalSite
	// Create Create Portal target site。
	//   @param creation - Portal target site creation parameters
	//   @returns PortalSite - Portal target site
	Create(creation PortalSiteCreation, _ivOpts ...rpcclient.InvokeOption) PortalSite
	// Update Modify Portal target site。
	//   @param id - Target site id
	//   @param update - Portal target site update parameters
	//   @returns PortalSite - Portal target site
	Update(id int, update PortalSiteUpdate, _ivOpts ...rpcclient.InvokeOption) PortalSite
	// Remove Delete Portal target site。
	//   @param id - Target site id
	Remove(id int, _ivOpts ...rpcclient.InvokeOption)
}

type _PortalSiteServiceClient struct {
	clientER PortalSiteServiceClientER
}

func NewPortalSiteServiceClient(clientER PortalSiteServiceClientER) PortalSiteServiceClient {
	return &_PortalSiteServiceClient{clientER: clientER}
}

func (client *_PortalSiteServiceClient) List(_ivOpts ...rpcclient.InvokeOption) []PortalSite {
	ret, err := client.clientER.List(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalSiteServiceClient) ListOptions(_ivOpts ...rpcclient.InvokeOption) PortalSiteOptions {
	ret, err := client.clientER.ListOptions(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalSiteServiceClient) Get(id int, _ivOpts ...rpcclient.InvokeOption) PortalSite {
	ret, err := client.clientER.Get(id, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalSiteServiceClient) Create(creation PortalSiteCreation, _ivOpts ...rpcclient.InvokeOption) PortalSite {
	ret, err := client.clientER.Create(creation, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalSiteServiceClient) Update(id int, update PortalSiteUpdate, _ivOpts ...rpcclient.InvokeOption) PortalSite {
	ret, err := client.clientER.Update(id, update, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_PortalSiteServiceClient) Remove(id int, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.Remove(id, _ivOpts...)
	ex.PanicIfError(err)
}

// PortalSiteService / ERClient

type PortalSiteServiceClientER interface {
	// List List Portal target sites。
	//   @returns []PortalSite - Portal target site list
	List(_ivOpts ...rpcclient.InvokeOption) ([]PortalSite, ex.Error)
	// ListOptions List Portal target site form options。
	//   @returns PortalSiteOptions - Portal target site form options
	ListOptions(_ivOpts ...rpcclient.InvokeOption) (PortalSiteOptions, ex.Error)
	// Get Read the Portal target site。
	//   @param id - Target site id
	//   @returns PortalSite - Portal target site
	Get(id int, _ivOpts ...rpcclient.InvokeOption) (PortalSite, ex.Error)
	// Create Create Portal target site。
	//   @param creation - Portal target site creation parameters
	//   @returns PortalSite - Portal target site
	Create(creation PortalSiteCreation, _ivOpts ...rpcclient.InvokeOption) (PortalSite, ex.Error)
	// Update Modify Portal target site。
	//   @param id - Target site id
	//   @param update - Portal target site update parameters
	//   @returns PortalSite - Portal target site
	Update(id int, update PortalSiteUpdate, _ivOpts ...rpcclient.InvokeOption) (PortalSite, ex.Error)
	// Remove Delete Portal target site。
	//   @param id - Target site id
	Remove(id int, _ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _PortalSiteServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewPortalSiteServiceClientER(rpcClient *rpcclient.Client) PortalSiteServiceClientER {
	return &_PortalSiteServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_PortalSiteServiceClientER) List(_ivOpts ...rpcclient.InvokeOption) ([]PortalSite, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalSiteServiceListSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]PortalSite)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalSiteServiceClientER) ListOptions(_ivOpts ...rpcclient.InvokeOption) (PortalSiteOptions, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalSiteServiceListOptionsSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.(PortalSiteOptions)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalSiteServiceClientER) Get(id int, _ivOpts ...rpcclient.InvokeOption) (PortalSite, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalSiteServiceGetSpec.Info(), &_PortalSiteServiceGetArguments{
		Id: id,
	}, _ivOpts...)
	ret, _ := retI.(PortalSite)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalSiteServiceClientER) Create(creation PortalSiteCreation, _ivOpts ...rpcclient.InvokeOption) (PortalSite, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalSiteServiceCreateSpec.Info(), &_PortalSiteServiceCreateArguments{
		Creation: creation,
	}, _ivOpts...)
	ret, _ := retI.(PortalSite)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalSiteServiceClientER) Update(id int, update PortalSiteUpdate, _ivOpts ...rpcclient.InvokeOption) (PortalSite, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_PortalSiteServiceUpdateSpec.Info(), &_PortalSiteServiceUpdateArguments{
		Id:     id,
		Update: update,
	}, _ivOpts...)
	ret, _ := retI.(PortalSite)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_PortalSiteServiceClientER) Remove(id int, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_PortalSiteServiceRemoveSpec.Info(), &_PortalSiteServiceRemoveArguments{
		Id: id,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

// RegistryServiceServer Hub's application registration service, called by Link

// RegistryService / Spec

var (
	_RegistryServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "RegistryService",
		SkelName:          "vine.hub.RegistryService",
		Hash:              "b8e96f22",
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
			_RegistryServiceHeartbeatSpec,
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
		ArgumentsType:               reflect.TypeFor[_RegistryServiceUnregisterArguments](),
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
	_RegistryServiceHeartbeatSpec = &rpcspec.MethodSpec{
		Name:                        "Heartbeat",
		SkelName:                    "heartbeat",
		ArgumentsType:               reflect.TypeFor[_RegistryServiceHeartbeatArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[bool](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			RegistryServiceClient.Heartbeat,
			RegistryServiceClientER.Heartbeat,
			RegistryServiceServer.Heartbeat,
			RegistryServiceServerER.Heartbeat,
		},
	}
)

// RegistryService / Arguments

type _RegistryServiceRegisterArguments struct {
	Registration AppRegistration `json:"registration" arg:"0"`
}

type _RegistryServiceUnregisterArguments struct {
	Name       string    `json:"name" arg:"0"`
	InstanceId skel.UUID `json:"instanceId" arg:"1"`
}

type _RegistryServiceHeartbeatArguments struct {
	Status AppStatus `json:"status" arg:"0"`
}

// RegistryService / Server

type RegistryServiceServer interface {
	// Register Register application instance。
	//   @param registration - Application instance registration information
	Register(registration AppRegistration)
	// Unregister Unregister an application instance。
	//   @param name - Application name
	//   @param instanceId - Application instance ID
	Unregister(name string, instanceId skel.UUID)
	// Heartbeat Application instance heartbeat。
	//   @param status - Application instance ID
	//   @returns bool - Whether the current instance is still registered in the Hub
	Heartbeat(status AppStatus) bool

	mustBeRegistryServiceServer()
}

// RegistryService / Server / DefaultServer

type DefaultRegistryServiceServer struct{}

func (*DefaultRegistryServiceServer) Register(AppRegistration) {
	ex.PanicNew(ex.InvalidRequest, "method register is not implemented")
}

func (*DefaultRegistryServiceServer) Unregister(string, skel.UUID) {
	ex.PanicNew(ex.InvalidRequest, "method unregister is not implemented")
}

func (*DefaultRegistryServiceServer) Heartbeat(AppStatus) bool {
	ex.PanicNew(ex.InvalidRequest, "method heartbeat is not implemented")
	return false
}

func (*DefaultRegistryServiceServer) mustBeRegistryServiceServer() {}

// RegistryService / ERServer

type RegistryServiceServerER interface {
	Register(registration AppRegistration) ex.Error
	Unregister(name string, instanceId skel.UUID) ex.Error
	Heartbeat(status AppStatus) (bool, ex.Error)

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

func (service *_WrapperRegistryServiceServerER) Unregister(name string, instanceId skel.UUID) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().Unregister(name, instanceId)
	return
}

func (service *_WrapperRegistryServiceServerER) Heartbeat(status AppStatus) (ret bool, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Heartbeat(status)
	return
}

func (*_WrapperRegistryServiceServerER) mustBeRegistryServiceServerER() {}

// RegistryService / ERServer / DefaultERServer

type DefaultRegistryServiceServerER struct {
	_WrapperRegistryServiceServerER
}

// RegistryService / Client

type RegistryServiceClient interface {
	// Register Register application instance。
	//   @param registration - Application instance registration information
	Register(registration AppRegistration, _ivOpts ...rpcclient.InvokeOption)
	// Unregister Unregister an application instance。
	//   @param name - Application name
	//   @param instanceId - Application instance ID
	Unregister(name string, instanceId skel.UUID, _ivOpts ...rpcclient.InvokeOption)
	// Heartbeat Application instance heartbeat。
	//   @param status - Application instance ID
	//   @returns bool - Whether the current instance is still registered in the Hub
	Heartbeat(status AppStatus, _ivOpts ...rpcclient.InvokeOption) bool
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

func (client *_RegistryServiceClient) Unregister(name string, instanceId skel.UUID, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.Unregister(name, instanceId, _ivOpts...)
	ex.PanicIfError(err)
}

func (client *_RegistryServiceClient) Heartbeat(status AppStatus, _ivOpts ...rpcclient.InvokeOption) bool {
	ret, err := client.clientER.Heartbeat(status, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// RegistryService / ERClient

type RegistryServiceClientER interface {
	// Register Register application instance。
	//   @param registration - Application instance registration information
	Register(registration AppRegistration, _ivOpts ...rpcclient.InvokeOption) ex.Error
	// Unregister Unregister an application instance。
	//   @param name - Application name
	//   @param instanceId - Application instance ID
	Unregister(name string, instanceId skel.UUID, _ivOpts ...rpcclient.InvokeOption) ex.Error
	// Heartbeat Application instance heartbeat。
	//   @param status - Application instance ID
	//   @returns bool - Whether the current instance is still registered in the Hub
	Heartbeat(status AppStatus, _ivOpts ...rpcclient.InvokeOption) (bool, ex.Error)
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

func (client *_RegistryServiceClientER) Unregister(name string, instanceId skel.UUID, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_RegistryServiceUnregisterSpec.Info(), &_RegistryServiceUnregisterArguments{
		Name:       name,
		InstanceId: instanceId,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}

func (client *_RegistryServiceClientER) Heartbeat(status AppStatus, _ivOpts ...rpcclient.InvokeOption) (bool, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_RegistryServiceHeartbeatSpec.Info(), &_RegistryServiceHeartbeatArguments{
		Status: status,
	}, _ivOpts...)
	ret, _ := retI.(bool)
	err, _ := errI.(ex.Error)
	return ret, err
}

// ServiceDebugServiceServer Hub Dashboard Service debugging service

// ServiceDebugService / Spec

var (
	_ServiceDebugServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "ServiceDebugService",
		SkelName:          "vine.hub.ServiceDebugService",
		Hash:              "fc747d85",
		ServerType:        reflect.TypeFor[ServiceDebugServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultServiceDebugServiceServer](),
		ClientType:        reflect.TypeFor[ServiceDebugServiceClient](),
		ClientCtor:        NewServiceDebugServiceClient,

		ERServerType:        reflect.TypeFor[ServiceDebugServiceServerER](),
		WrapperERServerCtor: _NewWrapperServiceDebugServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultServiceDebugServiceServerER](),
		ERClientType:        reflect.TypeFor[ServiceDebugServiceClientER](),
		ERClientCtor:        NewServiceDebugServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_ServiceDebugServiceListAppInstancesSpec,
			_ServiceDebugServiceListServicesSpec,
			_ServiceDebugServiceListServiceAppInstancesSpec,
			_ServiceDebugServiceListMethodsSpec,
			_ServiceDebugServiceBuildDefaultInvokeRequestSpec,
			_ServiceDebugServiceInvokeServiceSpec,
		},
	}
	_ServiceDebugServiceListAppInstancesSpec = &rpcspec.MethodSpec{
		Name:              "ListAppInstances",
		SkelName:          "listAppInstances",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]ServiceDebugAppInstance](),
		ValidateResult: func(value any) error {
			ret := value.([]ServiceDebugAppInstance)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ServiceDebugServiceClient.ListAppInstances,
			ServiceDebugServiceClientER.ListAppInstances,
			ServiceDebugServiceServer.ListAppInstances,
			ServiceDebugServiceServerER.ListAppInstances,
		},
	}
	_ServiceDebugServiceListServicesSpec = &rpcspec.MethodSpec{
		Name:              "ListServices",
		SkelName:          "listServices",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]ServiceDebugServiceItem](),
		ValidateResult: func(value any) error {
			ret := value.([]ServiceDebugServiceItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ServiceDebugServiceClient.ListServices,
			ServiceDebugServiceClientER.ListServices,
			ServiceDebugServiceServer.ListServices,
			ServiceDebugServiceServerER.ListServices,
		},
	}
	_ServiceDebugServiceListServiceAppInstancesSpec = &rpcspec.MethodSpec{
		Name:              "ListServiceAppInstances",
		SkelName:          "listServiceAppInstances",
		ArgumentsType:     reflect.TypeFor[_ServiceDebugServiceListServiceAppInstancesArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]ServiceDebugAppInstance](),
		ValidateResult: func(value any) error {
			ret := value.([]ServiceDebugAppInstance)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ServiceDebugServiceClient.ListServiceAppInstances,
			ServiceDebugServiceClientER.ListServiceAppInstances,
			ServiceDebugServiceServer.ListServiceAppInstances,
			ServiceDebugServiceServerER.ListServiceAppInstances,
		},
	}
	_ServiceDebugServiceListMethodsSpec = &rpcspec.MethodSpec{
		Name:              "ListMethods",
		SkelName:          "listMethods",
		ArgumentsType:     reflect.TypeFor[_ServiceDebugServiceListMethodsArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]ServiceDebugMethodItem](),
		ValidateResult: func(value any) error {
			ret := value.([]ServiceDebugMethodItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ServiceDebugServiceClient.ListMethods,
			ServiceDebugServiceClientER.ListMethods,
			ServiceDebugServiceServer.ListMethods,
			ServiceDebugServiceServerER.ListMethods,
		},
	}
	_ServiceDebugServiceBuildDefaultInvokeRequestSpec = &rpcspec.MethodSpec{
		Name:              "BuildDefaultInvokeRequest",
		SkelName:          "buildDefaultInvokeRequest",
		ArgumentsType:     reflect.TypeFor[_ServiceDebugServiceBuildDefaultInvokeRequestArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[ServiceDebugDefaultInvokeRequest](),
		ValidateResult: func(value any) error {
			ret := value.(ServiceDebugDefaultInvokeRequest)
			if err := (&ret).Validate("result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ServiceDebugServiceClient.BuildDefaultInvokeRequest,
			ServiceDebugServiceClientER.BuildDefaultInvokeRequest,
			ServiceDebugServiceServer.BuildDefaultInvokeRequest,
			ServiceDebugServiceServerER.BuildDefaultInvokeRequest,
		},
	}
	_ServiceDebugServiceInvokeServiceSpec = &rpcspec.MethodSpec{
		Name:                        "InvokeService",
		SkelName:                    "invokeService",
		ArgumentsType:               reflect.TypeFor[_ServiceDebugServiceInvokeServiceArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[ServiceDebugInvokeResponse](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			ServiceDebugServiceClient.InvokeService,
			ServiceDebugServiceClientER.InvokeService,
			ServiceDebugServiceServer.InvokeService,
			ServiceDebugServiceServerER.InvokeService,
		},
	}
)

// ServiceDebugService / Arguments

type _ServiceDebugServiceListServiceAppInstancesArguments struct {
	ServiceSkelName string `json:"serviceSkelName" arg:"0"`
	SchemaHash      string `json:"schemaHash" arg:"1"`
}

type _ServiceDebugServiceListMethodsArguments struct {
	ServiceSkelName string `json:"serviceSkelName" arg:"0"`
	SchemaHash      string `json:"schemaHash" arg:"1"`
}

type _ServiceDebugServiceBuildDefaultInvokeRequestArguments struct {
	ServiceSkelName string `json:"serviceSkelName" arg:"0"`
	SchemaHash      string `json:"schemaHash" arg:"1"`
	MethodSkelName  string `json:"methodSkelName" arg:"2"`
}

type _ServiceDebugServiceInvokeServiceArguments struct {
	Request ServiceDebugInvokeRequest `json:"request" arg:"0"`
}

// ServiceDebugService / Server

type ServiceDebugServiceServer interface {
	// ListAppInstances List application instances。
	ListAppInstances() []ServiceDebugAppInstance
	// ListServices List the services provided by the application instance。
	ListServices() []ServiceDebugServiceItem
	// ListServiceAppInstances List application instances that provide the specified service。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	ListServiceAppInstances(serviceSkelName string, schemaHash string) []ServiceDebugAppInstance
	// ListMethods List Service methods。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	ListMethods(serviceSkelName string, schemaHash string) []ServiceDebugMethodItem
	// BuildDefaultInvokeRequest Generate default Service call request。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	//   @param methodSkelName - Method Skel name
	BuildDefaultInvokeRequest(serviceSkelName string, schemaHash string, methodSkelName string) ServiceDebugDefaultInvokeRequest
	// InvokeService Call Service method。
	//   @param request - Debug call request
	InvokeService(request ServiceDebugInvokeRequest) ServiceDebugInvokeResponse

	mustBeServiceDebugServiceServer()
}

// ServiceDebugService / Server / DefaultServer

type DefaultServiceDebugServiceServer struct{}

func (*DefaultServiceDebugServiceServer) ListAppInstances() []ServiceDebugAppInstance {
	ex.PanicNew(ex.InvalidRequest, "method listAppInstances is not implemented")
	return []ServiceDebugAppInstance{}
}

func (*DefaultServiceDebugServiceServer) ListServices() []ServiceDebugServiceItem {
	ex.PanicNew(ex.InvalidRequest, "method listServices is not implemented")
	return []ServiceDebugServiceItem{}
}

func (*DefaultServiceDebugServiceServer) ListServiceAppInstances(string, string) []ServiceDebugAppInstance {
	ex.PanicNew(ex.InvalidRequest, "method listServiceAppInstances is not implemented")
	return []ServiceDebugAppInstance{}
}

func (*DefaultServiceDebugServiceServer) ListMethods(string, string) []ServiceDebugMethodItem {
	ex.PanicNew(ex.InvalidRequest, "method listMethods is not implemented")
	return []ServiceDebugMethodItem{}
}

func (*DefaultServiceDebugServiceServer) BuildDefaultInvokeRequest(string, string, string) ServiceDebugDefaultInvokeRequest {
	ex.PanicNew(ex.InvalidRequest, "method buildDefaultInvokeRequest is not implemented")
	return ServiceDebugDefaultInvokeRequest{}
}

func (*DefaultServiceDebugServiceServer) InvokeService(ServiceDebugInvokeRequest) ServiceDebugInvokeResponse {
	ex.PanicNew(ex.InvalidRequest, "method invokeService is not implemented")
	return ServiceDebugInvokeResponse{}
}

func (*DefaultServiceDebugServiceServer) mustBeServiceDebugServiceServer() {}

// ServiceDebugService / ERServer

type ServiceDebugServiceServerER interface {
	ListAppInstances() ([]ServiceDebugAppInstance, ex.Error)
	ListServices() ([]ServiceDebugServiceItem, ex.Error)
	ListServiceAppInstances(serviceSkelName string, schemaHash string) ([]ServiceDebugAppInstance, ex.Error)
	ListMethods(serviceSkelName string, schemaHash string) ([]ServiceDebugMethodItem, ex.Error)
	BuildDefaultInvokeRequest(serviceSkelName string, schemaHash string, methodSkelName string) (ServiceDebugDefaultInvokeRequest, ex.Error)
	InvokeService(request ServiceDebugInvokeRequest) (ServiceDebugInvokeResponse, ex.Error)

	mustBeServiceDebugServiceServerER()
}

// ServiceDebugService / ERServer / WrapperERServer

type _WrapperServiceDebugServiceServerER struct {
	DefaultServiceDebugServiceServer
	serverImpl ServiceDebugServiceServer
}

func _NewWrapperServiceDebugServiceServerER(serverImpl ServiceDebugServiceServer) ServiceDebugServiceServerER {
	return &_WrapperServiceDebugServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperServiceDebugServiceServerER) server() ServiceDebugServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultServiceDebugServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperServiceDebugServiceServerER) ListAppInstances() (ret []ServiceDebugAppInstance, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListAppInstances()
	return
}

func (service *_WrapperServiceDebugServiceServerER) ListServices() (ret []ServiceDebugServiceItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListServices()
	return
}

func (service *_WrapperServiceDebugServiceServerER) ListServiceAppInstances(serviceSkelName string, schemaHash string) (ret []ServiceDebugAppInstance, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListServiceAppInstances(serviceSkelName, schemaHash)
	return
}

func (service *_WrapperServiceDebugServiceServerER) ListMethods(serviceSkelName string, schemaHash string) (ret []ServiceDebugMethodItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListMethods(serviceSkelName, schemaHash)
	return
}

func (service *_WrapperServiceDebugServiceServerER) BuildDefaultInvokeRequest(serviceSkelName string, schemaHash string, methodSkelName string) (ret ServiceDebugDefaultInvokeRequest, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().BuildDefaultInvokeRequest(serviceSkelName, schemaHash, methodSkelName)
	return
}

func (service *_WrapperServiceDebugServiceServerER) InvokeService(request ServiceDebugInvokeRequest) (ret ServiceDebugInvokeResponse, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().InvokeService(request)
	return
}

func (*_WrapperServiceDebugServiceServerER) mustBeServiceDebugServiceServerER() {}

// ServiceDebugService / ERServer / DefaultERServer

type DefaultServiceDebugServiceServerER struct {
	_WrapperServiceDebugServiceServerER
}

// ServiceDebugService / Client

type ServiceDebugServiceClient interface {
	// ListAppInstances List application instances。
	ListAppInstances(_ivOpts ...rpcclient.InvokeOption) []ServiceDebugAppInstance
	// ListServices List the services provided by the application instance。
	ListServices(_ivOpts ...rpcclient.InvokeOption) []ServiceDebugServiceItem
	// ListServiceAppInstances List application instances that provide the specified service。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	ListServiceAppInstances(serviceSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) []ServiceDebugAppInstance
	// ListMethods List Service methods。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	ListMethods(serviceSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) []ServiceDebugMethodItem
	// BuildDefaultInvokeRequest Generate default Service call request。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	//   @param methodSkelName - Method Skel name
	BuildDefaultInvokeRequest(serviceSkelName string, schemaHash string, methodSkelName string, _ivOpts ...rpcclient.InvokeOption) ServiceDebugDefaultInvokeRequest
	// InvokeService Call Service method。
	//   @param request - Debug call request
	InvokeService(request ServiceDebugInvokeRequest, _ivOpts ...rpcclient.InvokeOption) ServiceDebugInvokeResponse
}

type _ServiceDebugServiceClient struct {
	clientER ServiceDebugServiceClientER
}

func NewServiceDebugServiceClient(clientER ServiceDebugServiceClientER) ServiceDebugServiceClient {
	return &_ServiceDebugServiceClient{clientER: clientER}
}

func (client *_ServiceDebugServiceClient) ListAppInstances(_ivOpts ...rpcclient.InvokeOption) []ServiceDebugAppInstance {
	ret, err := client.clientER.ListAppInstances(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_ServiceDebugServiceClient) ListServices(_ivOpts ...rpcclient.InvokeOption) []ServiceDebugServiceItem {
	ret, err := client.clientER.ListServices(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_ServiceDebugServiceClient) ListServiceAppInstances(serviceSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) []ServiceDebugAppInstance {
	ret, err := client.clientER.ListServiceAppInstances(serviceSkelName, schemaHash, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_ServiceDebugServiceClient) ListMethods(serviceSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) []ServiceDebugMethodItem {
	ret, err := client.clientER.ListMethods(serviceSkelName, schemaHash, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_ServiceDebugServiceClient) BuildDefaultInvokeRequest(serviceSkelName string, schemaHash string, methodSkelName string, _ivOpts ...rpcclient.InvokeOption) ServiceDebugDefaultInvokeRequest {
	ret, err := client.clientER.BuildDefaultInvokeRequest(serviceSkelName, schemaHash, methodSkelName, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_ServiceDebugServiceClient) InvokeService(request ServiceDebugInvokeRequest, _ivOpts ...rpcclient.InvokeOption) ServiceDebugInvokeResponse {
	ret, err := client.clientER.InvokeService(request, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// ServiceDebugService / ERClient

type ServiceDebugServiceClientER interface {
	// ListAppInstances List application instances。
	ListAppInstances(_ivOpts ...rpcclient.InvokeOption) ([]ServiceDebugAppInstance, ex.Error)
	// ListServices List the services provided by the application instance。
	ListServices(_ivOpts ...rpcclient.InvokeOption) ([]ServiceDebugServiceItem, ex.Error)
	// ListServiceAppInstances List application instances that provide the specified service。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	ListServiceAppInstances(serviceSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) ([]ServiceDebugAppInstance, ex.Error)
	// ListMethods List Service methods。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	ListMethods(serviceSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) ([]ServiceDebugMethodItem, ex.Error)
	// BuildDefaultInvokeRequest Generate default Service call request。
	//   @param serviceSkelName - Service Skel name
	//   @param schemaHash - Service schema hash
	//   @param methodSkelName - Method Skel name
	BuildDefaultInvokeRequest(serviceSkelName string, schemaHash string, methodSkelName string, _ivOpts ...rpcclient.InvokeOption) (ServiceDebugDefaultInvokeRequest, ex.Error)
	// InvokeService Call Service method。
	//   @param request - Debug call request
	InvokeService(request ServiceDebugInvokeRequest, _ivOpts ...rpcclient.InvokeOption) (ServiceDebugInvokeResponse, ex.Error)
}

type _ServiceDebugServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewServiceDebugServiceClientER(rpcClient *rpcclient.Client) ServiceDebugServiceClientER {
	return &_ServiceDebugServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_ServiceDebugServiceClientER) ListAppInstances(_ivOpts ...rpcclient.InvokeOption) ([]ServiceDebugAppInstance, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_ServiceDebugServiceListAppInstancesSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]ServiceDebugAppInstance)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_ServiceDebugServiceClientER) ListServices(_ivOpts ...rpcclient.InvokeOption) ([]ServiceDebugServiceItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_ServiceDebugServiceListServicesSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]ServiceDebugServiceItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_ServiceDebugServiceClientER) ListServiceAppInstances(serviceSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) ([]ServiceDebugAppInstance, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_ServiceDebugServiceListServiceAppInstancesSpec.Info(), &_ServiceDebugServiceListServiceAppInstancesArguments{
		ServiceSkelName: serviceSkelName,
		SchemaHash:      schemaHash,
	}, _ivOpts...)
	ret, _ := retI.([]ServiceDebugAppInstance)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_ServiceDebugServiceClientER) ListMethods(serviceSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) ([]ServiceDebugMethodItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_ServiceDebugServiceListMethodsSpec.Info(), &_ServiceDebugServiceListMethodsArguments{
		ServiceSkelName: serviceSkelName,
		SchemaHash:      schemaHash,
	}, _ivOpts...)
	ret, _ := retI.([]ServiceDebugMethodItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_ServiceDebugServiceClientER) BuildDefaultInvokeRequest(serviceSkelName string, schemaHash string, methodSkelName string, _ivOpts ...rpcclient.InvokeOption) (ServiceDebugDefaultInvokeRequest, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_ServiceDebugServiceBuildDefaultInvokeRequestSpec.Info(), &_ServiceDebugServiceBuildDefaultInvokeRequestArguments{
		ServiceSkelName: serviceSkelName,
		SchemaHash:      schemaHash,
		MethodSkelName:  methodSkelName,
	}, _ivOpts...)
	ret, _ := retI.(ServiceDebugDefaultInvokeRequest)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_ServiceDebugServiceClientER) InvokeService(request ServiceDebugInvokeRequest, _ivOpts ...rpcclient.InvokeOption) (ServiceDebugInvokeResponse, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_ServiceDebugServiceInvokeServiceSpec.Info(), &_ServiceDebugServiceInvokeServiceArguments{
		Request: request,
	}, _ivOpts...)
	ret, _ := retI.(ServiceDebugInvokeResponse)
	err, _ := errI.(ex.Error)
	return ret, err
}

// SkeletonServiceServer Hub's skeleton service, called by the Portal management client

// SkeletonService / Spec

var (
	_SkeletonServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "SkeletonService",
		SkelName:          "vine.hub.SkeletonService",
		Hash:              "6d2dd869",
		ServerType:        reflect.TypeFor[SkeletonServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultSkeletonServiceServer](),
		ClientType:        reflect.TypeFor[SkeletonServiceClient](),
		ClientCtor:        NewSkeletonServiceClient,

		ERServerType:        reflect.TypeFor[SkeletonServiceServerER](),
		WrapperERServerCtor: _NewWrapperSkeletonServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultSkeletonServiceServerER](),
		ERClientType:        reflect.TypeFor[SkeletonServiceClientER](),
		ERClientCtor:        NewSkeletonServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_SkeletonServiceListDomainsSpec,
			_SkeletonServiceListActorsSpec,
			_SkeletonServiceListServicesSpec,
			_SkeletonServiceListResourcesSpec,
			_SkeletonServiceListWebsSpec,
			_SkeletonServiceListTasksSpec,
			_SkeletonServiceListEventsSpec,
			_SkeletonServiceListDataSpec,
			_SkeletonServiceListConfigsSpec,
		},
	}
	_SkeletonServiceListDomainsSpec = &rpcspec.MethodSpec{
		Name:              "ListDomains",
		SkelName:          "listDomains",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonDomain](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonDomain)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListDomains,
			SkeletonServiceClientER.ListDomains,
			SkeletonServiceServer.ListDomains,
			SkeletonServiceServerER.ListDomains,
		},
	}
	_SkeletonServiceListActorsSpec = &rpcspec.MethodSpec{
		Name:              "ListActors",
		SkelName:          "listActors",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonActorItem](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonActorItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListActors,
			SkeletonServiceClientER.ListActors,
			SkeletonServiceServer.ListActors,
			SkeletonServiceServerER.ListActors,
		},
	}
	_SkeletonServiceListServicesSpec = &rpcspec.MethodSpec{
		Name:              "ListServices",
		SkelName:          "listServices",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonServiceItem](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonServiceItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListServices,
			SkeletonServiceClientER.ListServices,
			SkeletonServiceServer.ListServices,
			SkeletonServiceServerER.ListServices,
		},
	}
	_SkeletonServiceListResourcesSpec = &rpcspec.MethodSpec{
		Name:              "ListResources",
		SkelName:          "listResources",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonResourceItem](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonResourceItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListResources,
			SkeletonServiceClientER.ListResources,
			SkeletonServiceServer.ListResources,
			SkeletonServiceServerER.ListResources,
		},
	}
	_SkeletonServiceListWebsSpec = &rpcspec.MethodSpec{
		Name:              "ListWebs",
		SkelName:          "listWebs",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonWebItem](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonWebItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListWebs,
			SkeletonServiceClientER.ListWebs,
			SkeletonServiceServer.ListWebs,
			SkeletonServiceServerER.ListWebs,
		},
	}
	_SkeletonServiceListTasksSpec = &rpcspec.MethodSpec{
		Name:              "ListTasks",
		SkelName:          "listTasks",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonTask](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonTask)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListTasks,
			SkeletonServiceClientER.ListTasks,
			SkeletonServiceServer.ListTasks,
			SkeletonServiceServerER.ListTasks,
		},
	}
	_SkeletonServiceListEventsSpec = &rpcspec.MethodSpec{
		Name:              "ListEvents",
		SkelName:          "listEvents",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonEventItem](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonEventItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListEvents,
			SkeletonServiceClientER.ListEvents,
			SkeletonServiceServer.ListEvents,
			SkeletonServiceServerER.ListEvents,
		},
	}
	_SkeletonServiceListDataSpec = &rpcspec.MethodSpec{
		Name:              "ListData",
		SkelName:          "listData",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonData](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonData)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListData,
			SkeletonServiceClientER.ListData,
			SkeletonServiceServer.ListData,
			SkeletonServiceServerER.ListData,
		},
	}
	_SkeletonServiceListConfigsSpec = &rpcspec.MethodSpec{
		Name:              "ListConfigs",
		SkelName:          "listConfigs",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]SkeletonConfigItem](),
		ValidateResult: func(value any) error {
			ret := value.([]SkeletonConfigItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			SkeletonServiceClient.ListConfigs,
			SkeletonServiceClientER.ListConfigs,
			SkeletonServiceServer.ListConfigs,
			SkeletonServiceServerER.ListConfigs,
		},
	}
)

// SkeletonService / Server

type SkeletonServiceServer interface {
	// ListDomains List Domain skeleton。
	//   @returns []SkeletonDomain - Domain skeleton list
	ListDomains() []SkeletonDomain
	// ListActors List Actor Skeleton。
	//   @returns []SkeletonActorItem - Actor skeleton list
	ListActors() []SkeletonActorItem
	// ListServices List Service skeleton。
	//   @returns []SkeletonServiceItem - Service skeleton list
	ListServices() []SkeletonServiceItem
	// ListResources List Resource skeleton。
	//   @returns []SkeletonResourceItem - Resource skeleton list
	ListResources() []SkeletonResourceItem
	// ListWebs List Web Skeletons。
	//   @returns []SkeletonWebItem - Web skeleton list
	ListWebs() []SkeletonWebItem
	// ListTasks List Task skeleton。
	//   @returns []SkeletonTask - Task skeleton list
	ListTasks() []SkeletonTask
	// ListEvents List Event skeletons。
	//   @returns []SkeletonEventItem - Event skeleton list
	ListEvents() []SkeletonEventItem
	// ListData List Data skeleton。
	//   @returns []SkeletonData - Data skeleton list, including Enum
	ListData() []SkeletonData
	// ListConfigs List Config skeleton。
	//   @returns []SkeletonConfigItem - Config skeleton list
	ListConfigs() []SkeletonConfigItem

	mustBeSkeletonServiceServer()
}

// SkeletonService / Server / DefaultServer

type DefaultSkeletonServiceServer struct{}

func (*DefaultSkeletonServiceServer) ListDomains() []SkeletonDomain {
	ex.PanicNew(ex.InvalidRequest, "method listDomains is not implemented")
	return []SkeletonDomain{}
}

func (*DefaultSkeletonServiceServer) ListActors() []SkeletonActorItem {
	ex.PanicNew(ex.InvalidRequest, "method listActors is not implemented")
	return []SkeletonActorItem{}
}

func (*DefaultSkeletonServiceServer) ListServices() []SkeletonServiceItem {
	ex.PanicNew(ex.InvalidRequest, "method listServices is not implemented")
	return []SkeletonServiceItem{}
}

func (*DefaultSkeletonServiceServer) ListResources() []SkeletonResourceItem {
	ex.PanicNew(ex.InvalidRequest, "method listResources is not implemented")
	return []SkeletonResourceItem{}
}

func (*DefaultSkeletonServiceServer) ListWebs() []SkeletonWebItem {
	ex.PanicNew(ex.InvalidRequest, "method listWebs is not implemented")
	return []SkeletonWebItem{}
}

func (*DefaultSkeletonServiceServer) ListTasks() []SkeletonTask {
	ex.PanicNew(ex.InvalidRequest, "method listTasks is not implemented")
	return []SkeletonTask{}
}

func (*DefaultSkeletonServiceServer) ListEvents() []SkeletonEventItem {
	ex.PanicNew(ex.InvalidRequest, "method listEvents is not implemented")
	return []SkeletonEventItem{}
}

func (*DefaultSkeletonServiceServer) ListData() []SkeletonData {
	ex.PanicNew(ex.InvalidRequest, "method listData is not implemented")
	return []SkeletonData{}
}

func (*DefaultSkeletonServiceServer) ListConfigs() []SkeletonConfigItem {
	ex.PanicNew(ex.InvalidRequest, "method listConfigs is not implemented")
	return []SkeletonConfigItem{}
}

func (*DefaultSkeletonServiceServer) mustBeSkeletonServiceServer() {}

// SkeletonService / ERServer

type SkeletonServiceServerER interface {
	ListDomains() ([]SkeletonDomain, ex.Error)
	ListActors() ([]SkeletonActorItem, ex.Error)
	ListServices() ([]SkeletonServiceItem, ex.Error)
	ListResources() ([]SkeletonResourceItem, ex.Error)
	ListWebs() ([]SkeletonWebItem, ex.Error)
	ListTasks() ([]SkeletonTask, ex.Error)
	ListEvents() ([]SkeletonEventItem, ex.Error)
	ListData() ([]SkeletonData, ex.Error)
	ListConfigs() ([]SkeletonConfigItem, ex.Error)

	mustBeSkeletonServiceServerER()
}

// SkeletonService / ERServer / WrapperERServer

type _WrapperSkeletonServiceServerER struct {
	DefaultSkeletonServiceServer
	serverImpl SkeletonServiceServer
}

func _NewWrapperSkeletonServiceServerER(serverImpl SkeletonServiceServer) SkeletonServiceServerER {
	return &_WrapperSkeletonServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperSkeletonServiceServerER) server() SkeletonServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultSkeletonServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperSkeletonServiceServerER) ListDomains() (ret []SkeletonDomain, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListDomains()
	return
}

func (service *_WrapperSkeletonServiceServerER) ListActors() (ret []SkeletonActorItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListActors()
	return
}

func (service *_WrapperSkeletonServiceServerER) ListServices() (ret []SkeletonServiceItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListServices()
	return
}

func (service *_WrapperSkeletonServiceServerER) ListResources() (ret []SkeletonResourceItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListResources()
	return
}

func (service *_WrapperSkeletonServiceServerER) ListWebs() (ret []SkeletonWebItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListWebs()
	return
}

func (service *_WrapperSkeletonServiceServerER) ListTasks() (ret []SkeletonTask, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListTasks()
	return
}

func (service *_WrapperSkeletonServiceServerER) ListEvents() (ret []SkeletonEventItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListEvents()
	return
}

func (service *_WrapperSkeletonServiceServerER) ListData() (ret []SkeletonData, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListData()
	return
}

func (service *_WrapperSkeletonServiceServerER) ListConfigs() (ret []SkeletonConfigItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListConfigs()
	return
}

func (*_WrapperSkeletonServiceServerER) mustBeSkeletonServiceServerER() {}

// SkeletonService / ERServer / DefaultERServer

type DefaultSkeletonServiceServerER struct {
	_WrapperSkeletonServiceServerER
}

// SkeletonService / Client

type SkeletonServiceClient interface {
	// ListDomains List Domain skeleton。
	//   @returns []SkeletonDomain - Domain skeleton list
	ListDomains(_ivOpts ...rpcclient.InvokeOption) []SkeletonDomain
	// ListActors List Actor Skeleton。
	//   @returns []SkeletonActorItem - Actor skeleton list
	ListActors(_ivOpts ...rpcclient.InvokeOption) []SkeletonActorItem
	// ListServices List Service skeleton。
	//   @returns []SkeletonServiceItem - Service skeleton list
	ListServices(_ivOpts ...rpcclient.InvokeOption) []SkeletonServiceItem
	// ListResources List Resource skeleton。
	//   @returns []SkeletonResourceItem - Resource skeleton list
	ListResources(_ivOpts ...rpcclient.InvokeOption) []SkeletonResourceItem
	// ListWebs List Web Skeletons。
	//   @returns []SkeletonWebItem - Web skeleton list
	ListWebs(_ivOpts ...rpcclient.InvokeOption) []SkeletonWebItem
	// ListTasks List Task skeleton。
	//   @returns []SkeletonTask - Task skeleton list
	ListTasks(_ivOpts ...rpcclient.InvokeOption) []SkeletonTask
	// ListEvents List Event skeletons。
	//   @returns []SkeletonEventItem - Event skeleton list
	ListEvents(_ivOpts ...rpcclient.InvokeOption) []SkeletonEventItem
	// ListData List Data skeleton。
	//   @returns []SkeletonData - Data skeleton list, including Enum
	ListData(_ivOpts ...rpcclient.InvokeOption) []SkeletonData
	// ListConfigs List Config skeleton。
	//   @returns []SkeletonConfigItem - Config skeleton list
	ListConfigs(_ivOpts ...rpcclient.InvokeOption) []SkeletonConfigItem
}

type _SkeletonServiceClient struct {
	clientER SkeletonServiceClientER
}

func NewSkeletonServiceClient(clientER SkeletonServiceClientER) SkeletonServiceClient {
	return &_SkeletonServiceClient{clientER: clientER}
}

func (client *_SkeletonServiceClient) ListDomains(_ivOpts ...rpcclient.InvokeOption) []SkeletonDomain {
	ret, err := client.clientER.ListDomains(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_SkeletonServiceClient) ListActors(_ivOpts ...rpcclient.InvokeOption) []SkeletonActorItem {
	ret, err := client.clientER.ListActors(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_SkeletonServiceClient) ListServices(_ivOpts ...rpcclient.InvokeOption) []SkeletonServiceItem {
	ret, err := client.clientER.ListServices(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_SkeletonServiceClient) ListResources(_ivOpts ...rpcclient.InvokeOption) []SkeletonResourceItem {
	ret, err := client.clientER.ListResources(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_SkeletonServiceClient) ListWebs(_ivOpts ...rpcclient.InvokeOption) []SkeletonWebItem {
	ret, err := client.clientER.ListWebs(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_SkeletonServiceClient) ListTasks(_ivOpts ...rpcclient.InvokeOption) []SkeletonTask {
	ret, err := client.clientER.ListTasks(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_SkeletonServiceClient) ListEvents(_ivOpts ...rpcclient.InvokeOption) []SkeletonEventItem {
	ret, err := client.clientER.ListEvents(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_SkeletonServiceClient) ListData(_ivOpts ...rpcclient.InvokeOption) []SkeletonData {
	ret, err := client.clientER.ListData(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_SkeletonServiceClient) ListConfigs(_ivOpts ...rpcclient.InvokeOption) []SkeletonConfigItem {
	ret, err := client.clientER.ListConfigs(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

// SkeletonService / ERClient

type SkeletonServiceClientER interface {
	// ListDomains List Domain skeleton。
	//   @returns []SkeletonDomain - Domain skeleton list
	ListDomains(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonDomain, ex.Error)
	// ListActors List Actor Skeleton。
	//   @returns []SkeletonActorItem - Actor skeleton list
	ListActors(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonActorItem, ex.Error)
	// ListServices List Service skeleton。
	//   @returns []SkeletonServiceItem - Service skeleton list
	ListServices(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonServiceItem, ex.Error)
	// ListResources List Resource skeleton。
	//   @returns []SkeletonResourceItem - Resource skeleton list
	ListResources(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonResourceItem, ex.Error)
	// ListWebs List Web Skeletons。
	//   @returns []SkeletonWebItem - Web skeleton list
	ListWebs(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonWebItem, ex.Error)
	// ListTasks List Task skeleton。
	//   @returns []SkeletonTask - Task skeleton list
	ListTasks(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonTask, ex.Error)
	// ListEvents List Event skeletons。
	//   @returns []SkeletonEventItem - Event skeleton list
	ListEvents(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonEventItem, ex.Error)
	// ListData List Data skeleton。
	//   @returns []SkeletonData - Data skeleton list, including Enum
	ListData(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonData, ex.Error)
	// ListConfigs List Config skeleton。
	//   @returns []SkeletonConfigItem - Config skeleton list
	ListConfigs(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonConfigItem, ex.Error)
}

type _SkeletonServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewSkeletonServiceClientER(rpcClient *rpcclient.Client) SkeletonServiceClientER {
	return &_SkeletonServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_SkeletonServiceClientER) ListDomains(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonDomain, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListDomainsSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonDomain)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_SkeletonServiceClientER) ListActors(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonActorItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListActorsSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonActorItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_SkeletonServiceClientER) ListServices(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonServiceItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListServicesSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonServiceItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_SkeletonServiceClientER) ListResources(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonResourceItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListResourcesSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonResourceItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_SkeletonServiceClientER) ListWebs(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonWebItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListWebsSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonWebItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_SkeletonServiceClientER) ListTasks(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonTask, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListTasksSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonTask)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_SkeletonServiceClientER) ListEvents(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonEventItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListEventsSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonEventItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_SkeletonServiceClientER) ListData(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonData, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListDataSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonData)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_SkeletonServiceClientER) ListConfigs(_ivOpts ...rpcclient.InvokeOption) ([]SkeletonConfigItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_SkeletonServiceListConfigsSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]SkeletonConfigItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

// TaskDebugServiceServer Hub Dashboard Task Debugging Service

// TaskDebugService / Spec

var (
	_TaskDebugServiceSpec = &rpcspec.ServiceSpec{
		Type:              rpcspec.ServiceSpecTypeBoth,
		Name:              "TaskDebugService",
		SkelName:          "vine.hub.TaskDebugService",
		Hash:              "230a17bd",
		ServerType:        reflect.TypeFor[TaskDebugServiceServer](),
		DefaultServerType: reflect.TypeFor[*DefaultTaskDebugServiceServer](),
		ClientType:        reflect.TypeFor[TaskDebugServiceClient](),
		ClientCtor:        NewTaskDebugServiceClient,

		ERServerType:        reflect.TypeFor[TaskDebugServiceServerER](),
		WrapperERServerCtor: _NewWrapperTaskDebugServiceServerER,
		DefaultERServerType: reflect.TypeFor[*DefaultTaskDebugServiceServerER](),
		ERClientType:        reflect.TypeFor[TaskDebugServiceClientER](),
		ERClientCtor:        NewTaskDebugServiceClientER,
		Methods: []*rpcspec.MethodSpec{
			_TaskDebugServiceListTasksSpec,
			_TaskDebugServiceListTriggersSpec,
			_TaskDebugServiceBuildDefaultLaunchRequestSpec,
			_TaskDebugServiceLaunchTaskSpec,
		},
	}
	_TaskDebugServiceListTasksSpec = &rpcspec.MethodSpec{
		Name:              "ListTasks",
		SkelName:          "listTasks",
		ArgumentsType:     nil,
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]TaskDebugTaskItem](),
		ValidateResult: func(value any) error {
			ret := value.([]TaskDebugTaskItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			TaskDebugServiceClient.ListTasks,
			TaskDebugServiceClientER.ListTasks,
			TaskDebugServiceServer.ListTasks,
			TaskDebugServiceServerER.ListTasks,
		},
	}
	_TaskDebugServiceListTriggersSpec = &rpcspec.MethodSpec{
		Name:              "ListTriggers",
		SkelName:          "listTriggers",
		ArgumentsType:     reflect.TypeFor[_TaskDebugServiceListTriggersArguments](),
		ValidateArguments: nil,
		ResultType:        reflect.TypeFor[[]TaskDebugTriggerItem](),
		ValidateResult: func(value any) error {
			ret := value.([]TaskDebugTriggerItem)
			if err := rpcspec.CheckValueNotNil(ret, "result"); err != nil {
				return err
			}
			for i0 := range ret {
				if err := (&ret[i0]).Validate(rpcspec.JoinIndex("result", i0)); err != nil {
					return err
				}
			}
			return nil
		},
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			TaskDebugServiceClient.ListTriggers,
			TaskDebugServiceClientER.ListTriggers,
			TaskDebugServiceServer.ListTriggers,
			TaskDebugServiceServerER.ListTriggers,
		},
	}
	_TaskDebugServiceBuildDefaultLaunchRequestSpec = &rpcspec.MethodSpec{
		Name:                        "BuildDefaultLaunchRequest",
		SkelName:                    "buildDefaultLaunchRequest",
		ArgumentsType:               reflect.TypeFor[_TaskDebugServiceBuildDefaultLaunchRequestArguments](),
		ValidateArguments:           nil,
		ResultType:                  reflect.TypeFor[TaskDebugDefaultLaunchRequest](),
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			TaskDebugServiceClient.BuildDefaultLaunchRequest,
			TaskDebugServiceClientER.BuildDefaultLaunchRequest,
			TaskDebugServiceServer.BuildDefaultLaunchRequest,
			TaskDebugServiceServerER.BuildDefaultLaunchRequest,
		},
	}
	_TaskDebugServiceLaunchTaskSpec = &rpcspec.MethodSpec{
		Name:                        "LaunchTask",
		SkelName:                    "launchTask",
		ArgumentsType:               reflect.TypeFor[_TaskDebugServiceLaunchTaskArguments](),
		ValidateArguments:           nil,
		ResultType:                  nil,
		ValidateResult:              nil,
		ArgumentsContainsBinaryType: false,
		ResultContainsBinaryType:    false,
		MethodFuncs: []any{
			TaskDebugServiceClient.LaunchTask,
			TaskDebugServiceClientER.LaunchTask,
			TaskDebugServiceServer.LaunchTask,
			TaskDebugServiceServerER.LaunchTask,
		},
	}
)

// TaskDebugService / Arguments

type _TaskDebugServiceListTriggersArguments struct {
	TaskSkelName string `json:"taskSkelName" arg:"0"`
	SchemaHash   string `json:"schemaHash" arg:"1"`
}

type _TaskDebugServiceBuildDefaultLaunchRequestArguments struct {
	TaskSkelName    string `json:"taskSkelName" arg:"0"`
	SchemaHash      string `json:"schemaHash" arg:"1"`
	TriggerSkelName string `json:"triggerSkelName" arg:"2"`
}

type _TaskDebugServiceLaunchTaskArguments struct {
	Request TaskDebugLaunchRequest `json:"request" arg:"0"`
}

// TaskDebugService / Server

type TaskDebugServiceServer interface {
	// ListTasks List the tasks provided by the application instance。
	ListTasks() []TaskDebugTaskItem
	// ListTriggers List Task triggers。
	//   @param taskSkelName - Task Skel name
	//   @param schemaHash - Task schema hash
	ListTriggers(taskSkelName string, schemaHash string) []TaskDebugTriggerItem
	// BuildDefaultLaunchRequest Generate a default Task launch request。
	//   @param taskSkelName - Task Skel name
	//   @param schemaHash - Task schema hash
	//   @param triggerSkelName - Trigger Skel name
	BuildDefaultLaunchRequest(taskSkelName string, schemaHash string, triggerSkelName string) TaskDebugDefaultLaunchRequest
	// LaunchTask Initiate Task。
	//   @param request - Debug launch request
	LaunchTask(request TaskDebugLaunchRequest)

	mustBeTaskDebugServiceServer()
}

// TaskDebugService / Server / DefaultServer

type DefaultTaskDebugServiceServer struct{}

func (*DefaultTaskDebugServiceServer) ListTasks() []TaskDebugTaskItem {
	ex.PanicNew(ex.InvalidRequest, "method listTasks is not implemented")
	return []TaskDebugTaskItem{}
}

func (*DefaultTaskDebugServiceServer) ListTriggers(string, string) []TaskDebugTriggerItem {
	ex.PanicNew(ex.InvalidRequest, "method listTriggers is not implemented")
	return []TaskDebugTriggerItem{}
}

func (*DefaultTaskDebugServiceServer) BuildDefaultLaunchRequest(string, string, string) TaskDebugDefaultLaunchRequest {
	ex.PanicNew(ex.InvalidRequest, "method buildDefaultLaunchRequest is not implemented")
	return TaskDebugDefaultLaunchRequest{}
}

func (*DefaultTaskDebugServiceServer) LaunchTask(TaskDebugLaunchRequest) {
	ex.PanicNew(ex.InvalidRequest, "method launchTask is not implemented")
}

func (*DefaultTaskDebugServiceServer) mustBeTaskDebugServiceServer() {}

// TaskDebugService / ERServer

type TaskDebugServiceServerER interface {
	ListTasks() ([]TaskDebugTaskItem, ex.Error)
	ListTriggers(taskSkelName string, schemaHash string) ([]TaskDebugTriggerItem, ex.Error)
	BuildDefaultLaunchRequest(taskSkelName string, schemaHash string, triggerSkelName string) (TaskDebugDefaultLaunchRequest, ex.Error)
	LaunchTask(request TaskDebugLaunchRequest) ex.Error

	mustBeTaskDebugServiceServerER()
}

// TaskDebugService / ERServer / WrapperERServer

type _WrapperTaskDebugServiceServerER struct {
	DefaultTaskDebugServiceServer
	serverImpl TaskDebugServiceServer
}

func _NewWrapperTaskDebugServiceServerER(serverImpl TaskDebugServiceServer) TaskDebugServiceServerER {
	return &_WrapperTaskDebugServiceServerER{
		serverImpl: serverImpl,
	}
}

func (service *_WrapperTaskDebugServiceServerER) server() TaskDebugServiceServer {
	if service.serverImpl == nil {
		return &service.DefaultTaskDebugServiceServer
	}
	return service.serverImpl
}

func (service *_WrapperTaskDebugServiceServerER) ListTasks() (ret []TaskDebugTaskItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListTasks()
	return
}

func (service *_WrapperTaskDebugServiceServerER) ListTriggers(taskSkelName string, schemaHash string) (ret []TaskDebugTriggerItem, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().ListTriggers(taskSkelName, schemaHash)
	return
}

func (service *_WrapperTaskDebugServiceServerER) BuildDefaultLaunchRequest(taskSkelName string, schemaHash string, triggerSkelName string) (ret TaskDebugDefaultLaunchRequest, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().BuildDefaultLaunchRequest(taskSkelName, schemaHash, triggerSkelName)
	return
}

func (service *_WrapperTaskDebugServiceServerER) LaunchTask(request TaskDebugLaunchRequest) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	service.server().LaunchTask(request)
	return
}

func (*_WrapperTaskDebugServiceServerER) mustBeTaskDebugServiceServerER() {}

// TaskDebugService / ERServer / DefaultERServer

type DefaultTaskDebugServiceServerER struct {
	_WrapperTaskDebugServiceServerER
}

// TaskDebugService / Client

type TaskDebugServiceClient interface {
	// ListTasks List the tasks provided by the application instance。
	ListTasks(_ivOpts ...rpcclient.InvokeOption) []TaskDebugTaskItem
	// ListTriggers List Task triggers。
	//   @param taskSkelName - Task Skel name
	//   @param schemaHash - Task schema hash
	ListTriggers(taskSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) []TaskDebugTriggerItem
	// BuildDefaultLaunchRequest Generate a default Task launch request。
	//   @param taskSkelName - Task Skel name
	//   @param schemaHash - Task schema hash
	//   @param triggerSkelName - Trigger Skel name
	BuildDefaultLaunchRequest(taskSkelName string, schemaHash string, triggerSkelName string, _ivOpts ...rpcclient.InvokeOption) TaskDebugDefaultLaunchRequest
	// LaunchTask Initiate Task。
	//   @param request - Debug launch request
	LaunchTask(request TaskDebugLaunchRequest, _ivOpts ...rpcclient.InvokeOption)
}

type _TaskDebugServiceClient struct {
	clientER TaskDebugServiceClientER
}

func NewTaskDebugServiceClient(clientER TaskDebugServiceClientER) TaskDebugServiceClient {
	return &_TaskDebugServiceClient{clientER: clientER}
}

func (client *_TaskDebugServiceClient) ListTasks(_ivOpts ...rpcclient.InvokeOption) []TaskDebugTaskItem {
	ret, err := client.clientER.ListTasks(_ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_TaskDebugServiceClient) ListTriggers(taskSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) []TaskDebugTriggerItem {
	ret, err := client.clientER.ListTriggers(taskSkelName, schemaHash, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_TaskDebugServiceClient) BuildDefaultLaunchRequest(taskSkelName string, schemaHash string, triggerSkelName string, _ivOpts ...rpcclient.InvokeOption) TaskDebugDefaultLaunchRequest {
	ret, err := client.clientER.BuildDefaultLaunchRequest(taskSkelName, schemaHash, triggerSkelName, _ivOpts...)
	ex.PanicIfError(err)
	return ret
}

func (client *_TaskDebugServiceClient) LaunchTask(request TaskDebugLaunchRequest, _ivOpts ...rpcclient.InvokeOption) {
	err := client.clientER.LaunchTask(request, _ivOpts...)
	ex.PanicIfError(err)
}

// TaskDebugService / ERClient

type TaskDebugServiceClientER interface {
	// ListTasks List the tasks provided by the application instance。
	ListTasks(_ivOpts ...rpcclient.InvokeOption) ([]TaskDebugTaskItem, ex.Error)
	// ListTriggers List Task triggers。
	//   @param taskSkelName - Task Skel name
	//   @param schemaHash - Task schema hash
	ListTriggers(taskSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) ([]TaskDebugTriggerItem, ex.Error)
	// BuildDefaultLaunchRequest Generate a default Task launch request。
	//   @param taskSkelName - Task Skel name
	//   @param schemaHash - Task schema hash
	//   @param triggerSkelName - Trigger Skel name
	BuildDefaultLaunchRequest(taskSkelName string, schemaHash string, triggerSkelName string, _ivOpts ...rpcclient.InvokeOption) (TaskDebugDefaultLaunchRequest, ex.Error)
	// LaunchTask Initiate Task。
	//   @param request - Debug launch request
	LaunchTask(request TaskDebugLaunchRequest, _ivOpts ...rpcclient.InvokeOption) ex.Error
}

type _TaskDebugServiceClientER struct {
	rpcClient *rpcclient.Client
}

func NewTaskDebugServiceClientER(rpcClient *rpcclient.Client) TaskDebugServiceClientER {
	return &_TaskDebugServiceClientER{
		rpcClient: rpcClient,
	}
}

func (client *_TaskDebugServiceClientER) ListTasks(_ivOpts ...rpcclient.InvokeOption) ([]TaskDebugTaskItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_TaskDebugServiceListTasksSpec.Info(), nil, _ivOpts...)
	ret, _ := retI.([]TaskDebugTaskItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_TaskDebugServiceClientER) ListTriggers(taskSkelName string, schemaHash string, _ivOpts ...rpcclient.InvokeOption) ([]TaskDebugTriggerItem, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_TaskDebugServiceListTriggersSpec.Info(), &_TaskDebugServiceListTriggersArguments{
		TaskSkelName: taskSkelName,
		SchemaHash:   schemaHash,
	}, _ivOpts...)
	ret, _ := retI.([]TaskDebugTriggerItem)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_TaskDebugServiceClientER) BuildDefaultLaunchRequest(taskSkelName string, schemaHash string, triggerSkelName string, _ivOpts ...rpcclient.InvokeOption) (TaskDebugDefaultLaunchRequest, ex.Error) {
	retI, errI := client.rpcClient.Invoke(_TaskDebugServiceBuildDefaultLaunchRequestSpec.Info(), &_TaskDebugServiceBuildDefaultLaunchRequestArguments{
		TaskSkelName:    taskSkelName,
		SchemaHash:      schemaHash,
		TriggerSkelName: triggerSkelName,
	}, _ivOpts...)
	ret, _ := retI.(TaskDebugDefaultLaunchRequest)
	err, _ := errI.(ex.Error)
	return ret, err
}

func (client *_TaskDebugServiceClientER) LaunchTask(request TaskDebugLaunchRequest, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	_, errI := client.rpcClient.Invoke(_TaskDebugServiceLaunchTaskSpec.Info(), &_TaskDebugServiceLaunchTaskArguments{
		Request: request,
	}, _ivOpts...)
	err, _ := errI.(ex.Error)
	return err
}
