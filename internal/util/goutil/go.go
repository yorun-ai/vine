package goutil

import (
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
)

func GoSafely(fn any, args ...any) {
	go func() {
		defer Rescue()
		fnV := reflect.ValueOf(fn)
		fnV.Call(reflectArgs(args)) // discard results
	}()
}

// Rescue Call recover() and log the recovered, avoiding crash of entire application
func Rescue() {
	recovered := recover()
	rescueRecovered(recovered)
}

func reflectArgs(args []any) []reflect.Value {
	argValues := make([]reflect.Value, len(args))
	for i, arg := range args {
		argValues[i] = reflect.ValueOf(arg)
	}
	return argValues
}

func rescueRecovered(recovered any) {
	if recovered != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Groutine Paniced: %s\n%s", recovered, debug.Stack())
	}
}
