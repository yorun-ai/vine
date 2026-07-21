package di

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

type _BindType string

const (
	bindTypeNone          _BindType = ""
	bindTypeInterface     _BindType = "Interface"
	bindTypeStructPointer _BindType = "StructPointer"
	bindTypeMap           _BindType = "Map"
	bindTypeSlice         _BindType = "Slice"
	bindTypeFunc          _BindType = "Function"
)

type InitDefinition interface {
	DIInit()
}

type DisposeDefinition interface {
	DIDispose()
}

var (
	initDefinitionType    = T[InitDefinition]()
	disposeDefinitionType = T[DisposeDefinition]()
)

func analyzeBindType(targetType reflect.Type) _BindType {
	switch {
	case reflectutil.IsInterfaceType(targetType):
		return bindTypeInterface
	case reflectutil.IsStructPointerType(targetType):
		return bindTypeStructPointer
	case reflectutil.IsMapType(targetType):
		return bindTypeMap
	case reflectutil.IsSliceType(targetType):
		return bindTypeSlice
	case reflectutil.IsFuncType(targetType):
		return bindTypeFunc
	}
	return bindTypeNone
}

func isNilableType(targetType reflect.Type) bool {
	return analyzeBindType(targetType) != bindTypeNone
}

func T[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

func checkBindType(targetType reflect.Type) {
	vpre.Check(analyzeBindType(targetType) != bindTypeNone,
		"unsupported bind type %s, only interface, struct pointer, map, slice and func are supported", targetType)
}

func parseResolveTargetType(targetPtr any) reflect.Type {
	typePtr := reflect.TypeOf(targetPtr)
	vpre.Check(reflectutil.IsPointerType(typePtr), "type should be passed as pointer")

	kind := typePtr.Elem()
	checkBindType(kind)
	return kind
}
