package goutil

import (
	"context"
	"reflect"
	"sync"
	"time"
)

type Waitable struct {
	wg sync.WaitGroup
}

func (w *Waitable) GoSafely(fn any, args ...any) {
	w.wg.Go(func() {
		defer Rescue()

		fnV := reflect.ValueOf(fn)
		argValues := make([]reflect.Value, len(args))
		for i, arg := range args {
			argValues[i] = reflect.ValueOf(arg)
		}
		fnV.Call(argValues) // the results are discarded
	})
}

func (w *Waitable) GoTickSafely(ctx context.Context, interval time.Duration, fn any, args ...any) {
	w.GoSafely(Tick, ctx, interval, func() {
		w.GoSafely(fn, args...)
	})
}

func (w *Waitable) Wait() {
	w.wg.Wait()
}
