package di

import (
	"fmt"
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

const injectionTagName = "inject"

type _InjectedField struct {
	index []int
	name  string
	kind  reflect.Type
}

func scanInjectedFields(structPtrType reflect.Type) []*_InjectedField {
	return doScanInjectedFields(structPtrType.Elem())
}

func doScanInjectedFields(structType reflect.Type) []*_InjectedField {
	var injectedFields []*_InjectedField

	for fieldIndex := 0; fieldIndex < structType.NumField(); fieldIndex++ {
		field := structType.Field(fieldIndex)
		_, hasInjectionTag := field.Tag.Lookup(injectionTagName)

		if !field.IsExported() {
			vpre.Check(!hasInjectionTag, "unexported filed %s of %s cannot be injected", field.Name, structType.Name())
			continue
		}

		if field.Anonymous {
			injectedFields = append(injectedFields, scanEmbeddedInjectedFields(structType, fieldIndex, field, hasInjectionTag)...)
			continue
		}

		if !hasInjectionTag {
			continue
		}

		injectedFields = append(injectedFields, &_InjectedField{
			index: []int{fieldIndex},
			name:  field.Name,
			kind:  field.Type,
		})
	}

	return injectedFields
}

func scanEmbeddedInjectedFields(parentType reflect.Type, fieldIndex int, field reflect.StructField, hasInjectionTag bool) []*_InjectedField {
	vpre.Check(!hasInjectionTag, "Embed Struct %s of %s cannot be tagged with %s", field.Name, parentType.Name(), injectionTagName)

	embeddedStructType, ok := unwrapEmbeddedStructType(field.Type)
	if !ok {
		return nil
	}

	var injectedFields []*_InjectedField
	for _, injectedField := range doScanInjectedFields(embeddedStructType) {
		injectedField.index = append([]int{fieldIndex}, injectedField.index...)
		injectedField.name = fmt.Sprintf("%s.%s", field.Name, injectedField.name)
		injectedFields = append(injectedFields, injectedField)
	}
	vpre.Check(field.Type.Kind() != reflect.Ptr || len(injectedFields) == 0,
		"embedded pointer struct %s of %s contains injected fields, use value embedding instead",
		field.Name, parentType.Name())
	return injectedFields
}

func unwrapEmbeddedStructType(fieldType reflect.Type) (reflect.Type, bool) {
	switch {
	case fieldType.Kind() == reflect.Struct:
		return fieldType, true
	case reflectutil.IsStructPointerType(fieldType):
		return fieldType.Elem(), true
	default:
		return nil, false
	}
}
