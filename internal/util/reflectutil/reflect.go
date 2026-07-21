package reflectutil

import (
	"reflect"
	"unsafe"

	"go.yorun.ai/vine/util/vpre"
)

func GetPrivateField(item any, fieldName string) any {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	f := v.FieldByName(fieldName)
	f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	return f.Interface()
}

func PointerTo[T any](val T) *T {
	return &val
}

func TargetOf(val any) any {
	pv := reflect.ValueOf(val)
	vpre.Check(IsPointerType(pv.Type()), "val is not pointer")
	return pv.Elem().Interface()
}

func FlattenFields(iface any) []reflect.Value {
	var fields []reflect.Value
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)

	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)

		switch v.Kind() {
		case reflect.Struct:
			fields = append(fields, FlattenFields(v.Interface())...)
		default:
			fields = append(fields, v)
		}
	}

	return fields
}

func EmbeddedStructTypes(kind reflect.Type) []reflect.Type {
	if kind == nil {
		return nil
	}
	if kind.Kind() == reflect.Ptr {
		kind = kind.Elem()
	}
	if kind.Kind() != reflect.Struct {
		return nil
	}

	var embeddedTypes []reflect.Type
	for i := 0; i < kind.NumField(); i++ {
		field := kind.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			embeddedTypes = append(embeddedTypes, field.Type)
		}
	}
	return embeddedTypes
}

func EmbeddedStructFields(value reflect.Value) []reflect.Value {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	var fields []reflect.Value
	for i := 0; i < value.NumField(); i++ {
		fieldType := value.Type().Field(i)
		if !fieldType.Anonymous || fieldType.Type.Kind() != reflect.Struct {
			continue
		}

		fieldValue := value.Field(i)
		if fieldValue.CanAddr() && !fieldValue.CanInterface() {
			fieldValue = UnsafeField(fieldValue)
		}
		fields = append(fields, fieldValue)
	}
	return fields
}

func CloneStructPointer(instance any) any {
	kind := reflect.TypeOf(instance)
	value := reflect.ValueOf(instance)
	vpre.Check(IsStructPointerType(kind), "instance %s must be pointer to struct", kind)

	copied := reflect.New(kind.Elem())
	copied.Elem().Set(value.Elem())
	return copied.Interface()
}

func UnsafeField(field reflect.Value) reflect.Value {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
}
