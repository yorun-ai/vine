package access

import (
	"bytes"
	"io"
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
)

func (o *RpcOperation) Check() bool {
	if !o.loadMethodSchema() {
		return false
	}

	requirements := o.requirements()
	if len(requirements) == 0 {
		return true
	}

	expr := reorderPermExpr(mergeRequirements(requirements))
	if !o.checkActorPermissions(expr) {
		return false
	}

	if hasPermissionChecks(expr) && !o.readRequestBody() {
		return false
	}

	ok, code, message := evalPermExpr(expr, o.permissionCodeResults, o.tryCheckPermission)
	if !ok {
		o.writeError(code, message)
	}

	return ok
}

func (o *RpcOperation) requirements() []*skel.PermRequire {
	requirements := make([]*skel.PermRequire, 0, 2)
	if o.serviceSchema.Require != nil {
		requirements = append(requirements, o.serviceSchema.Require)
	}
	if o.methodSchema.Require != nil {
		requirements = append(requirements, o.methodSchema.Require)
	}
	return requirements
}

func (o *RpcOperation) checkActorPermissions(expr *skel.PermExpr) bool {
	codes := collectPermissionCodes(expr)
	if len(codes) == 0 {
		o.permissionCodeResults = map[string]bool{}
		return true
	}

	if !o.actorSchema.PermEnabled || o.actorSchema.PermService == nil || o.actorSchema.PermMethod == nil {
		o.writeError(ex.ClientForbidden, "permission service is not configured")
		return false
	}

	request := o.buildInvokeRequest(
		o.actorSchema.PermService.SkelName,
		o.actorSchema.PermMethod.SkelName,
		map[string]any{"codes": codes},
	)
	if !o.forwardCheckCodesRequest(request, o.actorSchema.PermService.SkelName) {
		return false
	}

	for _, permissionCode := range codes {
		if _, ok := o.permissionCodeResults[permissionCode]; !ok {
			o.writeError(ex.ServiceUnavailable, "permission service result missing code: "+permissionCode)
			return false
		}
	}

	return true
}

func (o *RpcOperation) readRequestBody() bool {
	if o.requestBody != nil {
		return true
	}

	originalBody := o.Request.Body
	body, err := rpchttp.ReadRequestBody(o.Request)
	if originalBody != nil {
		_ = originalBody.Close()
	}
	if err != nil {
		o.writeError(ex.InvalidRequest, err.Error())
		return false
	}

	o.Request.Body = http.NoBody
	if len(body) > 0 {
		o.Request.Body = io.NopCloser(bytes.NewReader(body))
	}
	o.Request.ContentLength = int64(len(body))
	o.requestBody = body
	return true
}

func (o *RpcOperation) tryCheckPermission(check *skel.PermCheckInvocation) (bool, ex.Code, string) {
	params, ok := o.extractCheckParams(check)
	if !ok {
		return false, ex.InvalidRequest, "permission check argument is missing"
	}

	request := o.buildInvokeRequest(
		check.ServiceSkelName,
		check.MethodSkelName,
		params,
	)
	return o.tryForwardCheckRequest(request, check.ServiceSkelName, "permission check failed")
}

func (o *RpcOperation) extractCheckParams(check *skel.PermCheckInvocation) (map[string]any, bool) {
	params := make(map[string]any, len(check.Arguments)+1)
	params["code"] = skel.PermissionCode(check.ResourceSkelName + ":" + check.ActionName)
	for _, argument := range check.Arguments {
		value, ok := o.extractCheckArgument(argument.JsonPath)
		if !ok {
			o.writeError(ex.InvalidRequest, "permission check argument is missing: "+argument.JsonPath)
			return nil, false
		}
		params[argument.Name] = value
	}
	return params, true
}

func (o *RpcOperation) extractCheckArgument(jsonPath string) (any, bool) {
	fullJsonPath := "params." + jsonPath
	switch rpchttp.MediaTypeOf(o.Request.Header.Get(rpchttp.HeaderContentType)) {
	case rpchttp.ContentTypeJson:
		return jsonGetByPath(o.requestBody, fullJsonPath)
	case rpchttp.ContentTypeCbor:
		return cborGetByPath(&o.cborPayload, o.requestBody, fullJsonPath)
	}
	return nil, false
}

func (o *RpcOperation) tryForwardCheckRequest(request *http.Request, serviceSkelName string, defaultMessage string) (bool, ex.Code, string) {
	_, code, message, ok := o.invoke[any](request, serviceSkelName, "permission", defaultMessage)
	return ok, code, message
}

func (o *RpcOperation) forwardCheckCodesRequest(request *http.Request, serviceSkelName string) bool {
	results, code, message, ok := o.invoke[map[string]bool](request, serviceSkelName, "permission", "permission check failed")
	if !ok {
		o.writeError(code, message)
		return false
	}

	o.permissionCodeResults = results
	return true
}
