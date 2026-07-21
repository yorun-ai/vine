package spec

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
)

var initializedMethodInfoCounter uint64

type testValidateArgumentsInput struct {
	Name *string `arg:"0"`
	Age  int     `arg:"1"`
}

type testDuplicateArgumentIndexInput struct {
	First  string `arg:"0"`
	Second int    `arg:"0"`
}

type testGapArgumentIndexInput struct {
	First string `arg:"0"`
	Third int    `arg:"2"`
}

func newInitializedMethodInfo(argumentsType reflect.Type, resultType reflect.Type, argumentsContainsBinaryType bool, resultContainsBinaryType bool) *_MethodInfo {
	service := ConvertSpecToInfoForTest(&ServiceSpec{
		Name:     "UserService",
		SkelName: fmt.Sprintf("user.service.%d", atomic.AddUint64(&initializedMethodInfoCounter, 1)),
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

func newInitializedMethodInfoWithValidators(validateArguments func(any) error, validateResult func(any) error) *_MethodInfo {
	service := ConvertSpecToInfoForTest(&ServiceSpec{
		Name:     "UserService",
		SkelName: fmt.Sprintf("user.service.%d", atomic.AddUint64(&initializedMethodInfoCounter, 1)),
		Methods: []*MethodSpec{{
			Name:              "CreateUser",
			SkelName:          "create_user",
			ValidateArguments: validateArguments,
			ValidateResult:    validateResult,
		}},
	}).(*_ServiceInfo)
	return service.methods[0].(*_MethodInfo)
}

func newInitializedMethodInfoWithValidateResult(validateResult func(any) error) *_MethodInfo {
	return newInitializedMethodInfoWithValidators(nil, validateResult)
}

func TestMethodInfoValidateArgumentsAcceptsDefaultGeneratedNoop(t *testing.T) {
	name := "vine"
	method := newInitializedMethodInfo(reflect.TypeOf(testValidateArgumentsInput{}), nil, false, false)

	if err := method.ValidateArguments(&testValidateArgumentsInput{Name: &name, Age: 7}); err != nil {
		t.Fatalf("ValidateArguments() error = %v", err)
	}
}

func TestMethodInfoNewArgumentsAndNewResult(t *testing.T) {
	method := newInitializedMethodInfo(reflect.TypeOf(testValidateArgumentsInput{}), reflect.TypeOf(""), false, false)

	if !method.HasArguments() {
		t.Fatalf("expected method to have arguments")
	}
	if !method.HasResult() {
		t.Fatalf("expected method to have result")
	}
	if _, ok := method.NewArguments().(*testValidateArgumentsInput); !ok {
		t.Fatalf("expected NewArguments to return *testValidateArgumentsInput")
	}
	if _, ok := method.NewResult().(*string); !ok {
		t.Fatalf("expected NewResult to return *string")
	}
}

func TestMethodInfoValidateArgumentsWithoutArgumentsIsNoop(t *testing.T) {
	method := newInitializedMethodInfo(nil, nil, false, false)
	method.name = "Ping"

	if err := method.ValidateArguments(nil); err != nil {
		t.Fatalf("ValidateArguments() error = %v", err)
	}
}

func TestMethodInfoPositionArgumentsRequiresPointer(t *testing.T) {
	method := newInitializedMethodInfo(reflect.TypeOf(testValidateArgumentsInput{}), nil, false, false)

	defer func() {
		if recover() == nil {
			t.Fatalf("expected PositionArguments to panic on non-pointer input")
		}
	}()

	method.PositionArguments(testValidateArgumentsInput{})
}

func TestServiceInfoInitBuildsArgumentFieldInfos(t *testing.T) {
	method := newInitializedMethodInfo(reflect.TypeOf(testValidateArgumentsInput{}), nil, false, false)

	if len(method.argumentFieldInfos) != 2 {
		t.Fatalf("unexpected argument field info count: got %d", len(method.argumentFieldInfos))
	}
	if method.argumentFieldInfos[0].Name != "Name" || method.argumentFieldInfos[0].ArgIndex != 0 {
		t.Fatalf("unexpected first argument field info: %#v", method.argumentFieldInfos[0])
	}
	if method.argumentFieldInfos[1].Name != "Age" || method.argumentFieldInfos[1].ArgIndex != 1 {
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

func TestMethodInfoValidateResult(t *testing.T) {
	wantErr := fmt.Errorf("bad result")
	method := newInitializedMethodInfoWithValidateResult(func(any) error {
		return wantErr
	})

	if got := method.ValidateResult(nil); got != wantErr {
		t.Fatalf("unexpected validate result error: %v", got)
	}
}

func TestMethodInfoValidateResultDefaultsToNoop(t *testing.T) {
	method := newInitializedMethodInfoWithValidateResult(nil)

	if err := method.ValidateResult(nil); err != nil {
		t.Fatalf("unexpected default validate result error: %v", err)
	}
}

func TestMethodInfoValidateArguments(t *testing.T) {
	wantErr := fmt.Errorf("bad arguments")
	method := newInitializedMethodInfoWithValidators(func(any) error {
		return wantErr
	}, nil)

	if got := method.ValidateArguments(nil); got != wantErr {
		t.Fatalf("unexpected validate arguments error: %v", got)
	}
}

func TestMethodInfoValidateArgumentsDefaultsToNoop(t *testing.T) {
	method := newInitializedMethodInfoWithValidators(nil, nil)

	if err := method.ValidateArguments(nil); err != nil {
		t.Fatalf("unexpected default validate arguments error: %v", err)
	}
}

func TestServiceInfoInitRejectsDuplicateArgumentIndexes(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate arg index panic")
		}
	}()

	_ = newInitializedMethodInfo(reflect.TypeOf(testDuplicateArgumentIndexInput{}), nil, false, false)
}

func TestServiceInfoInitRejectsArgumentIndexGaps(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected missing arg index panic")
		}
	}()

	_ = newInitializedMethodInfo(reflect.TypeOf(testGapArgumentIndexInput{}), nil, false, false)
}
