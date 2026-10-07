package access

import (
	"bytes"
	"io"
	"net/http"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

func (o *RpcOperation) Check() bool {
	if !o.loadMethodDescriptor() {
		return false
	}

	requirement := o.methodDescriptor.EffectiveRequire
	if requirement == nil {
		return true
	}

	expr := requirement.Expression
	if !o.checkActorPermissions(expr) {
		return false
	}

	if hasPermissionChecks(expr) && !o.readRequestBody() {
		return false
	}

	ok, code, message, reason := evalPermExpr(expr, o.permissionCodeResults, o.tryCheckPermission)
	if !ok {
		o.writeErrorWithReason(code, message, reason)
	}

	return ok
}

func (o *RpcOperation) checkActorPermissions(expr *skeldesc.PermissionExpression) bool {
	codes := collectPermissionCodes(expr)
	if len(codes) == 0 {
		o.permissionCodeResults = map[string]bool{}
		return true
	}

	if o.actorDescriptor.Permission == nil || o.actorDescriptor.Permission.Method() == nil {
		o.writeError(ex.ClientForbidden, "permission service is not configured")
		return false
	}

	request := o.buildInvokeRequest(
		o.actorDescriptor.Permission.Service.SkelName,
		o.actorDescriptor.Permission.Method().SkelName,
		map[string]any{"codes": codes},
	)
	if !o.forwardCheckCodesRequest(request, o.actorDescriptor.Permission.Service.SkelName) {
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

func (o *RpcOperation) tryCheckPermission(check *skeldesc.PermissionCheckInvocation) (bool, ex.Code, string, string) {
	params, ok := o.extractCheckParams(check)
	if !ok {
		return false, ex.InvalidRequest, "permission check argument is missing", ""
	}

	request := o.buildInvokeRequest(
		check.ServiceSkelName,
		check.MethodSkelName,
		params,
	)
	return o.tryForwardCheckRequest(request, check.ServiceSkelName, "permission check failed")
}

func (o *RpcOperation) extractCheckParams(check *skeldesc.PermissionCheckInvocation) (map[string]any, bool) {
	params := make(map[string]any, len(check.Arguments)+1)
	params[check.CodeArgumentName] = check.ResourceSkelName + ":" + check.ActionName
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
	return requestParamsGetByPath(&o.requestParams, o.requestBody, o.Request.Header.Get(rpchttp.HeaderContentType), jsonPath)
}

func (o *RpcOperation) tryForwardCheckRequest(request *http.Request, serviceSkelName string, defaultMessage string) (bool, ex.Code, string, string) {
	_, code, message, reason, ok := o.invoke[any](request, serviceSkelName, "permission", defaultMessage)
	return ok, code, message, reason
}

func (o *RpcOperation) forwardCheckCodesRequest(request *http.Request, serviceSkelName string) bool {
	results, code, message, reason, ok := o.invoke[map[string]bool](request, serviceSkelName, "permission", "permission check failed")
	if !ok {
		o.writeErrorWithReason(code, message, reason)
		return false
	}

	o.permissionCodeResults = results
	return true
}
