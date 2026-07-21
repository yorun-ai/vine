package goutil

import (
	"reflect"
)

func WrapWithOnPanic(fn func(), panicFn func()) func() {
	return func() {
		// use boolean assignment check mechanic to avoid add unnecessary stacktrace for panicked error
		panicked := true
		defer func() {
			if panicked {
				panicFn()
			}
		}()
		fn()
		panicked = false
	}
}

func RunWithOnPanic(fn func(), panicFn func()) {
	WrapWithOnPanic(fn, panicFn)()
}

func RunWithRecover(recoverFn func(any), fn any, args ...any) {
	defer func() {
		if r := recover(); r != nil {
			recoverFn(r)
		}
	}()

	fnV := reflect.ValueOf(fn)
	fnV.Call(reflectArgs(args)) // discard results
}
