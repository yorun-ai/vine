// Package vbean provides helpers for generated data beans, the Go types that
// carry Skel data across Rpc boundaries.
package vbean

import (
	"reflect"
	"sync"
)

// DeepClone returns a copy of value whose mutable data is isolated from the
// source value. It targets generated data beans: the structs, lists, maps,
// pointers and scalars that carry Rpc arguments and results.
//
// DeepClone preserves nil pointers, slices, maps and interfaces, and it deep
// copies every exported reference field of a struct. Two boundaries are
// intentional:
//
//   - Unexported fields are copied like any other Go assignment, so
//     reference-backed data they hold stays shared with the source.
//   - Functions, channels, unsafe pointers and values with pointer cycles are
//     copied as-is; they cannot cross an Rpc boundary.
//
// DeepClone is safe for concurrent use.
func DeepClone[T any](value T) T {
	source := reflect.ValueOf(value)
	if !source.IsValid() {
		var zero T
		return zero
	}

	valueType := reflect.TypeFor[T]()
	if valueType.Kind() == reflect.Interface {
		valueType = source.Type()
	}
	cloner := clonerFor(valueType)
	if cloner.identity {
		return value
	}

	cloned := reflect.New(valueType).Elem()
	cloner.copyInto(cloned, source)
	return cloned.Interface().(T)
}

// _Cloner copies values of one Go type into a settable target. Copying into an
// existing target keeps clones of nested beans out of the heap, which the clone
// call sites on Rpc boundaries notice.
type _Cloner struct {
	copyInto func(target reflect.Value, source reflect.Value)
	// identity reports that copying the value already isolates its mutable
	// data, so nested values of this type need no clone step of their own.
	identity bool
}

// _FieldCopy points at one exported struct field that needs its own clone step.
type _FieldCopy struct {
	index  int
	cloner *_Cloner
}

var identityCloner = &_Cloner{
	identity: true,
	copyInto: func(target reflect.Value, source reflect.Value) {
		target.Set(source)
	},
}

var (
	clonerMu     sync.RWMutex
	clonerByType = map[reflect.Type]*_Cloner{}
)

// clonerFor returns the cloner for valueType, building it on first use.
func clonerFor(valueType reflect.Type) *_Cloner {
	clonerMu.RLock()
	cloner, found := clonerByType[valueType]
	clonerMu.RUnlock()
	if found {
		return cloner
	}

	clonerMu.Lock()
	defer clonerMu.Unlock()
	return clonerLocked(valueType)
}

// clonerLocked returns the cloner for valueType while the write lock is held.
// The cloner is registered before its members are built, so recursive types
// refer to the cloner that is still being built instead of building forever.
func clonerLocked(valueType reflect.Type) *_Cloner {
	if cloner, found := clonerByType[valueType]; found {
		return cloner
	}

	cloner := &_Cloner{}
	clonerByType[valueType] = cloner
	cloner.copyInto, cloner.identity = buildClonerLocked(valueType)
	return cloner
}

func buildClonerLocked(valueType reflect.Type) (func(target reflect.Value, source reflect.Value), bool) {
	switch valueType.Kind() {
	case reflect.Pointer:
		elementCloner := clonerLocked(valueType.Elem())
		return func(target reflect.Value, source reflect.Value) {
			if source.IsNil() {
				target.SetZero()
				return
			}
			cloned := reflect.New(valueType.Elem())
			elementCloner.copyInto(cloned.Elem(), source.Elem())
			target.Set(cloned)
		}, false

	case reflect.Slice:
		elementCloner := clonerLocked(valueType.Elem())
		return func(target reflect.Value, source reflect.Value) {
			if source.IsNil() {
				target.SetZero()
				return
			}
			cloned := reflect.MakeSlice(valueType, source.Len(), source.Len())
			if elementCloner.identity {
				reflect.Copy(cloned, source)
			} else {
				for index := 0; index < source.Len(); index++ {
					elementCloner.copyInto(cloned.Index(index), source.Index(index))
				}
			}
			target.Set(cloned)
		}, false

	case reflect.Array:
		elementCloner := clonerLocked(valueType.Elem())
		if elementCloner.identity {
			return identityCloner.copyInto, true
		}
		return func(target reflect.Value, source reflect.Value) {
			for index := 0; index < source.Len(); index++ {
				elementCloner.copyInto(target.Index(index), source.Index(index))
			}
		}, false

	case reflect.Map:
		keyCloner := clonerLocked(valueType.Key())
		itemCloner := clonerLocked(valueType.Elem())
		return func(target reflect.Value, source reflect.Value) {
			if source.IsNil() {
				target.SetZero()
				return
			}
			cloned := reflect.MakeMapWithSize(valueType, source.Len())
			clonedKey := reflect.New(valueType.Key()).Elem()
			clonedItem := reflect.New(valueType.Elem()).Elem()
			iterator := source.MapRange()
			for iterator.Next() {
				if !keyCloner.identity {
					keyCloner.copyInto(clonedKey, iterator.Key())
				}
				if !itemCloner.identity {
					itemCloner.copyInto(clonedItem, iterator.Value())
				}
				key := iterator.Key()
				if !keyCloner.identity {
					key = clonedKey
				}
				item := iterator.Value()
				if !itemCloner.identity {
					item = clonedItem
				}
				cloned.SetMapIndex(key, item)
			}
			target.Set(cloned)
		}, false

	case reflect.Struct:
		fieldCopies := make([]_FieldCopy, 0, valueType.NumField())
		for index := 0; index < valueType.NumField(); index++ {
			field := valueType.Field(index)
			if !field.IsExported() {
				continue
			}
			fieldCloner := clonerLocked(field.Type)
			if fieldCloner.identity {
				continue
			}
			fieldCopies = append(fieldCopies, _FieldCopy{index: index, cloner: fieldCloner})
		}
		if len(fieldCopies) == 0 {
			return identityCloner.copyInto, true
		}
		return func(target reflect.Value, source reflect.Value) {
			target.Set(source)
			for _, fieldCloner := range fieldCopies {
				fieldCloner.cloner.copyInto(
					target.Field(fieldCloner.index),
					source.Field(fieldCloner.index),
				)
			}
		}, false

	case reflect.Interface:
		return func(target reflect.Value, source reflect.Value) {
			if source.IsNil() {
				target.SetZero()
				return
			}
			element := source.Elem()
			elementCloner := clonerFor(element.Type())
			if elementCloner.identity {
				target.Set(source)
				return
			}
			cloned := reflect.New(element.Type()).Elem()
			elementCloner.copyInto(cloned, element)
			target.Set(cloned)
		}, false

	default:
		return identityCloner.copyInto, true
	}
}
