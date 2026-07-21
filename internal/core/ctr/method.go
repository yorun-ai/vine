package ctr

import (
	"reflect"

	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type _MethodCacheKey struct {
	kind reflect.Type
	name string
}

var methodCache = vmap.NewMutexMap[_MethodCacheKey, reflect.Method]()

func getMethodByName(kind reflect.Type, name string) reflect.Method {
	key := _MethodCacheKey{kind: kind, name: name}
	if method, ok := methodCache.Load(key); ok {
		return method
	}

	method, ok := kind.MethodByName(name)
	vpre.Check(ok, "method=%s not found in type=%s", name, kind.Name())

	cached, _ := methodCache.LoadOrStore(key, method)
	return cached
}
