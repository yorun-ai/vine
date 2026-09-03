package spec

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

type Registry struct {
	taskInfoBySkelName            map[string]TaskInfo
	taskInfoByDefaultEmbeddedType map[reflect.Type]TaskInfo
	erDefaultEmbeddedTypes        map[reflect.Type]struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		taskInfoBySkelName:            map[string]TaskInfo{},
		taskInfoByDefaultEmbeddedType: map[reflect.Type]TaskInfo{},
		erDefaultEmbeddedTypes:        map[reflect.Type]struct{}{},
	}
}

var defaultRegistry = NewRegistry()

func GetTriggerInfo(taskSkelName string, triggerSkelName string) (TriggerInfo, bool) {
	return defaultRegistry.GetTriggerInfo(taskSkelName, triggerSkelName)
}

func (r *Registry) GetTriggerInfo(taskSkelName string, triggerSkelName string) (TriggerInfo, bool) {
	taskInfo := r.taskInfoBySkelName[taskSkelName]
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
	defaultRegistry.Register(taskSpec)
}

func (r *Registry) Register(taskSpec *TaskSpec) {
	taskInfo := r.initTaskInfo(taskSpec)

	vpre.CheckNil(r.taskInfoBySkelName[taskInfo.SkelName()], "task %s already registered", taskInfo.SkelName())
	vpre.CheckNil(r.taskInfoByDefaultEmbeddedType[taskInfo.DefaultRunnerType().Elem()], "default runner type %s already registered", taskInfo.DefaultRunnerType())
	vpre.CheckNil(r.taskInfoByDefaultEmbeddedType[taskInfo.DefaultERRunnerType().Elem()], "default er runner type %s already registered", taskInfo.DefaultERRunnerType())

	r.taskInfoBySkelName[taskInfo.SkelName()] = taskInfo
	r.registerDefaultEmbeddedTypes(taskInfo.DefaultRunnerType(), taskInfo, false)
	r.registerDefaultEmbeddedTypes(taskInfo.DefaultERRunnerType(), taskInfo, true)
}

func (r *Registry) initTaskInfo(taskSpec *TaskSpec) *_TaskInfo {
	triggers := make([]TriggerInfo, 0, len(taskSpec.Triggers))
	for _, triggerSpec := range taskSpec.Triggers {
		triggerInfo := &_TriggerInfo{
			name:               triggerSpec.Name,
			skelName:           triggerSpec.SkelName,
			launcherMethodName: triggerSpec.LauncherMethodName,
			runnerMethodName:   triggerSpec.RunnerMethodName,
			argumentsType:      triggerSpec.ArgumentsType,
			argumentsSensitive: triggerSpec.ArgumentsSensitive,
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

func (r *Registry) registerDefaultEmbeddedTypes(defaultRunnerType reflect.Type, taskInfo TaskInfo, isERType bool) {
	embeddedType := defaultRunnerType.Elem()
	r.taskInfoByDefaultEmbeddedType[embeddedType] = taskInfo
	if isERType {
		r.erDefaultEmbeddedTypes[embeddedType] = struct{}{}
	}
}

func getTaskInfo(implType reflect.Type) (TaskInfo, bool) {
	return defaultRegistry.getTaskInfo(implType)
}

func (r *Registry) getTaskInfo(implType reflect.Type) (TaskInfo, bool) {
	var taskInfo TaskInfo
	isERType := false
	for _, embeddedType := range reflectutil.EmbeddedStructTypes(implType) {
		info := r.taskInfoByDefaultEmbeddedType[embeddedType]
		if info == nil {
			continue
		}
		vpre.CheckNil(taskInfo, "multiple embedded default runner type found on %s.%s", implType.PkgPath(), implType.Name())
		taskInfo = info
		_, isERType = r.erDefaultEmbeddedTypes[embeddedType]
	}
	vpre.CheckNotNil(taskInfo, "no embedded default runner type found on %s.%s", implType.PkgPath(), implType.Name())
	return taskInfo, isERType
}

func RegisteredTaskLauncherFactories() []any {
	return defaultRegistry.RegisteredTaskLauncherFactories()
}

func (r *Registry) RegisteredTaskLauncherFactories() []any {
	var factories []any
	for _, taskInfo := range r.taskInfoBySkelName {
		factories = append(factories, taskInfo.LauncherCtor())
	}
	return factories
}
