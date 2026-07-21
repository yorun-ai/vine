package di

import (
	"reflect"

	"go.yorun.ai/vine/util/vpre"
)

type Scope string

const (
	noScope        Scope = ""
	SingletonScope Scope = "SingletonScope"
	ExecutionScope Scope = "ExecutionScope"
	TransientScope Scope = "TransientScope"
)

func isValidScope(scope Scope) bool {
	switch scope {
	case SingletonScope, ExecutionScope, TransientScope:
		return true
	default:
		return false
	}
}

type SingletonScoped struct{}

type ExecutionScoped struct{}

type TransientScoped struct{}

func scanDeclaredScope(structPtrType reflect.Type) Scope {
	return doScanDeclaredScope(structPtrType.Elem(), map[reflect.Type]bool{})
}

var (
	singletonScopedType = T[SingletonScoped]()
	executionScopedType = T[ExecutionScoped]()
	transientScopedType = T[TransientScoped]()
	scopedTypeToScope   = map[reflect.Type]Scope{
		singletonScopedType: SingletonScope,
		executionScopedType: ExecutionScope,
		transientScopedType: TransientScope,
	}
)

func doScanDeclaredScope(structType reflect.Type, scanningTypes map[reflect.Type]bool) Scope {
	vpre.Check(!scanningTypes[structType], "cyclic embedded struct found when scan scope for %s", structType)
	scanningTypes[structType] = true
	defer delete(scanningTypes, structType)

	selfScope := noScope
	embeddedStructTypes := []reflect.Type{}

	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		if !field.Anonymous {
			continue
		}

		if scope, ok := scopedTypeToScope[field.Type]; ok {
			vpre.Check(selfScope == noScope, "both %s and %s found in embed field of %s", selfScope, field.Type, structType)
			selfScope = scope
			continue
		}

		if field.Type.Kind() == reflect.Ptr {
			if _, ok := scopedTypeToScope[field.Type.Elem()]; ok {
				vpre.Panicf("pointer scoped marker %s is not allowed in embed field of %s, use %s instead", field.Type, structType, field.Type.Elem())
			}
		}

		embeddedStructType, ok := unwrapEmbeddedStructType(field.Type)
		if !ok {
			continue
		}
		embeddedStructTypes = append(embeddedStructTypes, embeddedStructType)
	}

	if selfScope != noScope {
		return selfScope
	}

	return scanEmbeddedDeclaredScope(structType, embeddedStructTypes, scanningTypes)
}

func scanEmbeddedDeclaredScope(parentType reflect.Type, embeddedStructTypes []reflect.Type, scanningTypes map[reflect.Type]bool) Scope {
	embeddedScope := noScope
	embeddedScopeType := reflect.Type(nil)

	for _, fieldType := range embeddedStructTypes {
		scope := doScanDeclaredScope(fieldType, scanningTypes)
		if scope != noScope {
			vpre.Check(embeddedScope == noScope, "conflicted embed scoped field type on %s found, %s.scope=%s, but %s.scope=%s",
				parentType, embeddedScopeType, embeddedScope, fieldType, scope)
			embeddedScope = scope
			embeddedScopeType = fieldType
		}
	}

	return embeddedScope
}
