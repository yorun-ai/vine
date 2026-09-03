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

func TestCloneInprocRequestArgumentsUsesMethodClone(t *testing.T) {
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
				arguments := value.(*inprocCloneArguments)
				return &inprocCloneArguments{
					Payload: inprocClonePayload{Names: append([]string(nil), arguments.Payload.Names...)},
				}
			},
		}},
	})).Methods()[0]
	arguments := &inprocCloneArguments{Payload: inprocClonePayload{Names: []string{"vine"}}}

	cloned := CloneInprocRequestArguments(arguments, methodInfo).(*inprocCloneArguments)

	if cloneCalls != 1 {
		t.Fatalf("CloneArguments call count = %d, want 1", cloneCalls)
	}
	cloned.Payload.Names[0] = "changed"
	if arguments.Payload.Names[0] != "vine" {
		t.Fatalf("generated clone did not isolate arguments: %#v", arguments.Payload.Names)
	}
}

func TestCloneInprocResponseResultUsesMethodClone(t *testing.T) {
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
				result := value.(inprocClonePayload)
				return inprocClonePayload{Names: append([]string(nil), result.Names...)}
			},
		}},
	})).Methods()[0]
	result := inprocClonePayload{Names: []string{"vine"}}

	cloned := CloneInprocResponseResult(result, methodInfo).(inprocClonePayload)

	if cloneCalls != 1 {
		t.Fatalf("CloneResult call count = %d, want 1", cloneCalls)
	}
	cloned.Names[0] = "changed"
	if result.Names[0] != "vine" {
		t.Fatalf("generated clone did not isolate result: %#v", result.Names)
	}
}
