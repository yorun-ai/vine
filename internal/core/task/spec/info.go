package spec

import (
	"fmt"
	"reflect"

	"go.yorun.ai/vine/util/vpre"
)

type TaskInfo interface {
	Name() string
	SkelName() string
	Hash() string
	RunnerType() reflect.Type
	ERRunnerType() reflect.Type
	DefaultRunnerType() reflect.Type
	DefaultERRunnerType() reflect.Type
	WrapperERRunnerCtor() any
	LauncherType() reflect.Type
	LauncherCtor() any
	Triggers() []TriggerInfo
}

type TriggerInfo interface {
	Name() string
	SkelName() string
	LauncherMethodName() string
	RunnerMethodName() string
	ArgumentsType() reflect.Type
	Task() TaskInfo
	HasArguments() bool
	NewArguments() any
	PositionArguments(arguments any) []any
	ValidateArguments(arguments any) error
}

type _emptyArguments struct{}

type TaskSpec struct {
	Name     string
	SkelName string
	Hash     string

	RunnerType          reflect.Type
	DefaultRunnerType   reflect.Type
	ERRunnerType        reflect.Type
	WrapperERRunnerCtor any
	DefaultERRunnerType reflect.Type
	LauncherType        reflect.Type
	LauncherCtor        any

	Triggers []*TriggerSpec

	info *_TaskInfo
}

func (s *TaskSpec) Info() TaskInfo {
	return s.info
}

type TriggerSpec struct {
	Name               string
	SkelName           string
	LauncherMethodName string
	RunnerMethodName   string

	ArgumentsType reflect.Type

	info *_TriggerInfo
}

func (s *TriggerSpec) Info() TriggerInfo {
	return s.info
}

type _TaskInfo struct {
	name     string
	skelName string
	hash     string

	runnerType          reflect.Type
	defaultRunnerType   reflect.Type
	erRunnerType        reflect.Type
	defaultERRunnerType reflect.Type
	wrapperERRunnerCtor any
	launcherType        reflect.Type
	launcherCtor        any

	triggers []TriggerInfo
}

func (ti *_TaskInfo) initTriggerInfos() {
	for _, triggerInfo := range ti.triggers {
		triggerInfo.(*_TriggerInfo).task = ti
	}
}

func (ti *_TaskInfo) Name() string {
	return ti.name
}

func (ti *_TaskInfo) SkelName() string {
	return ti.skelName
}

func (ti *_TaskInfo) Hash() string {
	return ti.hash
}

func (ti *_TaskInfo) RunnerType() reflect.Type {
	return ti.runnerType
}

func (ti *_TaskInfo) ERRunnerType() reflect.Type {
	return ti.erRunnerType
}

func (ti *_TaskInfo) DefaultRunnerType() reflect.Type {
	return ti.defaultRunnerType
}

func (ti *_TaskInfo) DefaultERRunnerType() reflect.Type {
	return ti.defaultERRunnerType
}

func (ti *_TaskInfo) WrapperERRunnerCtor() any {
	return ti.wrapperERRunnerCtor
}

func (ti *_TaskInfo) LauncherType() reflect.Type {
	return ti.launcherType
}

func (ti *_TaskInfo) LauncherCtor() any {
	return ti.launcherCtor
}

func (ti *_TaskInfo) Triggers() []TriggerInfo {
	return append([]TriggerInfo(nil), ti.triggers...)
}

type _TriggerInfo struct {
	name               string
	skelName           string
	launcherMethodName string
	runnerMethodName   string

	argumentsType      reflect.Type
	argumentFieldInfos []_ArgumentFieldInfo

	task TaskInfo
}

func (ti *_TriggerInfo) Name() string {
	return ti.name
}

func (ti *_TriggerInfo) SkelName() string {
	return ti.skelName
}

func (ti *_TriggerInfo) LauncherMethodName() string {
	return ti.launcherMethodName
}

func (ti *_TriggerInfo) RunnerMethodName() string {
	return ti.runnerMethodName
}

func (ti *_TriggerInfo) ArgumentsType() reflect.Type {
	return ti.argumentsType
}

func (ti *_TriggerInfo) Task() TaskInfo {
	return ti.task
}

func (ti *_TriggerInfo) HasArguments() bool {
	return ti.argumentsType != nil
}

func (ti *_TriggerInfo) NewArguments() any {
	if !ti.HasArguments() {
		return &_emptyArguments{}
	}
	return reflect.New(ti.argumentsType).Interface()
}

func (ti *_TriggerInfo) PositionArguments(arguments any) []any {
	if !ti.HasArguments() {
		return nil
	}

	argsValue := reflect.ValueOf(arguments).Elem()
	positionalArguments := make([]any, len(ti.argumentFieldInfos))
	for _, argFieldInfo := range ti.argumentFieldInfos {
		positionalArguments[argFieldInfo.ArgIndex] = argsValue.Field(argFieldInfo.FieldIndex).Interface()
	}
	return positionalArguments
}

func (ti *_TriggerInfo) ValidateArguments(arguments any) error {
	if arguments == nil {
		return fmt.Errorf("arguments of %s cannot be nil", ti.name)
	}
	if !ti.HasArguments() {
		return nil
	}

	argsValue := reflect.ValueOf(arguments).Elem()
	for _, argFieldInfo := range ti.argumentFieldInfos {
		argField := argsValue.Field(argFieldInfo.FieldIndex)
		if argFieldInfo.CheckNotNil && argField.IsNil() {
			return fmt.Errorf("unexpected nil value on arg %s of %s", argFieldInfo.Name, ti.name)
		}
	}
	return nil
}

type _ArgumentFieldInfo struct {
	Name        string
	FieldIndex  int
	ArgIndex    int
	CheckNotNil bool
}

func buildArgumentFieldInfos(argumentsType reflect.Type) []_ArgumentFieldInfo {
	vpre.Check(argumentsType.Kind() == reflect.Struct, "task arguments type must be a struct, got %s", argumentsType)

	infos := make([]_ArgumentFieldInfo, 0, argumentsType.NumField())
	for fieldIndex := 0; fieldIndex < argumentsType.NumField(); fieldIndex++ {
		field := argumentsType.Field(fieldIndex)
		infos = append(infos, _ArgumentFieldInfo{
			Name:        field.Name,
			FieldIndex:  fieldIndex,
			ArgIndex:    fieldIndex,
			CheckNotNil: field.Type.Kind() == reflect.Ptr,
		})
	}
	return infos
}
