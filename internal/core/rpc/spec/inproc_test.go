package spec

import (
	"reflect"
	"testing"
)

type inprocClonePayload struct {
	Names []string `json:"names"`
}

type inprocCloneArguments struct {
	Payload inprocClonePayload `json:"payload" skel:"index(0)"`
}

func TestCloneInprocRequestArgumentsIgnoresMethodClone(t *testing.T) {
	cloneCalls := 0
	methodInfo := ConvertSpecToInfoForTest(new(ServiceSpec{
		Name:     "InprocCloneService",
		SkelName: "test.inproc.clone.arguments",
		Methods: []*MethodSpec{{
			Name:          "Clone",
			SkelName:      "clone",
			ArgumentsType: reflect.TypeFor[inprocCloneArguments](),
			CloneArguments: func(value any) any {
				cloneCalls++
				return value
			},
		}},
	})).Methods()[0]
	arguments := &inprocCloneArguments{Payload: inprocClonePayload{Names: []string{"vine"}}}

	cloned := CloneInprocRequestArguments(arguments, methodInfo).(*inprocCloneArguments)

	if cloneCalls != 0 {
		t.Fatalf("CloneArguments call count = %d, want 0", cloneCalls)
	}
	if cloned == arguments {
		t.Fatal("runtime clone returned the caller-owned value")
	}
	cloned.Payload.Names[0] = "changed"
	if arguments.Payload.Names[0] != "vine" {
		t.Fatalf("runtime clone did not isolate arguments: %#v", arguments.Payload.Names)
	}
}

func TestCloneInprocResponseResultIgnoresMethodClone(t *testing.T) {
	cloneCalls := 0
	methodInfo := ConvertSpecToInfoForTest(new(ServiceSpec{
		Name:     "InprocCloneService",
		SkelName: "test.inproc.clone.result",
		Methods: []*MethodSpec{{
			Name:       "Clone",
			SkelName:   "clone",
			ResultType: reflect.TypeFor[inprocClonePayload](),
			CloneResult: func(value any) any {
				cloneCalls++
				return value
			},
		}},
	})).Methods()[0]
	result := inprocClonePayload{Names: []string{"vine"}}

	cloned := CloneInprocResponseResult(result, methodInfo).(inprocClonePayload)

	if cloneCalls != 0 {
		t.Fatalf("CloneResult call count = %d, want 0", cloneCalls)
	}
	cloned.Names[0] = "changed"
	if result.Names[0] != "vine" {
		t.Fatalf("runtime clone did not isolate result: %#v", result.Names)
	}
}
