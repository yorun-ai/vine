package task_test

import (
	"testing"

	"go.yorun.ai/vine/core/task"
)

func TestFacadeCreatesContainerExecutor(t *testing.T) {
	var executor task.Executor = task.NewContainerExecutor(nil, nil)
	if executor == nil {
		t.Fatal("expected container executor")
	}

	_ = task.LauncherOption{}
	_ = task.ServerOption{}
}
