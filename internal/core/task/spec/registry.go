package spec

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

var taskInfoBySkelName = map[string]TaskInfo{}
var taskInfoByDefaultEmbeddedType = map[reflect.Type]TaskInfo{}
var erDefaultEmbeddedTypes = map[reflect.Type]struct{}{}

func GetTriggerInfo(taskSkelName string, triggerSkelName string) (TriggerInfo, bool) {
	taskInfo := taskInfoBySkelName[taskSkelName]
	if taskInfo == nil {
		return nil, false
	}
	for _, triggerInfo := range taskInfo.Triggers() {
		if triggerInfo.SkelName() == triggerSkelName {
			return triggerInfo, true
		}
	}
	return nil, false
}

func Register(taskSpec *TaskSpec) {
	taskInfo := initTaskInfo(taskSpec)

	vpre.CheckNil(taskInfoBySkelName[taskInfo.SkelName()], "task %s already registered", taskInfo.SkelName())
	vpre.CheckNil(taskInfoByDefaultEmbeddedType[taskInfo.DefaultRunnerType().Elem()], "default runner type %s already registered", taskInfo.DefaultRunnerType())
	vpre.CheckNil(taskInfoByDefaultEmbeddedType[taskInfo.DefaultERRunnerType().Elem()], "default er runner type %s already registered", taskInfo.DefaultERRunnerType())

	taskInfoBySkelName[taskInfo.SkelName()] = taskInfo
	registerDefaultEmbeddedTypes(taskInfo.DefaultRunnerType(), taskInfo, false)
	registerDefaultEmbeddedTypes(taskInfo.DefaultERRunnerType(), taskInfo, true)
}

func initTaskInfo(taskSpec *TaskSpec) *_TaskInfo {
	triggers := make([]TriggerInfo, 0, len(taskSpec.Triggers))
	for _, triggerSpec := range taskSpec.Triggers {
		triggerInfo := &_TriggerInfo{
			name:               triggerSpec.Name,
			skelName:           triggerSpec.SkelName,
			launcherMethodName: triggerSpec.LauncherMethodName,
			runnerMethodName:   triggerSpec.RunnerMethodName,
			argumentsType:      triggerSpec.ArgumentsType,
		}
		if triggerSpec.ArgumentsType != nil {
			triggerInfo.argumentFieldInfos = buildArgumentFieldInfos(triggerSpec.ArgumentsType)
		}
		triggerSpec.info = triggerInfo
		triggers = append(triggers, triggerInfo)
	}

	taskInfo := &_TaskInfo{
		name:                taskSpec.Name,
		skelName:            taskSpec.SkelName,
		hash:                taskSpec.Hash,
		runnerType:          taskSpec.RunnerType,
		defaultRunnerType:   taskSpec.DefaultRunnerType,
		erRunnerType:        taskSpec.ERRunnerType,
		defaultERRunnerType: taskSpec.DefaultERRunnerType,
		wrapperERRunnerCtor: taskSpec.WrapperERRunnerCtor,
		launcherType:        taskSpec.LauncherType,
		launcherCtor:        taskSpec.LauncherCtor,
		triggers:            triggers,
	}
	taskInfo.initTriggerInfos()
	taskSpec.info = taskInfo
	return taskInfo
}

func registerDefaultEmbeddedTypes(defaultRunnerType reflect.Type, taskInfo TaskInfo, isERType bool) {
	embeddedType := defaultRunnerType.Elem()
	taskInfoByDefaultEmbeddedType[embeddedType] = taskInfo
	if isERType {
		erDefaultEmbeddedTypes[embeddedType] = struct{}{}
	}
}

func getTaskInfo(implType reflect.Type) (TaskInfo, bool) {
	var taskInfo TaskInfo
	isERType := false
	for _, embeddedType := range reflectutil.EmbeddedStructTypes(implType) {
		info := taskInfoByDefaultEmbeddedType[embeddedType]
		if info == nil {
			continue
		}
		vpre.CheckNil(taskInfo, "multiple embedded default runner type found on %s.%s", implType.PkgPath(), implType.Name())
		taskInfo = info
		_, isERType = erDefaultEmbeddedTypes[embeddedType]
	}
	vpre.CheckNotNil(taskInfo, "no embedded default runner type found on %s.%s", implType.PkgPath(), implType.Name())
	return taskInfo, isERType
}

func RegisteredTaskLauncherFactories() []any {
	var factories []any
	for _, taskInfo := range taskInfoBySkelName {
		factories = append(factories, taskInfo.LauncherCtor())
	}
	return factories
}
