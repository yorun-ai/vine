package spec

import (
	"reflect"
	"strings"
	"testing"
)

type testInfoTaskImpl struct {
	defaultTestInfoTaskRunner
}

func (*testInfoTaskImpl) RunForGroup(testInfoTaskArguments) {}

func TestNewImplDictResolvesTaskImplFromEmbeddedDefaultRunner(t *testing.T) {
	ensureInfoTaskRegistered()

	dict := NewImplDict()
	dict.Add(reflect.TypeOf(&testInfoTaskImpl{}))

	triggerImpl, err := dict.GetTriggerImpl("test.info.task", "forGroup")
	if err != nil {
		t.Fatalf("GetTriggerImpl() error = %v", err)
	}
	if triggerImpl.Info().Name() != "ForGroup" {
		t.Fatalf("unexpected trigger impl: %+v", triggerImpl)
	}
	if triggerImpl.Method().Name != "RunForGroup" {
		t.Fatalf("unexpected runner method: %s", triggerImpl.Method().Name)
	}
}

func TestNewImplDictReturnsErrorWhenTriggerMissing(t *testing.T) {
	ensureInfoTaskRegistered()

	dict := NewImplDict()
	dict.Add(reflect.TypeOf(&testInfoTaskImpl{}))

	_, err := dict.GetTriggerImpl("test.info.task", "missing")
	if err == nil {
		t.Fatal("expected missing trigger error")
	}
	if !strings.Contains(err.Error(), "trigger missing not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
