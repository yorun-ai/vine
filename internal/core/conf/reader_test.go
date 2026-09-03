package conf

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
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
	registry := NewRegistry()

	registry.Register(ConfigSpec{
		Name:      "ReaderTestConfig",
		SkelName:  "demo.user.ReaderTestConfig",
		Lifecycle: LifecycleEternal,
		Type:      reflect.TypeFor[*readerTestConfig](),
	})

	reader := newReader(&corelink.TestLinker{
		EternalConfigByKey: map[string]string{
			"demo.user.ReaderTestConfig": `{"name":"demo"}`,
		},
	}, registry)

	value, ok := reader.GetByType(reflect.TypeFor[*readerTestConfig]()).(*readerTestConfig)
	if !ok {
		t.Fatal("expected reader to return *readerTestConfig")
	}
	if value.Name != "demo" {
		t.Fatalf("unexpected decoded name: %q", value.Name)
	}
}

func TestReaderGetByTypeUsesLocalLifecycle(t *testing.T) {
	registry := NewRegistry()

	registry.Register(ConfigSpec{
		Name:      "ReaderTestInstantConfig",
		SkelName:  "demo.user.ReaderTestInstantConfig",
		Lifecycle: LifecycleInstant,
		Type:      reflect.TypeFor[*readerTestInstantConfig](),
	})

	reader := newReader(&corelink.TestLinker{
		EternalConfigByKey: map[string]string{
			"demo.user.ReaderTestInstantConfig": `{"name":"eternal"}`,
		},
		InstantConfigByKey: map[string]string{
			"demo.user.ReaderTestInstantConfig": `{"name":"instant"}`,
		},
	}, registry)

	value, ok := reader.GetByType(reflect.TypeFor[*readerTestInstantConfig]()).(*readerTestInstantConfig)
	if !ok {
		t.Fatal("expected reader to return *readerTestInstantConfig")
	}
	if value.Name != "instant" {
		t.Fatalf("unexpected decoded name: %q", value.Name)
	}
}

func TestReaderGetByTypePanicsWhenConfigJSONIsEmpty(t *testing.T) {
	registry := NewRegistry()

	registry.Register(ConfigSpec{
		Name:      "ReaderTestConfig",
		SkelName:  "demo.user.ReaderTestConfig",
		Lifecycle: LifecycleEternal,
		Type:      reflect.TypeFor[*readerTestConfig](),
	})

	reader := newReader(&corelink.TestLinker{
		EternalConfigByKey: map[string]string{
			"demo.user.ReaderTestConfig": "",
		},
	}, registry)

	require.PanicsWithError(t, "config demo.user.ReaderTestConfig json is empty", func() {
		reader.GetByType(reflect.TypeFor[*readerTestConfig]())
	})
}

func TestReaderGetByTypePanicsWhenConfigJSONIsInvalid(t *testing.T) {
	registry := NewRegistry()

	registry.Register(ConfigSpec{
		Name:      "ReaderTestConfig",
		SkelName:  "demo.user.ReaderTestConfig",
		Lifecycle: LifecycleEternal,
		Type:      reflect.TypeFor[*readerTestConfig](),
	})

	reader := newReader(&corelink.TestLinker{
		EternalConfigByKey: map[string]string{
			"demo.user.ReaderTestConfig": `{"name":`,
		},
	}, registry)

	require.PanicsWithError(t, `unmarshal config demo.user.ReaderTestConfig failed: jsontext: unexpected EOF within "/name" after offset 8`, func() {
		reader.GetByType(reflect.TypeFor[*readerTestConfig]())
	})
}
