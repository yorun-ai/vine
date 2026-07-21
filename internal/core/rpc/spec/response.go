package spec

import (
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
)

type Response interface {
	Server() meta.App

	Method() MethodInfo
	Result() any
	Error() ex.Error
}

type ResponseImpl struct {
	ServerValue meta.App

	MethodValue MethodInfo
	ResultValue any
	ErrorValue  ex.Error
}

func (r *ResponseImpl) Server() meta.App {
	return r.ServerValue
}

func (r *ResponseImpl) Method() MethodInfo {
	return r.MethodValue
}

func (r *ResponseImpl) Result() any {
	return r.ResultValue
}

func (r *ResponseImpl) Error() ex.Error {
	return r.ErrorValue
}
