package spec

import (
	"encoding/json"
	"reflect"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

// Inproc Rpc clones request and response values to keep network Rpc mutation
// semantics. Without this, server-side writes to slice/map/pointer fields could
// leak back into caller-owned arguments.

func CloneInprocRequestArguments(arguments any, methodInfo MethodInfo) any {
	if methodInfo.ArgumentsType() == nil || arguments == nil {
		return arguments
	}

	target := methodInfo.NewArguments()
	cloneInprocValue(arguments, target, methodInfo.ArgumentsContainsBinaryType())
	return target
}

func CloneInprocResponseResult(result any, methodInfo MethodInfo) any {
	if methodInfo.ResultType() == nil || result == nil {
		return result
	}

	target := methodInfo.NewResult()
	cloneInprocValue(result, target, methodInfo.ResultContainsBinaryType())
	return reflect.ValueOf(target).Elem().Interface()
}

func cloneInprocValue(source any, target any, containsBinaryType bool) {
	if containsBinaryType {
		vpre.MustNil(cbor.Unmarshal(vcode.MustMarshalCbor(source), target))
		return
	}
	vpre.MustNil(json.Unmarshal(vcode.MustMarshalJson(source), target))
}
