package task_test

import (
	"reflect"

	"go.yorun.ai/vine/core/di"
	"go.yorun.ai/vine/core/task"
)

var (
	_ task.LauncherOption
	_ task.ServerOption
	_ func([]reflect.Type, []di.BindApplier) task.Executor = task.NewContainerExecutor
)
