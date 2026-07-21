package spec

import (
	"reflect"
	"testing"
)

type inprocClonePayload struct {
	Names []string `json:"names"`
}

type inprocCloneArguments struct {
	Payload inprocClonePayload `json:"payload" arg:"0"`
}

func TestCloneInprocRequestArguments(t *testing.T) {
	methodInfo := newInitializedMethodInfo(reflect.TypeOf(inprocCloneArguments{}), nil, false, false)
	arguments := &inprocCloneArguments{
		Payload: inprocClonePayload{Names: []string{"vine"}},
	}

	cloned := CloneInprocRequestArguments(arguments, methodInfo)

	got, ok := cloned.(*inprocCloneArguments)
	if !ok {
		t.Fatalf("expected *inprocCloneArguments, got %#v", cloned)
	}
	got.Payload.Names[0] = "changed"
	if arguments.Payload.Names[0] != "vine" {
		t.Fatalf("expected original arguments to stay unchanged, got %#v", arguments.Payload.Names)
	}
}

func TestCloneInprocResponseResult(t *testing.T) {
	methodInfo := newInitializedMethodInfo(nil, reflect.TypeOf(inprocClonePayload{}), false, false)
	result := inprocClonePayload{Names: []string{"vine"}}

	cloned := CloneInprocResponseResult(result, methodInfo)

	got, ok := cloned.(inprocClonePayload)
	if !ok {
		t.Fatalf("expected inprocClonePayload, got %#v", cloned)
	}
	got.Names[0] = "changed"
	if result.Names[0] != "vine" {
		t.Fatalf("expected original result to stay unchanged, got %#v", result.Names)
	}
}
