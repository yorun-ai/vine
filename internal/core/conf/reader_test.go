package conf

import (
	"reflect"
	"testing"

	corelink "go.yorun.ai/vine/internal/core/link"
)

type readerTestConfig struct {
	ConfigModel
	Name string `json:"name"`
}

type readerTestInstantConfig struct {
	ConfigModel
	Name string `json:"name"`
}

func TestReaderGetByTypeDecodesLinkConfig(t *testing.T) {
	resetRegistryForTest()

	Register(ConfigSpec{
		Name:      "ReaderTestConfig",
		SkelName:  "demo.user.ReaderTestConfig",
		Lifecycle: LifecycleEternal,
		Type:      reflect.TypeFor[*readerTestConfig](),
	})

	reader := NewReader(&corelink.TestLinker{
		EternalConfigByKey: map[string]string{
			"demo.user.ReaderTestConfig": `{"name":"demo"}`,
		},
	})

	value, ok := reader.GetByType(reflect.TypeFor[*readerTestConfig]()).(*readerTestConfig)
	if !ok {
		t.Fatal("expected reader to return *readerTestConfig")
	}
	if value.Name != "demo" {
		t.Fatalf("unexpected decoded name: %q", value.Name)
	}
}

func TestReaderGetByTypeUsesLocalLifecycle(t *testing.T) {
	resetRegistryForTest()

	Register(ConfigSpec{
		Name:      "ReaderTestInstantConfig",
		SkelName:  "demo.user.ReaderTestInstantConfig",
		Lifecycle: LifecycleInstant,
		Type:      reflect.TypeFor[*readerTestInstantConfig](),
	})

	reader := NewReader(&corelink.TestLinker{
		EternalConfigByKey: map[string]string{
			"demo.user.ReaderTestInstantConfig": `{"name":"eternal"}`,
		},
		InstantConfigByKey: map[string]string{
			"demo.user.ReaderTestInstantConfig": `{"name":"instant"}`,
		},
	})

	value, ok := reader.GetByType(reflect.TypeFor[*readerTestInstantConfig]()).(*readerTestInstantConfig)
	if !ok {
		t.Fatal("expected reader to return *readerTestInstantConfig")
	}
	if value.Name != "instant" {
		t.Fatalf("unexpected decoded name: %q", value.Name)
	}
}
