package spec

import (
	"reflect"
	"testing"
)

func TestConvertSpecToInfoForTestBuildsTriggerInfo(t *testing.T) {
	taskInfo := ConvertSpecToInfoForTest(&TaskSpec{
		Name:                "RebuildTask",
		SkelName:            "demo.RebuildTask",
		RunnerType:          reflect.TypeOf((*testInfoTaskRunner)(nil)).Elem(),
		DefaultRunnerType:   reflect.TypeOf(&defaultTestInfoTaskRunner{}),
		ERRunnerType:        reflect.TypeOf((*testInfoTaskRunnerER)(nil)).Elem(),
		WrapperERRunnerCtor: newWrapperTestInfoTaskRunnerER,
		DefaultERRunnerType: reflect.TypeOf(&defaultTestInfoTaskRunnerER{}),
		LauncherType:        reflect.TypeOf((*testInfoTaskLauncher)(nil)).Elem(),
		Triggers: []*TriggerSpec{{
			Name:               "AtTime",
			SkelName:           "atTime",
			LauncherMethodName: "LaunchAtTime",
			RunnerMethodName:   "RunAtTime",
			ArgumentsType:      reflect.TypeOf(testInfoTaskArguments{}),
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
}
