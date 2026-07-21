package conf

import (
	"reflect"
	"testing"
)

type regTestConfig struct {
	Name string `json:"name"`
}

type regTestConfigContract interface {
	Name() string
}

func TestRegisterEternal(t *testing.T) {
	resetRegistryForTest()

	Register(ConfigSpec{
		Name:      "RegTestConfig",
		SkelName:  "demo.user.RegTestConfig",
		Lifecycle: LifecycleEternal,
		Type:      reflect.TypeFor[regTestConfig](),
	})

	reg, ok := infoBySkelName["demo.user.RegTestConfig"]
	if !ok {
		t.Fatal("expected registration")
	}
	if reg.Name != "RegTestConfig" {
		t.Fatalf("unexpected name: %s", reg.Name)
	}
	if reg.Type != reflect.TypeFor[regTestConfig]() {
		t.Fatalf("unexpected mapped type: %v", reg.Type)
	}
	if reg.Lifecycle != LifecycleEternal {
		t.Fatalf("unexpected lifecycle: %s", reg.Lifecycle)
	}
	if reg.SkelName != "demo.user.RegTestConfig" {
		t.Fatalf("unexpected skelName: %s", reg.SkelName)
	}
	types := RegisteredTypes()
	if len(types) != 1 {
		t.Fatalf("unexpected registered types count: %d", len(types))
	}
	if types[0] != reflect.TypeFor[regTestConfig]() {
		t.Fatalf("unexpected registered type: %v", types[0])
	}
}

func TestRegisterInstant(t *testing.T) {
	resetRegistryForTest()

	Register(ConfigSpec{
		Name:      "RegTestConfig",
		SkelName:  "demo.user.RegTestConfig",
		Lifecycle: LifecycleInstant,
		Type:      reflect.TypeFor[regTestConfig](),
	})

	reg, ok := infoBySkelName["demo.user.RegTestConfig"]
	if !ok {
		t.Fatal("expected registration")
	}
	if reg.Lifecycle != LifecycleInstant {
		t.Fatalf("unexpected lifecycle: %s", reg.Lifecycle)
	}
}

func TestRegisterInterfaceWithConstructor(t *testing.T) {
	resetRegistryForTest()

	Register(ConfigSpec{
		Name:      "RegTestConfigContract",
		SkelName:  "demo.user.RegTestConfigContract",
		Lifecycle: LifecycleInstant,
		Type:      reflect.TypeFor[regTestConfigContract](),
	})

	reg, ok := infoBySkelName["demo.user.RegTestConfigContract"]
	if !ok {
		t.Fatal("expected registration")
	}
	if reg.Type != reflect.TypeFor[regTestConfigContract]() {
		t.Fatalf("unexpected mapped type: %v", reg.Type)
	}
}

func TestRegisterRejectsDuplicateSkelName(t *testing.T) {
	resetRegistryForTest()

	Register(ConfigSpec{
		Name:      "RegTestConfig",
		SkelName:  "demo.user.RegTestConfig",
		Lifecycle: LifecycleInstant,
		Type:      reflect.TypeFor[*regTestConfig](),
	})

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration panic")
		}
	}()

	Register(ConfigSpec{
		Name:      "RegTestConfig",
		SkelName:  "demo.user.RegTestConfig",
		Lifecycle: LifecycleInstant,
		Type:      reflect.TypeFor[*regTestConfig](),
	})
}

func resetRegistryForTest() {
	infoBySkelName = map[string]*_ConfigInfo{}
	infoByType = map[reflect.Type]*_ConfigInfo{}
}
