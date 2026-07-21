package reflectutil

import "reflect"

var errorInterface = reflect.TypeFor[error]()

func IsErrorType(kind reflect.Type) bool {
	return kind != nil && kind.Implements(errorInterface)
}

func IsError(val any) bool {
	return IsErrorType(reflect.TypeOf(val))
}

func IsInterfaceType(kind reflect.Type) bool {
	return kind != nil && kind.Kind() == reflect.Interface
}

func IsInterface(val any) bool {
	return IsInterfaceType(reflect.TypeOf(val))
}

func IsPointerType(kind reflect.Type) bool {
	return kind != nil && kind.Kind() == reflect.Ptr
}

func IsPointer(val any) bool {
	return IsPointerType(reflect.TypeOf(val))
}

func IsStructPointerType(kind reflect.Type) bool {
	return kind != nil && kind.Kind() == reflect.Ptr && kind.Elem().Kind() == reflect.Struct
}

func IsStructPointer(val any) bool {
	return IsStructPointerType(reflect.TypeOf(val))
}

func IsFuncType(kind reflect.Type) bool {
	return kind != nil && kind.Kind() == reflect.Func
}

func IsFunc(val any) bool {
	return IsFuncType(reflect.TypeOf(val))
}

func IsMapType(kind reflect.Type) bool {
	return kind != nil && kind.Kind() == reflect.Map
}

func IsMap(val any) bool {
	return IsMapType(reflect.TypeOf(val))
}

func IsSliceType(kind reflect.Type) bool {
	return kind != nil && kind.Kind() == reflect.Slice
}

func IsSlice(val any) bool {
	return IsSliceType(reflect.TypeOf(val))
}

func IsNilValue(val reflect.Value) bool {
	if !val.IsValid() {
		return true
	}

	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}

func IsNil(val any) bool {
	return IsNilValue(reflect.ValueOf(val))
}
