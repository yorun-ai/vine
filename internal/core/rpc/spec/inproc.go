package spec

// Inproc Rpc isolates request and response values so mutations cannot cross the
// caller/handler boundary. It does not guarantee transport encoding,
// normalization, custom marshaling, or codec failure behavior.

// CloneInprocRequestArguments returns arguments isolated from caller-owned
// mutable values.
func CloneInprocRequestArguments(arguments any, methodInfo MethodInfo) any {
	if methodInfo.ArgumentsType() == nil || arguments == nil {
		return arguments
	}
	return methodInfo.CloneArguments(arguments)
}

// CloneInprocResponseResult returns a result isolated from handler-owned mutable
// values.
func CloneInprocResponseResult(result any, methodInfo MethodInfo) any {
	if methodInfo.ResultType() == nil || result == nil {
		return result
	}
	return methodInfo.CloneResult(result)
}
