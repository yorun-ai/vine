package impl

import (
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/task"
)

type TaskServiceServerImpl struct {
	skeled.DefaultTaskServiceServer

	Manager *task.Manager `inject:""`
}

func (s *TaskServiceServerImpl) LaunchTask(launch skeled.TaskLaunch) {
	s.Manager.LaunchTask(launch)
}
