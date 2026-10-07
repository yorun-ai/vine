package legacy

import (
	"fmt"
	"reflect"
	"strings"

	"go.yorun.ai/skel/descriptor"
)

func convertDomain(value *DomainSchema) *descriptor.Domain {
	if value == nil {
		return nil
	}
	result := &descriptor.Domain{
		Name:        value.Domain,
		Description: value.Description,
		Hash:        value.Hash,
		Full:        value.Full,
		Generated:   convertGeneratedInfo(value.Generated),
		Enums:       convertSlice(value.Enums, convertEnum),
		Data:        convertSlice(value.Data, convertData),
		Configs:     convertSlice(value.Configs, convertConfig),
		Webs:        convertSlice(value.Webs, convertWeb),
		Events:      convertSlice(value.Events, convertEvent),
		Actors:      convertSlice(value.Actors, convertActor),
		Resources:   convertSlice(value.Resources, convertResource),
		Services:    convertSlice(value.Services, convertService),
		Tasks:       convertSlice(value.Tasks, convertTask),
	}
	return result
}

func convertGeneratedInfo(value *GeneratedInfo) *descriptor.GeneratedInfo {
	if value == nil {
		return nil
	}
	result := &descriptor.GeneratedInfo{
		CompilerVersion: value.CompilerVersion,
	}
	return result
}

func convertEnum(value *EnumSchema) *descriptor.Enum {
	if value == nil {
		return nil
	}
	result := &descriptor.Enum{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		Items:            convertSlice(value.Items, convertEnumItem),
	}
	return result
}

func convertEnumItem(value *EnumItemSchema) *descriptor.EnumItem {
	if value == nil {
		return nil
	}
	result := &descriptor.EnumItem{
		Name:             value.Name,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
	}
	return result
}

func convertData(value *DataSchema) *descriptor.Data {
	if value == nil {
		return nil
	}
	result := &descriptor.Data{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		Sensitive:        value.Sensitive,
		TypeParameters:   append([]string(nil), value.TypeParameters...),
		Members:          convertSlice(value.Members, convertMember),
	}
	return result
}

func convertConfig(value *ConfigSchema) *descriptor.Config {
	if value == nil {
		return nil
	}
	result := &descriptor.Config{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		Pub:              value.Pub,
		Sensitive:        value.Sensitive,
		Lifecycle:        normalizeLifecycle(value.Lifecycle),
		Members:          convertSlice(value.Members, convertMember),
	}
	return result
}

func convertWeb(value *WebSchema) *descriptor.Web {
	if value == nil {
		return nil
	}
	result := &descriptor.Web{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		AuthMode:         normalizeAuth(value.AuthMode, "web"),
		Audiences:        convertSlice(value.Audiences, convertActorAudience),
		MountPath:        value.MountPath,
	}
	return result
}

func convertEvent(value *EventSchema) *descriptor.Event {
	if value == nil {
		return nil
	}
	result := &descriptor.Event{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		Pub:              value.Pub,
		Ext:              value.Ext,
		Sensitive:        value.Sensitive,
		Members:          convertSlice(value.Members, convertMember),
	}

	if result.Ext {
		result.Pub = false
	}

	return result
}

func convertActorAudience(value *ActorAudienceSchema) *descriptor.ActorAudience {
	if value == nil {
		return nil
	}
	result := &descriptor.ActorAudience{
		Name:     value.Name,
		SkelName: value.SkelName,
		Via:      descriptor.ActorViaKind(value.Via),
	}
	return result
}

func convertService(value *ServiceSchema) *descriptor.Service {
	if value == nil {
		return nil
	}
	result := &descriptor.Service{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		Pub:              value.Pub,
		Api:              value.Api,
		Ext:              value.Ext,
		AuthMode:         normalizeAuth(value.AuthMode, "service"),
		Audiences:        convertSlice(value.Audiences, convertActorAudience),
		Require:          convertPermissionRequire(value.Require),
		Methods:          convertSlice(value.Methods, convertMethod),
	}

	if result.Ext {
		result.Pub = false
	}

	return result
}

func convertMethod(value *MethodSchema) *descriptor.Method {
	if value == nil {
		return nil
	}
	result := &descriptor.Method{
		Name:               value.Name,
		SkelName:           value.SkelName,
		Description:        value.Description,
		Deprecated:         value.Deprecated,
		DeprecatedReason:   value.DeprecatedReason,
		Hash:               value.Hash,
		Example:            value.Example,
		AuthMode:           normalizeAuth(value.AuthMode, "method"),
		Require:            convertPermissionRequire(value.Require),
		InputDescription:   value.InputDescription,
		ArgumentsSensitive: value.ArgumentsSensitive,
		OutputDescription:  value.OutputDescription,
		OutputExample:      value.OutputExample,
		ResultSensitive:    value.ResultSensitive,
		Arguments:          convertSlice(value.Arguments, convertMember),
		ResultType:         convertType(value.ResultType),
	}
	return result
}

func convertPermissionRequire(value *PermRequire) *descriptor.PermissionRequire {
	if value == nil {
		return nil
	}
	result := &descriptor.PermissionRequire{
		Expression: convertPermissionExpression(value.Expr),
	}
	return result
}

func convertPermissionExpression(value *PermExpr) *descriptor.PermissionExpression {
	if value == nil {
		return nil
	}
	result := &descriptor.PermissionExpression{
		Mode:     descriptor.PermissionRequireMode(value.Mode),
		Code:     value.Code,
		Check:    convertPermissionCheckInvocation(value.Check),
		Children: convertSlice(value.Children, convertPermissionExpression),
	}
	return result
}

func convertPermissionCheckInvocation(value *PermCheckInvocation) *descriptor.PermissionCheckInvocation {
	if value == nil {
		return nil
	}
	result := &descriptor.PermissionCheckInvocation{
		ResourceSkelName: value.ResourceSkelName,
		ActionName:       value.ActionName,
		CheckName:        value.CheckName,
		ServiceSkelName:  value.ServiceSkelName,
		MethodSkelName:   value.MethodSkelName,
		CodeArgumentName: value.CodeArgumentName,
		Arguments:        convertSlice(value.Arguments, convertPermissionCheckArgument),
	}
	return result
}

func convertPermissionCheckArgument(value *PermCheckArgument) *descriptor.PermissionCheckArgument {
	if value == nil {
		return nil
	}
	result := &descriptor.PermissionCheckArgument{
		Name:     value.Name,
		JsonPath: value.JsonPath,
		Type:     convertType(value.Type),
	}
	return result
}

func convertTask(value *TaskSchema) *descriptor.Task {
	if value == nil {
		return nil
	}
	result := &descriptor.Task{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		Triggers:         convertSlice(value.Triggers, convertTaskTrigger),
	}
	return result
}

func convertTaskTrigger(value *TriggerSchema) *descriptor.TaskTrigger {
	if value == nil {
		return nil
	}
	result := &descriptor.TaskTrigger{
		Name:               value.Name,
		SkelName:           value.SkelName,
		Description:        value.Description,
		Deprecated:         value.Deprecated,
		DeprecatedReason:   value.DeprecatedReason,
		Hash:               value.Hash,
		InputDescription:   value.InputDescription,
		ArgumentsSensitive: value.ArgumentsSensitive,
		Arguments:          convertSlice(value.Arguments, convertMember),
	}
	return result
}

func convertMember(value *MemberSchema) *descriptor.Member {
	if value == nil {
		return nil
	}
	result := &descriptor.Member{
		Name:             value.Name,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Example:          value.Example,
		Sensitive:        value.Sensitive,
		Type:             convertType(value.Type),
	}
	return result
}

func convertType(value *TypeSchema) *descriptor.Type {
	if value == nil {
		return nil
	}
	result := &descriptor.Type{
		Kind:          descriptor.TypeKind(value.Kind),
		Nullable:      value.Nullable,
		Scalar:        normalizeScalar(value.Scalar),
		Name:          value.Name,
		SkelName:      value.SkelName,
		TypeArguments: convertSlice(value.TypeArguments, convertType),
		Element:       convertType(value.Element),
		Key:           convertType(value.Key),
		Value:         convertType(value.Value),
	}
	return result
}

func convertSlice[S any, T any](values []*S, convert func(*S) *T) []*T {
	if values == nil {
		return nil
	}
	result := make([]*T, len(values))
	for i, value := range values {
		result[i] = convert(value)
	}
	return result
}
func normalizeAuth(value AuthMode, owner string) descriptor.AuthMode {
	switch value {
	case "", AuthModeUnset:
		if owner == "method" {
			return descriptor.AuthModeInherit
		}
		return descriptor.AuthModeRequired
	case AuthModeAuth:
		return descriptor.AuthModeRequired
	case AuthModeNoAuth:
		if owner == "web" {
			return descriptor.AuthModeOff
		}
		return descriptor.AuthModeOptional
	default:
		return descriptor.AuthMode(value)
	}
}
func normalizeScalar(value Scalar) descriptor.Scalar {
	switch value {
	case ScalarLong:
		return descriptor.ScalarInt
	case ScalarDouble:
		return descriptor.ScalarFloat
	}
	return descriptor.Scalar(value)
}
func convertActor(value *ActorSchema) *descriptor.Actor {
	if value == nil {
		return nil
	}
	result := &descriptor.Actor{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		Vias:             append([]descriptor.ActorViaKind(nil), value.Vias...),
	}
	if value.AuthEnabled {
		result.Auth = &descriptor.ActorAuth{
			Credential:      convertData(value.AuthCredential),
			Info:            convertData(value.AuthInfo),
			IdentifierField: value.IdentifierField,
			Service:         convertService(value.AuthService),
			MethodName:      legacyMethodName(value.AuthMethod),
		}
	}
	if value.PermEnabled {
		result.Permission = &descriptor.ActorPermission{
			Service:    convertService(value.PermService),
			MethodName: legacyMethodName(value.PermMethod),
		}
	}
	return result
}
func legacyMethodName(method *MethodSchema) string {
	if method == nil {
		return ""
	}
	return method.Name
}
func convertResourceCheck(value *ResourceCheckSchema) *descriptor.ResourceCheck {
	if value == nil {
		return nil
	}
	return &descriptor.ResourceCheck{
		Name:             value.Name,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		MethodName:       legacyMethodName(value.Method),
	}
}
func convertResourceAction(value *ResourceActionSchema) *descriptor.ResourceAction {
	if value == nil {
		return nil
	}
	return &descriptor.ResourceAction{
		Name:             value.Name,
		PermissionCode:   value.PermissionCode,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Checks:           convertSlice(value.Checks, convertResourceCheck),
	}
}
func convertResource(value *ResourceSchema) *descriptor.Resource {
	if value == nil {
		return nil
	}
	return &descriptor.Resource{
		Name:             value.Name,
		SkelName:         value.SkelName,
		Description:      value.Description,
		Deprecated:       value.Deprecated,
		DeprecatedReason: value.DeprecatedReason,
		Hash:             value.Hash,
		Checks:           convertSlice(value.Checks, convertResourceCheck),
		Actions:          convertSlice(value.Actions, convertResourceAction),
		CheckService:     convertService(value.CheckService),
	}
}

// Convert builds a descriptor without modifying the caller's schema. The registration
// boundary validates the result, including its computed effective policies.
func Convert(value *DomainSchema) (*descriptor.Domain, error) {
	if value == nil {
		return nil, fmt.Errorf("nil legacy domain schema")
	}
	if err := validateMethodCopies(value); err != nil {
		return nil, err
	}
	result := convertDomain(value)
	services := append([]*descriptor.Service(nil), result.Services...)
	for _, actor := range result.Actors {
		if actor == nil {
			return nil, fmt.Errorf("nil legacy actor")
		}
		if actor.Auth != nil {
			services = append(services, actor.Auth.Service)
		}
		if actor.Permission != nil {
			services = append(services, actor.Permission.Service)
		}
	}
	for _, resource := range result.Resources {
		if resource == nil {
			return nil, fmt.Errorf("nil legacy resource")
		}
		if resource.CheckService != nil {
			services = append(services, resource.CheckService)
		}
	}
	for _, service := range services {
		if service == nil {
			return nil, fmt.Errorf("nil legacy service")
		}
		for _, method := range service.Methods {
			effective, err := descriptor.ComputeEffectivePolicy(service, method)
			if err != nil {
				return nil, fmt.Errorf("service %s: %w", service.SkelName, err)
			}
			method.EffectiveAuthMode, method.EffectiveRequire = effective.AuthMode, effective.Require
		}
	}
	return result, nil
}

func normalizeLifecycle(value string) descriptor.ConfigLifecycle {
	if value == "" {
		return descriptor.ConfigLifecycleEternal
	}
	return descriptor.ConfigLifecycle(strings.ToLower(value))
}

// Older generators embedded callback methods twice. Reject conflicting copies
// before replacing the duplicate with a reference to the service method.
func validateMethodCopies(domain *DomainSchema) error {
	check := func(service *ServiceSchema, method *MethodSchema) error {
		if service == nil || method == nil {
			return fmt.Errorf("legacy callback requires a service and method")
		}
		for _, owned := range service.Methods {
			if owned != nil && owned.Name == method.Name {
				if !reflect.DeepEqual(convertMethod(owned), convertMethod(method)) {
					return fmt.Errorf("legacy service %s has conflicting copies of method %s", service.SkelName, method.Name)
				}
				return nil
			}
		}
		return fmt.Errorf("legacy service %s does not contain callback %s", service.SkelName, method.Name)
	}
	for _, actor := range domain.Actors {
		if actor == nil {
			continue
		}
		if actor.AuthEnabled {
			if err := check(actor.AuthService, actor.AuthMethod); err != nil {
				return err
			}
		}
		if actor.PermEnabled {
			if err := check(actor.PermService, actor.PermMethod); err != nil {
				return err
			}
		}
	}
	for _, resource := range domain.Resources {
		if resource == nil {
			continue
		}
		checks := append([]*ResourceCheckSchema(nil), resource.Checks...)
		for _, action := range resource.Actions {
			if action != nil {
				checks = append(checks, action.Checks...)
			}
		}
		for _, callback := range checks {
			if callback == nil {
				return fmt.Errorf("nil legacy resource check")
			}
			if err := check(resource.CheckService, callback.Method); err != nil {
				return err
			}
		}
	}
	return nil
}
