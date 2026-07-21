package spec

import (
	"fmt"
	"reflect"

	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type ImplDict struct {
	taskByName    map[string]*_TaskImpl
	triggerByInfo map[TriggerInfo]*_TriggerImpl
}

func NewImplDict() *ImplDict {
	return &ImplDict{
		taskByName:    map[string]*_TaskImpl{},
		triggerByInfo: map[TriggerInfo]*_TriggerImpl{},
	}
}

func (d *ImplDict) Add(implType reflect.Type) {
	taskInfo, isERType := getTaskInfo(implType)
	vpre.CheckNil(d.taskByName[taskInfo.SkelName()], "task %s already added", taskInfo.SkelName())

	taskImpl := &_TaskImpl{
		kind:     implType,
		isERType: isERType,
		info:     taskInfo,
		triggers: map[string]*_TriggerImpl{},
	}
	for _, triggerInfo := range taskInfo.Triggers() {
		method, ok := implType.MethodByName(triggerInfo.RunnerMethodName())
		vpre.Check(ok, "trigger %s not found on %s", triggerInfo.RunnerMethodName(), implType)
		triggerImpl := &_TriggerImpl{
			kind:     implType,
			method:   method,
			isERType: isERType,
			info:     triggerInfo,
		}
		taskImpl.triggers[triggerInfo.SkelName()] = triggerImpl
		d.triggerByInfo[triggerInfo] = triggerImpl
	}
	d.taskByName[taskInfo.SkelName()] = taskImpl
}

func (d *ImplDict) GetTriggerImpl(taskSkelName string, triggerSkelName string) (TriggerImpl, error) {
	taskImpl, ok := d.taskByName[taskSkelName]
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskSkelName)
	}
	return taskImpl.TriggerImpl(triggerSkelName)
}

func (d *ImplDict) GetTriggerImplByInfo(triggerInfo TriggerInfo) (TriggerImpl, error) {
	if triggerInfo == nil {
		return nil, fmt.Errorf("trigger info is nil")
	}

	triggerImpl, ok := d.triggerByInfo[triggerInfo]
	if !ok {
		return nil, fmt.Errorf("trigger %s not found", triggerInfo.SkelName())
	}
	return triggerImpl, nil
}

func (d *ImplDict) IterateTaskImpl(iterate func(info TaskImpl)) {
	vmap.ForEach(d.taskByName, func(_ string, info *_TaskImpl) {
		iterate(info)
	})
}

type TaskImpl interface {
	Type() reflect.Type
	IsERType() bool
	Info() TaskInfo
	TriggerImpl(triggerSkelName string) (TriggerImpl, error)
}

type _TaskImpl struct {
	kind     reflect.Type
	isERType bool
	info     TaskInfo
	triggers map[string]*_TriggerImpl
}

func (i *_TaskImpl) Type() reflect.Type {
	return i.kind
}

func (i *_TaskImpl) IsERType() bool {
	return i.isERType
}

func (i *_TaskImpl) Info() TaskInfo {
	return i.info
}

func (i *_TaskImpl) TriggerImpl(triggerSkelName string) (TriggerImpl, error) {
	triggerImpl, ok := i.triggers[triggerSkelName]
	if !ok {
		return nil, fmt.Errorf("trigger %s not found", triggerSkelName)
	}
	return triggerImpl, nil
}

type TriggerImpl interface {
	Type() reflect.Type
	Method() reflect.Method
	IsERType() bool
	Info() TriggerInfo
}

type _TriggerImpl struct {
	kind     reflect.Type
	method   reflect.Method
	isERType bool
	info     TriggerInfo
}

func (i *_TriggerImpl) Type() reflect.Type {
	return i.kind
}

func (i *_TriggerImpl) Method() reflect.Method {
	return i.method
}

func (i *_TriggerImpl) IsERType() bool {
	return i.isERType
}

func (i *_TriggerImpl) Info() TriggerInfo {
	return i.info
}
