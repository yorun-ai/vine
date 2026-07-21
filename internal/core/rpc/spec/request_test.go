package spec

import (
	"reflect"
	"testing"
)

type testPositionalArgumentsInput struct {
	Second int    `arg:"1"`
	First  string `arg:"0"`
}

func TestRequestImplPositionalArgumentsUsesMethodArgumentOrder(t *testing.T) {
	method := newInitializedMethodInfo(reflect.TypeOf(testPositionalArgumentsInput{}), nil, false, false)
	method.name = "CreateUser"
	req := &RequestImpl{
		MethodInfoValue: method,
		ArgumentsValue: &testPositionalArgumentsInput{
			First:  "vine",
			Second: 7,
		},
	}

	got := req.PositionalArguments()
	if len(got) != 2 {
		t.Fatalf("unexpected positional argument count: got %d", len(got))
	}
	if got[0] != "vine" {
		t.Fatalf("unexpected first positional argument: %#v", got[0])
	}
	if got[1] != 7 {
		t.Fatalf("unexpected second positional argument: %#v", got[1])
	}
}

func TestRequestImplPositionalArgumentsWithoutMethodArgumentsReturnsNil(t *testing.T) {
	method := newInitializedMethodInfo(nil, nil, false, false)
	method.name = "Ping"
	req := &RequestImpl{
		MethodInfoValue: method,
	}

	if got := req.PositionalArguments(); got != nil {
		t.Fatalf("expected nil positional arguments, got %#v", got)
	}
}
