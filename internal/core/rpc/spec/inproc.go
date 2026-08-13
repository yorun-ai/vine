package spec

// Inproc Rpc clones request and response values to keep network Rpc mutation
// semantics. Without this, server-side writes to slice/map/pointer fields could
// leak back into caller-owned arguments.

func CloneInprocRequestArguments(arguments any, methodInfo MethodInfo) any {
	if methodInfo.ArgumentsType() == nil || arguments == nil {
		return arguments
	}
	return methodInfo.CloneArguments(arguments)
}

func CloneInprocResponseResult(result any, methodInfo MethodInfo) any {
	if methodInfo.ResultType() == nil || result == nil {
		return result
	}
	return methodInfo.CloneResult(result)
}
