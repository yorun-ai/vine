package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type scopeNoMarker struct{}

type scopeSingleton struct {
	SingletonScoped
}

type scopeExecution struct {
	ExecutionScoped
}

type scopeTransient struct {
	TransientScoped
}

type scopeInheritedSingleton struct {
	scopeSingleton
}

type scopeInheritedExecution struct {
	scopeExecution
}

type scopeInheritedTransient struct {
	scopeTransient
}

type scopeInheritedExecutionFromPointer struct {
	*scopeExecution
}

type scopeExecutionOverridesEmbedded struct {
	ExecutionScoped
	scopeInheritedSingleton
	scopeInheritedTransient
}

type scopeConflictedFromEmbeds struct {
	scopeInheritedSingleton
	scopeInheritedTransient
}

type scopeAnonymousNonStruct interface {
	Run()
}

type scopeAnonymousNonStructEmbed struct {
	scopeAnonymousNonStruct
}

type scopePointerExecutionMarker struct {
	*ExecutionScoped
}

type scopeCycleA struct {
	*scopeCycleB
}

type scopeCycleB struct {
	*scopeCycleA
}

func TestScanDefaultScope(t *testing.T) {
	tests := []struct {
		name string
		kind reflect.Type
		want Scope
	}{
		{
			name: "no marker",
			kind: T[*scopeNoMarker](),
			want: noScope,
		},
		{
			name: "singleton marker on self",
			kind: T[*scopeSingleton](),
			want: SingletonScope,
		},
		{
			name: "execution marker on self",
			kind: T[*scopeExecution](),
			want: ExecutionScope,
		},
		{
			name: "transient marker on self",
			kind: T[*scopeTransient](),
			want: TransientScope,
		},
		{
			name: "inherit singleton from embed",
			kind: T[*scopeInheritedSingleton](),
			want: SingletonScope,
		},
		{
			name: "inherit execution from embed",
			kind: T[*scopeInheritedExecution](),
			want: ExecutionScope,
		},
		{
			name: "inherit transient from embed",
			kind: T[*scopeInheritedTransient](),
			want: TransientScope,
		},
		{
			name: "inherit execution from embedded pointer struct",
			kind: T[*scopeInheritedExecutionFromPointer](),
			want: ExecutionScope,
		},
		{
			name: "skip unsupported anonymous non-struct field",
			kind: T[*scopeAnonymousNonStructEmbed](),
			want: noScope,
		},
		{
			name: "self marker overrides embedded scopes",
			kind: T[*scopeExecutionOverridesEmbedded](),
			want: ExecutionScope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scanDeclaredScope(tt.kind))
		})
	}
}

func TestScanDefaultScopePanicsOnConflictedEmbeds(t *testing.T) {
	assert.Panics(t, func() {
		scanDeclaredScope(T[*scopeConflictedFromEmbeds]())
	})
}

func TestScanDefaultScopePanicsOnPointerMarker(t *testing.T) {
	assert.Panics(t, func() {
		scanDeclaredScope(T[*scopePointerExecutionMarker]())
	})
}

func TestScanDefaultScopePanicsOnCyclicEmbeds(t *testing.T) {
	assert.Panics(t, func() {
		scanDeclaredScope(T[*scopeCycleA]())
	})
}
