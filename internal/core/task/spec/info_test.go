package spec

import (
	"reflect"
	"testing"
)

func TestConvertSpecToInfoForTestBuildsTriggerInfo(t *testing.T) {
	taskInfo := ConvertSpecToInfoForTest(&TaskSpec{
		Name:                "RebuildTask",
		SkelName:            "demo.RebuildTask",
		RunnerType:          reflect.TypeFor[testInfoTaskRunner](),
		DefaultRunnerType:   reflect.TypeFor[*defaultTestInfoTaskRunner](),
		ERRunnerType:        reflect.TypeFor[testInfoTaskRunnerER](),
		WrapperERRunnerCtor: newWrapperTestInfoTaskRunnerER,
		DefaultERRunnerType: reflect.TypeFor[*defaultTestInfoTaskRunnerER](),
		LauncherType:        reflect.TypeFor[testInfoTaskLauncher](),
		Triggers: []*TriggerSpec{{
			Name:               "AtTime",
			SkelName:           "atTime",
			LauncherMethodName: "LaunchAtTime",
			RunnerMethodName:   "RunAtTime",
			ArgumentsType:      reflect.TypeFor[testInfoTaskArguments](),
			ArgumentsSensitive: true,
		}},
	})

	if taskInfo.Name() != "RebuildTask" || taskInfo.SkelName() != "demo.RebuildTask" {
		t.Fatalf("unexpected task info: %+v", taskInfo)
	}
	if len(taskInfo.Triggers()) != 1 {
		t.Fatalf("unexpected trigger count: %d", len(taskInfo.Triggers()))
	}
	triggerInfo := taskInfo.Triggers()[0]
	if triggerInfo.Task() != taskInfo {
		t.Fatalf("unexpected trigger task: %+v", triggerInfo.Task())
	}
	if triggerInfo.Name() != "AtTime" {
		t.Fatalf("unexpected trigger name: %s", triggerInfo.Name())
	}
	if triggerInfo.LauncherMethodName() != "LaunchAtTime" {
		t.Fatalf("unexpected launcher method name: %s", triggerInfo.LauncherMethodName())
	}
	if triggerInfo.RunnerMethodName() != "RunAtTime" {
		t.Fatalf("unexpected runner method name: %s", triggerInfo.RunnerMethodName())
	}
	if !triggerInfo.ArgumentsSensitive() {
		t.Fatal("expected trigger arguments to be sensitive")
	}
}

func TestTriggerNullableArguments(t *testing.T) {
	type arguments struct {
		Note   *string
		Items  *[]string
		Labels *map[string]string
	}
	trigger := &_TriggerInfo{name: "Run", argumentsType: reflect.TypeFor[arguments](), argumentFieldInfos: buildArgumentFieldInfos(reflect.TypeFor[arguments]())}
	emptyItems, emptyLabels := []string{}, map[string]string{}
	note, items, labels := "note", []string{"item"}, map[string]string{"key": "value"}
	for _, args := range []*arguments{{}, {Items: &emptyItems, Labels: &emptyLabels}, {Note: &note, Items: &items, Labels: &labels}} {
		if err := trigger.ValidateArguments(args); err != nil {
			t.Fatalf("valid nullable arguments rejected: %v", err)
		}
		want := []any{args.Note, args.Items, args.Labels}
		if got := trigger.PositionArguments(args); !reflect.DeepEqual(got, want) {
			t.Fatalf("arguments changed: got %#v want %#v", got, want)
		}
	}
}

func TestTriggerRejectsNilArgumentObject(t *testing.T) {
	type arguments struct{ Note *string }
	trigger := &_TriggerInfo{name: "Run", argumentsType: reflect.TypeFor[arguments]()}
	for _, args := range []any{nil, (*arguments)(nil)} {
		if err := trigger.ValidateArguments(args); err == nil {
			t.Fatal("nil argument object must be rejected")
		}
	}
}
