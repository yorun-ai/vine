package rpcgw

import (
	"cmp"
	"encoding/json/v2"
	"net/http"
	"slices"
	"strings"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel/jsonschema"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vpre"
)

type _InspectService struct {
	Name             string `json:"name"`
	SkelName         string `json:"skelName"`
	Description      string `json:"description,omitempty"`
	Deprecated       bool   `json:"deprecated,omitzero"`
	DeprecatedReason string `json:"deprecatedReason,omitempty"`
}

type _InspectServiceList struct {
	Services []_InspectService `json:"services"`
}

type _InspectMethod struct {
	Name             string                      `json:"name"`
	SkelName         string                      `json:"skelName"`
	Description      string                      `json:"description,omitempty"`
	Deprecated       bool                        `json:"deprecated,omitzero"`
	DeprecatedReason string                      `json:"deprecatedReason,omitempty"`
	AuthMode         skeldesc.AuthMode           `json:"authMode"`
	Require          *skeldesc.PermissionRequire `json:"require,omitempty"`
}

type _InspectMethodList struct {
	Service _InspectService  `json:"service"`
	Methods []_InspectMethod `json:"methods"`
}

type _InspectDetail struct {
	_InspectMethod
	Service      string            `json:"service"`
	InputSchema  jsonschema.Schema `json:"inputSchema"`
	OutputSchema jsonschema.Schema `json:"outputSchema"`
}

type _InspectError struct {
	Code    ex.Code `json:"code"`
	Message string  `json:"message"`
}

func (g *RpcGateway) serveInspect(ctx *spec.Context) {
	w, r := ctx.ResponseWriter, ctx.Request
	w.Header().Set("Cache-Control", "no-store")
	g.mutex.Lock()
	enabled := g.inspectEnabled
	g.mutex.Unlock()
	if !enabled {
		writeInspectError(w, ex.NotFound, "rpcgw path is not found")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, OPTIONS")
		writeInspectError(w, ex.InvalidRequest, "inspect requires GET")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, pathInspect)
	list := path == "" || path == "/" || path == "/_list"
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if !list && (len(parts) != 2 || parts[0] == "" || parts[1] == "") {
		writeInspectError(w, ex.NotFound, "inspect path is not found")
		return
	}
	snapshot, visible := g.apiSnapshot()
	if snapshot.Syncing {
		w.Header().Set("Retry-After", "1")
		writeInspectError(w, ex.ServiceUnavailable, "inspect descriptors are syncing")
		return
	}
	if list {
		services := make([]_InspectService, 0, len(visible))
		for _, service := range visible {
			services = append(services, inspectServiceSummary(service))
		}
		slices.SortFunc(services, func(a _InspectService, b _InspectService) int { return cmp.Compare(a.SkelName, b.SkelName) })
		writeInspect(w, http.StatusOK, _InspectServiceList{Services: services})
		return
	}
	service := visible[parts[0]]
	if service == nil {
		writeInspectError(w, ex.NotFound, "inspect service is not found")
		return
	}
	if parts[1] == "_list" {
		methods := make([]_InspectMethod, 0, len(service.Methods))
		for _, method := range service.Methods {
			methods = append(methods, inspectMethod(method))
		}
		slices.SortFunc(methods, func(a _InspectMethod, b _InspectMethod) int { return cmp.Compare(a.SkelName, b.SkelName) })
		writeInspect(w, http.StatusOK, _InspectMethodList{Service: inspectServiceSummary(service), Methods: methods})
		return
	}
	method := service.MethodBySkelName(parts[1])
	if method == nil {
		writeInspectError(w, ex.NotFound, "inspect method is not found")
		return
	}
	input, output, err := jsonschema.Method(method, snapshot.Data, snapshot.Enums)
	if err != nil {
		rpcGatewayLogger.Warn("vine.portal inspect schema is unavailable", "service", service.SkelName, "method", method.SkelName, "error", err)
		writeInspectError(w, ex.ServiceUnavailable, "inspect schema is unavailable")
		return
	}
	writeInspect(w, http.StatusOK, _InspectDetail{_InspectMethod: inspectMethod(method), Service: service.SkelName, InputSchema: input, OutputSchema: output})
}

func inspectServiceSummary(service *skeldesc.Service) _InspectService {
	return _InspectService{
		Name: service.Name, SkelName: service.SkelName, Description: service.Description,
		Deprecated: service.Deprecated, DeprecatedReason: service.DeprecatedReason,
	}
}

func inspectMethod(method *skeldesc.Method) _InspectMethod {
	return _InspectMethod{
		Name: method.Name, SkelName: method.SkelName, Description: method.Description,
		Deprecated: method.Deprecated, DeprecatedReason: method.DeprecatedReason,
		AuthMode: method.EffectiveAuthMode, Require: method.EffectiveRequire,
	}
}

func writeInspectError(w http.ResponseWriter, code ex.Code, message string) {
	writeInspect(w, ex.HTTPStatusCode(code), _InspectError{Code: code, Message: message})
}

func writeInspect(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	vpre.MustNil(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
