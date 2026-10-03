// Package spectest initializes contract fixtures through isolated production registries.
package spectest

import (
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
)

// RpcService registers a service locally and returns the initialized method's service.
func RpcService(service *rpcspec.ServiceSpec) rpcspec.ServiceInfo {
	registry := rpcspec.NewRegistry()
	registry.Register(service)
	return service.Methods[0].Info().Service()
}

// Task registers a task locally and returns its initialized information.
func Task(task *taskspec.TaskSpec) taskspec.TaskInfo {
	registry := taskspec.NewRegistry()
	registry.Register(task)
	return task.Info()
}
