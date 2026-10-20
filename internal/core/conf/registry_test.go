package conf

import (
	"reflect"
	"testing"
)

type regTestConfig struct {
	Name string `json:"name"`
}

func TestRegisterRejectsDuplicateSkelName(t *testing.T) {
	registry := NewRegistry()

	registry.Register(ConfigSpec{
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

	registry.Register(ConfigSpec{
		Name:      "RegTestConfig",
		SkelName:  "demo.user.RegTestConfig",
		Lifecycle: LifecycleInstant,
		Type:      reflect.TypeFor[*regTestConfig](),
	})
}
