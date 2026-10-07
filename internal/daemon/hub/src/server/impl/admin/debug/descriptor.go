package debug

import (
	"strings"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/util/vslice"
)

func (s *ServiceDebugApiServiceServerImpl) hasServiceDescriptor(serviceSkelName string, descriptorHash string) bool {
	for _, version := range s.DescriptorRepo.ListServiceDescriptorVersions() {
		if version.Descriptor.SkelName == serviceSkelName && (descriptorHash == "" || version.DescriptorHash == descriptorHash) {
			return true
		}
	}
	return false
}

func (s *ServiceDebugApiServiceServerImpl) findServiceDescriptor(serviceSkelName string, descriptorHash string) *skeldesc.Service {
	for _, version := range s.DescriptorRepo.ListServiceDescriptorVersions() {
		if version.Descriptor.SkelName == serviceSkelName && (descriptorHash == "" || version.DescriptorHash == descriptorHash) {
			return version.Descriptor
		}
	}
	ex.PanicNew(ex.NotFound, "service descriptor not found")
	panic("unreachable")
}

func (s *ServiceDebugApiServiceServerImpl) findMethodDescriptor(serviceDescriptor *skeldesc.Service, methodSkelName string) *skeldesc.Method {
	method := serviceDescriptor.MethodBySkelName(methodSkelName)
	ex.PanicNewIfNot(method != nil, ex.NotFound, "method descriptor not found")
	return method
}

func (s *ServiceDebugApiServiceServerImpl) findActorDescriptor(actorSkelName string) *skeldesc.Actor {
	for _, descriptor := range s.DescriptorRepo.ListActorDescriptors() {
		if descriptor.SkelName == actorSkelName {
			return descriptor
		}
	}
	ex.PanicNew(ex.NotFound, "actor descriptor not found")
	panic("unreachable")
}

func (s *ServiceDebugApiServiceServerImpl) serviceDebugActors(serviceDescriptor *skeldesc.Service) []skeled.ServiceDebugActorItem {
	ret := make([]skeled.ServiceDebugActorItem, 0, len(serviceDescriptor.Audiences))
	for _, audience := range serviceDescriptor.Audiences {
		actor := s.findActorDescriptor(audience.SkelName)
		if actor.Auth == nil {
			continue
		}
		ret = append(ret, skeled.ServiceDebugActorItem{
			Name:          actor.Name,
			SkelName:      actor.SkelName,
			InfoSkelName:  actor.Auth.Info.SkelName,
			ActorInfoJson: s.defaultBuilder().defaultActorInfoJson(actor.SkelName),
		})
	}
	return vslice.SortBy(ret, func(a skeled.ServiceDebugActorItem, b skeled.ServiceDebugActorItem) bool {
		return strings.Compare(a.SkelName, b.SkelName) < 0
	})
}

func toServiceDebugMethodItem(method *skeldesc.Method) skeled.ServiceDebugMethodItem {
	return skeled.ServiceDebugMethodItem{
		Name:              method.Name,
		SkelName:          method.SkelName,
		Description:       method.Description,
		Deprecated:        method.Deprecated,
		DeprecatedReason:  method.DeprecatedReason,
		InputDescription:  method.InputDescription,
		OutputDescription: method.OutputDescription,
		Example:           method.Example,
		OutputExample:     method.OutputExample,
		Arguments:         toDebugSkeletonFields(method.Arguments),
		ResultType:        formatSkeletonType(method.ResultType),
	}
}

func toDebugSkeletonFields(descriptors []*skeldesc.Member) []skeled.SkeletonField {
	ret := make([]skeled.SkeletonField, 0, len(descriptors))
	for _, descriptor := range descriptors {
		ret = append(ret, skeled.SkeletonField{
			Name:             descriptor.Name,
			Type:             formatSkeletonType(descriptor.Type),
			Description:      descriptor.Description,
			Deprecated:       descriptor.Deprecated,
			DeprecatedReason: descriptor.DeprecatedReason,
			Example:          descriptor.Example,
			Sensitive:        descriptor.Sensitive,
		})
	}
	return ret
}

func formatSkeletonType(typeDescriptor *skeldesc.Type) string {
	if typeDescriptor == nil {
		return ""
	}
	var ret string
	switch typeDescriptor.Kind {
	case skeldesc.TypeKindScalar:
		ret = string(typeDescriptor.Scalar)
	case skeldesc.TypeKindEnum, skeldesc.TypeKindData, skeldesc.TypeKindConfig, skeldesc.TypeKindEvent:
		ret = formatSkeletonNamedType(typeDescriptor)
		if len(typeDescriptor.TypeArguments) > 0 {
			args := make([]string, 0, len(typeDescriptor.TypeArguments))
			for _, arg := range typeDescriptor.TypeArguments {
				args = append(args, formatSkeletonType(arg))
			}
			ret += "<" + strings.Join(args, ", ") + ">"
		}
	case skeldesc.TypeKindTypeParameter:
		ret = typeDescriptor.Name
		if ret == "" {
			ret = shortSkelName(typeDescriptor.SkelName)
		}
	case skeldesc.TypeKindList:
		ret = "list<" + formatSkeletonType(typeDescriptor.Element) + ">"
	case skeldesc.TypeKindMap:
		ret = "map<" + formatSkeletonType(typeDescriptor.Key) + ", " + formatSkeletonType(typeDescriptor.Value) + ">"
	default:
		ret = string(typeDescriptor.Kind)
	}
	if typeDescriptor.Nullable {
		ret += "?"
	}
	return ret
}

func formatSkeletonNamedType(typeDescriptor *skeldesc.Type) string {
	if typeDescriptor.SkelName != "" {
		return typeDescriptor.SkelName
	}
	if typeDescriptor.Name != "" {
		return typeDescriptor.Name
	}
	return shortSkelName(typeDescriptor.SkelName)
}

func shortSkelName(skelName string) string {
	_, name, ok := strings.Cut(skelName, ".")
	for ok {
		skelName = name
		_, name, ok = strings.Cut(skelName, ".")
	}
	return skelName
}
