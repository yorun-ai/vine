package task

import (
	"reflect"

	"go.yorun.ai/vine/core/di"
	internaltask "go.yorun.ai/vine/internal/core/task"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
)

// LauncherOption configures a task launcher and its transport.
type LauncherOption = internaltask.LauncherOption

// LaunchOption configures one task launch.
type LaunchOption = internaltask.LaunchOption

// Launcher starts registered tasks.
type Launcher = internaltask.Launcher

// ServerOption configures a task runner server.
type ServerOption = internaltask.Option

// Executor invokes a task runner.
type Executor = internaltask.Executor

// Server receives task runs and dispatches them to an Executor.
type Server = internaltask.Server

// Context carries metadata for one task execution.
type Context = taskspec.Context

// Run is the transport envelope for a task execution.
type Run = taskspec.Run

// TaskSpec describes a generated task contract.
type TaskSpec = taskspec.TaskSpec

// TriggerSpec describes a generated task trigger contract.
type TriggerSpec = taskspec.TriggerSpec

// TaskInfo is runtime metadata derived from a TaskSpec.
type TaskInfo = taskspec.TaskInfo

// TriggerInfo is runtime metadata derived from a TriggerSpec.
type TriggerInfo = taskspec.TriggerInfo

// NewLauncher creates a task launcher from option.
func NewLauncher(option LauncherOption) *Launcher {
	return internaltask.NewLauncher(option)
}

// NewServer creates a task runner server from option.
func NewServer(option ServerOption) *Server {
	return internaltask.NewServer(option)
}

// NewContainerExecutor creates an Executor backed by a DI container and filter chain.
func NewContainerExecutor(filterTypes []reflect.Type, bindAppliers []di.BindApplier) Executor {
	return internaltask.NewContainerExecutor(filterTypes, bindAppliers)
}

// Register adds taskSpec to the process-wide task registry.
func Register(taskSpec *TaskSpec) {
	taskspec.Register(taskSpec)
}
