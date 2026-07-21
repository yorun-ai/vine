// Package di provides dependency injection primitives.
//
// Lifecycle ownership:
// Execution-scoped instances are disposed when the corresponding execution is
// completed. Singleton-scoped instances are owned by the application/component
// lifecycle that created the injector; plain injectors do not automatically
// dispose singleton instances.
//
// Binding scope semantics:
// Scopes are attached to binding targets rather than underlying instance
// sources. Forwarding-style bindings such as ToImplementation and factory-based
// bindings therefore define the lifecycle for that requested target only. If
// the same concrete implementation is also requested directly, it is resolved
// through its own binding and may have a different lifecycle unless explicitly
// bound to the same scope or instance source.
//
// For example:
//
//	b.Bind(T[Service]()).ToImplementation(T[*ServiceImpl]()).In(SingletonScope)
//	b.Bind(T[*ServiceImpl]()).In(TransientScope)
//
// Resolving Service uses the singleton binding above, while resolving
// *ServiceImpl directly uses the transient binding.
package di
