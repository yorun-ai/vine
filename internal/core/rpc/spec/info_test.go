package spec

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

var initializedMethodInfoCounter atomic.Uint64

func TestArgumentSkelIndexes(t *testing.T) {
	type arguments struct {
		Second string `skel:"index(1),sensitive"`
		First  int    `skel:"index(0)"`
	}
	method := newInitializedMethodInfo(reflect.TypeFor[arguments](), nil, false, false)
	require.Equal(t, []any{42, " second "}, method.PositionArguments(&arguments{First: 42, Second: " second "}))
	for _, tag := range []reflect.StructTag{`skel:"index(0)"`, `skel:"index(0),sensitive"`} {
		kind := reflect.StructOf([]reflect.StructField{{Name: "Value", Type: reflect.TypeFor[string](), Tag: tag}})
		require.Equal(t, 0, buildArgumentFieldInfos(kind)[0].ArgIndex)
	}
	for _, tag := range []reflect.StructTag{`arg:"0"`, `skel:"sensitive"`, `skel:"index(x)"`, `skel:"index(-1)"`, `skel:"index(1)"`, `skel:"index(0),index(0)"`} {
		t.Run(string(tag), func(t *testing.T) {
			kind := reflect.StructOf([]reflect.StructField{{Name: "Value", Type: reflect.TypeFor[string](), Tag: tag}})
			require.Panics(t, func() { buildArgumentFieldInfos(kind) })
		})
	}
	type duplicate struct {
		First  int `skel:"index(0)"`
		Second int `skel:"index(0)"`
	}
	require.Panics(t, func() { buildArgumentFieldInfos(reflect.TypeFor[duplicate]()) })
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

func newInitializedMethodInfo(argumentsType reflect.Type, resultType reflect.Type, argumentsContainsBinaryType bool, resultContainsBinaryType bool) *_MethodInfo {
	service := ConvertSpecToInfoForTest(&ServiceSpec{
		Name:     "UserService",
		SkelName: fmt.Sprintf("user.service.%d", initializedMethodInfoCounter.Add(1)),
		Methods: []*MethodSpec{{
			Name:                        "CreateUser",
			SkelName:                    "create_user",
			ArgumentsType:               argumentsType,
			ResultType:                  resultType,
			ArgumentsContainsBinaryType: argumentsContainsBinaryType,
			ResultContainsBinaryType:    resultContainsBinaryType,
		}},
	}).(*_ServiceInfo)
	return service.methods[0].(*_MethodInfo)
}

func TestMethodInfoNewArgumentsAndNewResult(t *testing.T) {
	method := newInitializedMethodInfo(reflect.TypeFor[testArgumentsInput](), reflect.TypeFor[string](), false, false)

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
	method := newInitializedMethodInfo(reflect.TypeFor[testArgumentsInput](), nil, false, false)

	defer func() {
		if recover() == nil {
			t.Fatalf("expected PositionArguments to panic on non-pointer input")
		}
	}()

	method.PositionArguments(testArgumentsInput{})
}

func TestServiceInfoInitBuildsArgumentFieldInfos(t *testing.T) {
	method := newInitializedMethodInfo(reflect.TypeFor[testArgumentsInput](), nil, false, false)

	if len(method.argumentFieldInfos) != 2 {
		t.Fatalf("unexpected argument field info count: got %d", len(method.argumentFieldInfos))
	}
	if method.argumentFieldInfos[0].FieldIndex != 0 || method.argumentFieldInfos[0].ArgIndex != 0 {
		t.Fatalf("unexpected first argument field info: %#v", method.argumentFieldInfos[0])
	}
	if method.argumentFieldInfos[1].FieldIndex != 1 || method.argumentFieldInfos[1].ArgIndex != 1 {
		t.Fatalf("unexpected second argument field info: %#v", method.argumentFieldInfos[1])
	}
}

func TestMethodInfoArgumentsContainsBinaryType(t *testing.T) {
	method := newInitializedMethodInfo(nil, nil, true, false)

	if !method.ArgumentsContainsBinaryType() {
		t.Fatalf("expected method arguments to contain binary type")
	}
	if method.ResultContainsBinaryType() {
		t.Fatalf("expected method result to not contain binary type")
	}
}

func TestMethodInfoResultContainsBinaryType(t *testing.T) {
	method := newInitializedMethodInfo(nil, nil, false, true)

	if method.ArgumentsContainsBinaryType() {
		t.Fatalf("expected method arguments to not contain binary type")
	}
	if !method.ResultContainsBinaryType() {
		t.Fatalf("expected method result to contain binary type")
	}
}

func TestServiceInfoInitRejectsDuplicateArgumentIndexes(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate arg index panic")
		}
	}()

	_ = newInitializedMethodInfo(reflect.TypeFor[testDuplicateArgumentIndexInput](), nil, false, false)
}

func TestServiceInfoInitRejectsArgumentIndexGaps(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected missing arg index panic")
		}
	}()

	_ = newInitializedMethodInfo(reflect.TypeFor[testGapArgumentIndexInput](), nil, false, false)
}
