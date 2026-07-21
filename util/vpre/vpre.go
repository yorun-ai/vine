package vpre

import (
	"errors"
	"fmt"
	"reflect"
)

// ALL functions below panic directly to avoid unnecessary stack

// panic

// Panic panics with err.
func Panic(err error) {
	panic(err)
}

// Panicf formats an error and panics with it.
func Panicf(panicTpl string, panicArgs ...any) {
	panicf(panicTpl, panicArgs...)
}

// check

// Check panics with a formatted error when condition is false.
func Check(condition bool, panicTpl string, panicArgs ...any) {
	if !condition {
		panicf(panicTpl, panicArgs...)
	}
}

// CheckNot panics with a formatted error when condition is true.
func CheckNot(condition bool, panicTpl string, panicArgs ...any) {
	if condition {
		panicf(panicTpl, panicArgs...)
	}
}

// CheckOK panics unless dict contains key.
func CheckOK[K comparable, V any](dict map[K]V, key K, panicTpl string, panicArgs ...any) {
	if _, ok := dict[key]; !ok {
		panicf(panicTpl, panicArgs...)
	}
}

// CheckNotOK panics when dict contains key.
func CheckNotOK[K comparable, V any](dict map[K]V, key K, panicTpl string, panicArgs ...any) {
	if _, ok := dict[key]; ok {
		panicf(panicTpl, panicArgs...)
	}
}

// CheckNil panics unless val is nil, including typed nil values.
func CheckNil(val any, panicTpl string, panicArgs ...any) {
	if !isNil(val) {
		panicf(panicTpl, panicArgs...)
	}
}

// CheckNotNil panics when val is nil, including typed nil values.
func CheckNotNil(val any, panicTpl string, panicArgs ...any) {
	if isNil(val) {
		panicf(panicTpl, panicArgs...)
	}
}

// CheckEmpty panics unless str is empty.
func CheckEmpty(str string, panicTpl string, panicArgs ...any) {
	if str != "" {
		panicf(panicTpl, panicArgs...)
	}
}

// CheckNotEmpty panics when str is empty.
func CheckNotEmpty(str string, panicTpl string, panicArgs ...any) {
	if str == "" {
		panicf(panicTpl, panicArgs...)
	}
}

// CheckNilError panics with a wrapped formatted error when err is non-nil.
func CheckNilError(err error, panicTpl string, panicArgs ...any) {
	CheckNilErrorWithAction(err, nil, panicTpl, panicArgs...)
}

// CheckNilErrorWithAction calls action and panics with a wrapped error when err is non-nil.
func CheckNilErrorWithAction(err error, action func(error), panicTpl string, panicArgs ...any) {
	if err != nil {
		if action != nil {
			action(err)
		}
		panic(fmt.Errorf("%s: %w", fmt.Sprintf(panicTpl, panicArgs...), err))
	}
}

// CheckFunc panics with the lazily constructed message when condition is false.
func CheckFunc(condition bool, panicMsgFunc func() string) {
	if !condition {
		panic(errors.New(panicMsgFunc()))
	}
}

// must

var unexpectedCondition = errors.New("unexpected condition")

// Must panics with a generic invariant error when condition is false.
func Must(condition bool) {
	if !condition {
		panic(unexpectedCondition)
	}
}

// MustNotReach always panics to mark unreachable control flow.
func MustNotReach() {
	panic(unexpectedCondition)
}

// MustNil panics with a generic invariant error unless val is nil.
func MustNil(val any) {
	if !isNil(val) {
		panic(unexpectedCondition)
	}
}

// MustNotNil panics with a generic invariant error when val is nil.
func MustNotNil(val any) {
	if isNil(val) {
		panic(unexpectedCondition)
	}
}

// MustEmpty panics with a generic invariant error unless str is empty.
func MustEmpty(str string) {
	if str != "" {
		panic(unexpectedCondition)
	}
}

// MustNotEmpty panics with a generic invariant error when str is empty.
func MustNotEmpty(str string) {
	if str == "" {
		panic(unexpectedCondition)
	}
}

// util

func isNil(val any) bool {
	valValue := reflect.ValueOf(val)
	if !valValue.IsValid() {
		return true
	}

	switch valValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valValue.IsNil()
	default:
		return false
	}
}

func panicf(panicTpl string, panicArgs ...any) {
	panic(fmt.Errorf(panicTpl, panicArgs...))
}
