package app

import (
	"reflect"
	"time"

	"go.yorun.ai/vine/util/vpre"
)

type ListenerTypeAdder func(reflect.Type, ...ListenerOption)

type ListenerOption interface {
	applyListener(*_ListenerOptions)
}

type _ListenerOptionFunc func(*_ListenerOptions)

func (f _ListenerOptionFunc) applyListener(options *_ListenerOptions) {
	f(options)
}

type _ListenerOptions struct {
	Timeout     time.Duration
	Concurrency int
	NoRetry     bool
}

type _ListenerTypeEntry struct {
	kind    reflect.Type
	options _ListenerOptions
}

const (
	defaultListenerTimeout     = 30 * time.Second
	defaultListenerConcurrency = 10
	defaultListenerNoRetry     = false
)

func WithListenerTimeout(timeout time.Duration) ListenerOption {
	vpre.Check(timeout > 0, "listener timeout must be greater than 0")
	return _ListenerOptionFunc(func(options *_ListenerOptions) {
		options.Timeout = timeout
	})
}

func WithListenerConcurrency(concurrency int) ListenerOption {
	vpre.Check(concurrency > 0, "listener concurrency must be greater than 0")
	return _ListenerOptionFunc(func(options *_ListenerOptions) {
		options.Concurrency = concurrency
	})
}

func WithListenerNoRetry() ListenerOption {
	return _ListenerOptionFunc(func(options *_ListenerOptions) {
		options.NoRetry = true
	})
}

func newListenerOptions(options []ListenerOption) _ListenerOptions {
	parsed := _ListenerOptions{
		Timeout:     defaultListenerTimeout,
		Concurrency: defaultListenerConcurrency,
		NoRetry:     defaultListenerNoRetry,
	}
	for _, option := range options {
		option.applyListener(&parsed)
	}
	return parsed
}
