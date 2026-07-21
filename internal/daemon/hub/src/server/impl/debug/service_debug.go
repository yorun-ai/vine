package debug

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vslice"
)

type ServiceDebugServiceServerImpl struct {
	skeled.DefaultServiceDebugServiceServer

	RegistryRepo core.RegistryRepo `inject:""`
	SchemaRepo   core.SchemaRepo   `inject:""`
}

func (s *ServiceDebugServiceServerImpl) defaultBuilder() _DebugDefaultBuilder {
	return _DebugDefaultBuilder{SchemaRepo: s.SchemaRepo}
}

func (s *ServiceDebugServiceServerImpl) ListAppInstances() []skeled.ServiceDebugAppInstance {
	return listDebugAppInstances(s.RegistryRepo.ListAppStatuses(), func(*core.AppStatus) bool { return true })
}

func listDebugAppInstances(statuses []*core.AppStatus, include func(*core.AppStatus) bool) []skeled.ServiceDebugAppInstance {
	ret := make([]skeled.ServiceDebugAppInstance, 0, len(statuses))
	for _, status := range statuses {
		if !include(status) {
			continue
		}
		ret = append(ret, skeled.ServiceDebugAppInstance{
			AppName:       status.Name,
			AppInstanceId: status.InstanceId,
			AppVersion:    status.Version,
			Endpoint:      status.Endpoint,
		})
	}
	return vslice.SortBy(ret, func(a skeled.ServiceDebugAppInstance, b skeled.ServiceDebugAppInstance) bool {
		if a.AppName != b.AppName {
			return strings.Compare(a.AppName, b.AppName) < 0
		}
		return strings.Compare(a.AppInstanceId, b.AppInstanceId) < 0
	})
}

func (s *ServiceDebugServiceServerImpl) ListServices() []skeled.ServiceDebugServiceItem {
	ret := []skeled.ServiceDebugServiceItem{}
	seen := map[string]struct{}{}
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		for _, handler := range status.ServiceHandlers {
			if !s.hasServiceSchema(handler.ServiceSkelName, handler.SchemaHash) {
				continue
			}
			key := handler.ServiceSkelName + "\x00" + handler.SchemaHash
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			ret = append(ret, skeled.ServiceDebugServiceItem{
				ServiceSkelName: handler.ServiceSkelName,
				SchemaHash:      handler.SchemaHash,
			})
		}
	}
	return vslice.SortBy(ret, func(a skeled.ServiceDebugServiceItem, b skeled.ServiceDebugServiceItem) bool {
		if a.ServiceSkelName != b.ServiceSkelName {
			return strings.Compare(a.ServiceSkelName, b.ServiceSkelName) < 0
		}
		return strings.Compare(a.SchemaHash, b.SchemaHash) < 0
	})
}

func (s *ServiceDebugServiceServerImpl) ListServiceAppInstances(serviceSkelName string, schemaHash string) []skeled.ServiceDebugAppInstance {
	return listDebugAppInstances(s.RegistryRepo.ListAppStatuses(), func(status *core.AppStatus) bool {
		return statusHasServiceHandler(status, serviceSkelName, schemaHash)
	})
}

func (s *ServiceDebugServiceServerImpl) ListMethods(serviceSkelName string, schemaHash string) []skeled.ServiceDebugMethodItem {
	serviceSchema := s.findServiceSchema(serviceSkelName, schemaHash)
	ret := make([]skeled.ServiceDebugMethodItem, 0, len(serviceSchema.Methods))
	for _, method := range serviceSchema.Methods {
		ret = append(ret, toServiceDebugMethodItem(method))
	}
	return vslice.SortBy(ret, func(a skeled.ServiceDebugMethodItem, b skeled.ServiceDebugMethodItem) bool {
		return strings.Compare(a.SkelName, b.SkelName) < 0
	})
}

func (s *ServiceDebugServiceServerImpl) BuildDefaultInvokeRequest(serviceSkelName string, schemaHash string, methodSkelName string) skeled.ServiceDebugDefaultInvokeRequest {
	serviceSchema := s.findServiceSchema(serviceSkelName, schemaHash)
	methodSchema := s.findMethodSchema(serviceSchema, methodSkelName)
	trace := meta.InitialTrace()
	actors := s.serviceDebugActors(serviceSchema)
	actorSkelName := (*string)(nil)
	actorInfoJson := skel.JSON("{}")
	if len(actors) > 0 {
		defaultActorSkelName := actors[0].SkelName
		actorSkelName = &defaultActorSkelName
		actorInfoJson = actors[0].ActorInfoJson
	}
	return skeled.ServiceDebugDefaultInvokeRequest{
		TraceId:       trace.Id(),
		SpanId:        trace.Span(),
		Actors:        actors,
		ActorSkelName: actorSkelName,
		ActorInfoJson: actorInfoJson,
		ParamsJson:    s.defaultBuilder().defaultParamsJson(methodSchema),
	}
}

func (s *ServiceDebugServiceServerImpl) InvokeService(request skeled.ServiceDebugInvokeRequest) skeled.ServiceDebugInvokeResponse {
	var params any
	if strings.TrimSpace(string(request.ParamsJson)) == "" {
		params = map[string]any{}
	} else {
		err := json.Unmarshal([]byte(request.ParamsJson), &params)
		ex.PanicNewIfError(err, ex.InvalidRequest)
	}

	registration := s.resolveRpcServiceRegistration(request)

	trace := debugTrace(request.TraceId, request.SpanId)
	actor := s.debugActor(request.ActorSkelName, request.ActorInfoJson)
	ex.PanicNewIfNot(request.TimeoutSeconds > 0, ex.InvalidRequest, "timeoutSeconds must be greater than 0")
	debugContext, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()
	httpRequest := rpchttp.BuildInvokeRequest(rpchttp.InvokeRequest{
		Context:         meta.NewContext(debugContext, trace, nil, actor),
		Endpoint:        registration.Endpoint,
		ServiceSkelName: request.ServiceSkelName,
		MethodSkelName:  request.MethodSkelName,
		Params:          params,
		Trace:           trace,
		Client:          debugClient(),
		Actor:           actor,
	})

	response, err := doServiceDebugInvokeRequest(httpRequest)
	ex.PanicNewIfError(err, ex.ServiceUnavailable)
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	ex.PanicNewIfError(err, ex.ServiceUnavailable)

	return skeled.ServiceDebugInvokeResponse{
		HttpStatus:  response.StatusCode,
		RpcStatus:   response.Header.Get(rpchttp.HeaderRpcStatus),
		HeadersJson: skel.JSON(string(vcode.MustMarshalJson(response.Header))),
		BodyJson:    skel.JSON(string(bodyBytes)),
	}
}

func (s *ServiceDebugServiceServerImpl) resolveRpcServiceRegistration(request skeled.ServiceDebugInvokeRequest) *core.RpcServiceRegistration {
	appName := optionalTrimmedString(request.AppName)
	appInstanceId := optionalTrimmedString(request.AppInstanceId)
	if appName != "" && appInstanceId != "" {
		_, statusOK := s.RegistryRepo.GetAppStatus(appName, appInstanceId)
		ex.PanicNewIfNot(statusOK, ex.NotFound, "app instance not found")
		registration, ok := s.RegistryRepo.GetRpcServiceRegistration(
			request.ServiceSkelName,
			appName,
			appInstanceId,
		)
		ex.PanicNewIfNot(ok, ex.NotFound, "rpc service registration not found")
		return registration
	}

	candidates := s.serviceAppInstances(request.ServiceSkelName, request.SchemaHash)
	ex.PanicNewIfNot(len(candidates) > 0, ex.NotFound, "rpc service registration not found")
	ex.PanicNewIfNot(len(candidates) == 1, ex.InvalidRequest, "multiple app instances provide this service; please select a specific app instance")
	registration, ok := s.RegistryRepo.GetRpcServiceRegistration(
		request.ServiceSkelName,
		candidates[0].AppName,
		candidates[0].AppInstanceId,
	)
	ex.PanicNewIfNot(ok, ex.NotFound, "rpc service registration not found")
	return registration
}

func (s *ServiceDebugServiceServerImpl) serviceAppInstances(serviceSkelName string, schemaHash string) []skeled.ServiceDebugAppInstance {
	return listDebugAppInstances(s.RegistryRepo.ListAppStatuses(), func(status *core.AppStatus) bool {
		return statusHasServiceHandler(status, serviceSkelName, schemaHash)
	})
}

func statusHasServiceHandler(status *core.AppStatus, serviceSkelName string, schemaHash string) bool {
	for _, handler := range status.ServiceHandlers {
		if handler.ServiceSkelName == serviceSkelName && (schemaHash == "" || handler.SchemaHash == schemaHash) {
			return true
		}
	}
	return false
}

func optionalTrimmedString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
