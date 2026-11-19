package spec

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

var initializedMethodInfoCounter atomic.Uint64

func TestMethodInfoPositionsArgumentsBySkelIndex(t *testing.T) {
	type arguments struct {
		Second string `skel:"index(1),sensitive"`
		First  int    `skel:"index(0)"`
	}
	method := newInitializedMethodInfo(reflect.TypeFor[arguments](), nil)
	require.Equal(t, []any{42, " second "}, method.PositionArguments(&arguments{First: 42, Second: " second "}))
}

func TestServiceInfoInitRejectsMissingOrNegativeArgumentIndexes(t *testing.T) {
	for _, tag := range []reflect.StructTag{`skel:"sensitive"`, `skel:"index(-1)"`} {
		t.Run(string(tag), func(t *testing.T) {
			kind := reflect.StructOf([]reflect.StructField{{
				Name: "Value",
				Type: reflect.TypeFor[string](),
				Tag:  tag,
			}})
			require.Panics(t, func() {
				newInitializedMethodInfo(kind, nil)
			})
		})
	}
}

type testArgumentsInput struct {
	Name *string `skel:"index(0)"`
	Age  int     `skel:"index(1)"`
}

type testDuplicateArgumentIndexInput struct {
	First  string `skel:"index(0)"`
	Second int    `skel:"index(0)"`
}

type testGapArgumentIndexInput struct {
	First string `skel:"index(0)"`
	Third int    `skel:"index(2)"`
}

func newInitializedMethodInfo(argumentsType reflect.Type, resultType reflect.Type) *_MethodInfo {
	service := registerTestService(&ServiceSpec{
		Type:     ServiceSpecTypeBoth,
		Name:     "UserService",
		SkelName: fmt.Sprintf("user.service.%d", initializedMethodInfoCounter.Add(1)),
		Methods: []*MethodSpec{{
			Name:          "CreateUser",
			SkelName:      "create_user",
			ArgumentsType: argumentsType,
			ResultType:    resultType,
		}},
	}).(*_ServiceInfo)
	return service.methods[0].(*_MethodInfo)
}

func TestMethodInfoNewArgumentsAndNewResult(t *testing.T) {
	method := newInitializedMethodInfo(reflect.TypeFor[testArgumentsInput](), reflect.TypeFor[string]())

	if !method.HasArguments() {
		t.Fatalf("expected method to have arguments")
	}
	if !method.HasResult() {
		t.Fatalf("expected method to have result")
	}
	if _, ok := method.NewArguments().(*testArgumentsInput); !ok {
		t.Fatalf("expected NewArguments to return *testArgumentsInput")
	}
	if _, ok := method.NewResult().(*string); !ok {
		t.Fatalf("expected NewResult to return *string")
	}
}

func TestMethodInfoPositionArgumentsRequiresPointer(t *testing.T) {
	method := newInitializedMethodInfo(reflect.TypeFor[testArgumentsInput](), nil)

	defer func() {
		if recover() == nil {
			t.Fatalf("expected PositionArguments to panic on non-pointer input")
		}
	}()

	method.PositionArguments(testArgumentsInput{})
}

func TestServiceInfoInitRejectsDuplicateArgumentIndexes(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate arg index panic")
		}
	}()

	_ = newInitializedMethodInfo(reflect.TypeFor[testDuplicateArgumentIndexInput](), nil)
}

func TestServiceInfoInitRejectsArgumentIndexGaps(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected missing arg index panic")
		}
	}()

	_ = newInitializedMethodInfo(reflect.TypeFor[testGapArgumentIndexInput](), nil)
}
