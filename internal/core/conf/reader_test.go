package conf

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	corelink "go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/skel"
)

type readerTestConfig struct {
	ConfigModel
	Name string `json:"name"`
}

type readerTestInstantConfig struct {
	ConfigModel
	Name string `json:"name"`
}

type readerTestEnum string

type readerTrimConfig struct {
	ConfigModel
	Name     string               `json:"name"`
	Blank    string               `json:"blank"`
	Optional *string              `json:"optional"`
	Missing  *string              `json:"missing"`
	Items    *[]*string           `json:"items"`
	Values   *map[string]*string  `json:"values"`
	Labels   []string             `json:"labels"`
	Headers  map[string]string    `json:"headers"`
	Empty    []string             `json:"empty"`
	NilItems []string             `json:"nilItems"`
	NilMap   map[string]string    `json:"nilMap"`
	JSON     skel.JSON            `json:"json"`
	JSONs    []skel.JSON          `json:"jsons"`
	JSONMap  map[string]skel.JSON `json:"jsonMap"`
	Enum     readerTestEnum       `json:"enum"`
	Count    int                  `json:"count"`
	Enabled  bool                 `json:"enabled"`
}

func TestReaderTrimsStringsWithoutChangingOtherValuesOrSnapshots(t *testing.T) {
	const key = "demo.ReaderTrimConfig"
	const raw = `{
		"name":"\u2003 hello  world\ninside \u00a0",
		"blank":" \t\r\n", "optional":" optional ", "missing":null,
		"items":[" first ",null,"\t"],
		"values":{" key ":" value ","key":" other ","nil":null},
		"labels":[" label "],"headers":{" key ":" value "},
		"empty":[],"nilItems":null,"nilMap":null,
		"json":"  {\"text\":\" keep \"}  ",
		"jsons":["  {}  "],"jsonMap":{" key ":"  []  "},
		"enum":" unchanged ","count":42,"enabled":true
	}`
	for _, lifecycle := range []Lifecycle{LifecycleEternal, LifecycleInstant} {
		t.Run(string(lifecycle), func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(ConfigSpec{
				Name: "ReaderTrimConfig", SkelName: key, Lifecycle: lifecycle,
				Type: reflect.TypeFor[*readerTrimConfig](),
			})
			linker := &corelink.TestLinker{
				EternalConfigByKey: map[string]string{key: raw},
				InstantConfigByKey: map[string]string{key: raw},
			}
			reader := newReader(linker, registry)
			value := reader.GetByType(reflect.TypeFor[*readerTrimConfig]()).(*readerTrimConfig)
			require.Equal(t, "hello  world\ninside", value.Name)
			require.Empty(t, value.Blank)
			require.Equal(t, "optional", *value.Optional)
			require.Nil(t, value.Missing)
			require.Equal(t, []*string{new("first"), nil, new("")}, *value.Items)
			require.Equal(t, map[string]*string{" key ": new("value"), "key": new("other"), "nil": nil}, *value.Values)
			require.Equal(t, []string{"label"}, value.Labels)
			require.Equal(t, map[string]string{" key ": "value"}, value.Headers)
			require.NotNil(t, value.Empty)
			require.Empty(t, value.Empty)
			require.Nil(t, value.NilItems)
			require.Nil(t, value.NilMap)
			require.Equal(t, skel.JSON(`  {"text":" keep "}  `), value.JSON)
			require.Equal(t, []skel.JSON{"  {}  "}, value.JSONs)
			require.Equal(t, map[string]skel.JSON{" key ": "  []  "}, value.JSONMap)
			require.Equal(t, readerTestEnum(" unchanged "), value.Enum)
			require.Equal(t, 42, value.Count)
			require.True(t, value.Enabled)
			require.Equal(t, raw, linker.EternalConfigByKey[key])
			require.Equal(t, raw, linker.InstantConfigByKey[key])
			*(*value.Items)[0] = "mutated"
			*(*value.Values)[" key "] = "mutated"
			next := reader.GetByType(reflect.TypeFor[*readerTrimConfig]()).(*readerTrimConfig)
			require.Equal(t, "first", *(*next.Items)[0])
			require.Equal(t, "value", *(*next.Values)[" key "])
		})
	}
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
