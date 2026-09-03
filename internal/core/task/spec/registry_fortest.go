package spec

func ConvertSpecToInfoForTest(taskSpec *TaskSpec) TaskInfo {
	return NewRegistry().initTaskInfo(taskSpec)
}
