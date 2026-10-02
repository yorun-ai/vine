package conf

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	corelink "go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/redact"
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

type readerSensitiveConfig struct {
	ConfigModel
	Plain    string              `json:"plain"`
	Secret   string              `json:"secret" skel:"sensitive"`
	Password string              `json:"password" skel:"sensitive"`
	Optional *string             `json:"optional"`
	Missing  *string             `json:"missing"`
	Items    *[]*string          `json:"items" skel:"sensitive"`
	Values   *map[string]*string `json:"values"`
	Empty    []string            `json:"empty"`
	NilItems []string            `json:"nilItems"`
}

func TestReaderPreservesSensitiveStringValues(t *testing.T) {
	const key = "demo.SensitiveConfig"
	const raw = `{"plain":" plain ","secret":" secret ","password":"\u2003 password \n","optional":" \t ","missing":null,"items":[" item ",null],"values":{" key ":" value ","nil":null},"empty":[],"nilItems":null}`
	for _, lifecycle := range []Lifecycle{LifecycleEternal, LifecycleInstant} {
		t.Run(string(lifecycle), func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(ConfigSpec{SkelName: key, Lifecycle: lifecycle, Type: reflect.TypeFor[*readerSensitiveConfig]()})
			reader := newReader(&corelink.TestLinker{
				EternalConfigByKey: map[string]string{key: raw}, InstantConfigByKey: map[string]string{key: raw},
			}, registry)
			value := reader.GetByType(reflect.TypeFor[*readerSensitiveConfig]()).(*readerSensitiveConfig)
			require.Equal(t, " plain ", value.Plain)
			require.Equal(t, " secret ", value.Secret)
			require.Equal(t, "\u2003 password \n", value.Password)
			require.Equal(t, " \t ", *value.Optional)
			require.Nil(t, value.Missing)
			require.Equal(t, []*string{new(" item "), nil}, *value.Items)
			require.Equal(t, map[string]*string{" key ": new(" value "), "nil": nil}, *value.Values)
			require.NotNil(t, value.Empty)
			require.Empty(t, value.Empty)
			require.Nil(t, value.NilItems)
			result, err := redact.Render(value)
			require.NoError(t, err)
			require.Contains(t, result.JSON, `"password":"<redacted>"`)
			require.Contains(t, result.JSON, `"secret":"<redacted>"`)
			require.Contains(t, result.JSON, `"items":"<redacted>"`)
			require.Contains(t, result.JSON, `" key ":" value "`)
		})
	}
}

type readerValueConfig struct {
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

func TestReaderPreservesValuesAndIsolatesSnapshots(t *testing.T) {
	const key = "demo.ReaderValueConfig"
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
				Name: "ReaderValueConfig", SkelName: key, Lifecycle: lifecycle,
				Type: reflect.TypeFor[*readerValueConfig](),
			})
			linker := &corelink.TestLinker{
				EternalConfigByKey: map[string]string{key: raw},
				InstantConfigByKey: map[string]string{key: raw},
			}
			reader := newReader(linker, registry)
			value := reader.GetByType(reflect.TypeFor[*readerValueConfig]()).(*readerValueConfig)
			require.Equal(t, "\u2003 hello  world\ninside \u00a0", value.Name)
			require.Equal(t, " \t\r\n", value.Blank)
			require.Equal(t, " optional ", *value.Optional)
			require.Nil(t, value.Missing)
			require.Equal(t, []*string{new(" first "), nil, new("\t")}, *value.Items)
			require.Equal(t, map[string]*string{" key ": new(" value "), "key": new(" other "), "nil": nil}, *value.Values)
			require.Equal(t, []string{" label "}, value.Labels)
			require.Equal(t, map[string]string{" key ": " value "}, value.Headers)
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
			next := reader.GetByType(reflect.TypeFor[*readerValueConfig]()).(*readerValueConfig)
			require.Equal(t, " first ", *(*next.Items)[0])
			require.Equal(t, " value ", *(*next.Values)[" key "])
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

type readerEnumMapConfig struct {
	ConfigModel
	ByName   map[string]readerTestEnum         `json:"byName"`
	ByStatus map[readerTestEnum]readerTestEnum `json:"byStatus"`
	Labels   map[readerTestEnum]string         `json:"labels"`
}

func TestReaderEnumMapKeysAndValues(t *testing.T) {
	const key = "demo.EnumMapConfig"
	const raw = `{"byName":{"primary":"ACTIVE"},"byStatus":{"ACTIVE":"LOCKED"},"labels":{"ACTIVE":" label "}}`
	for _, lifecycle := range []Lifecycle{LifecycleEternal, LifecycleInstant} {
		t.Run(string(lifecycle), func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(ConfigSpec{
				SkelName: key, Lifecycle: lifecycle, Type: reflect.TypeFor[*readerEnumMapConfig](),
			})
			reader := newReader(new(corelink.TestLinker{
				EternalConfigByKey: map[string]string{key: raw},
				InstantConfigByKey: map[string]string{key: raw},
			}), registry)
			value := reader.GetByType(reflect.TypeFor[*readerEnumMapConfig]()).(*readerEnumMapConfig)
			require.Equal(t, map[string]readerTestEnum{"primary": "ACTIVE"}, value.ByName)
			require.Equal(t, map[readerTestEnum]readerTestEnum{"ACTIVE": "LOCKED"}, value.ByStatus)
			require.Equal(t, map[readerTestEnum]string{"ACTIVE": " label "}, value.Labels)
		})
	}
}

type readerEntry[TValue any] struct {
	Value TValue `json:"value"`
}

type readerNestedData struct {
	Name     string             `json:"name"`
	Token    string             `json:"token" skel:"sensitive"`
	Content  skel.Binary        `json:"content"`
	Children []readerNestedData `json:"children"`
}

type readerWholeSensitiveData struct {
	Value string `json:"value"`
}

func (readerWholeSensitiveData) SkelSensitive() {}

type readerStructuredConfig struct {
	ConfigModel
	Nested     *readerNestedData                           `json:"nested"`
	Groups     map[string][]readerEntry[*readerNestedData] `json:"groups"`
	Payload    skel.Binary                                 `json:"payload"`
	Payloads   []readerEntry[*skel.Binary]                 `json:"payloads"`
	PayloadMap map[string]skel.Binary                      `json:"payloadMap"`
	Whole      readerWholeSensitiveData                    `json:"whole"`
	Private    *readerNestedData                           `json:"private" skel:"sensitive"`
}

func TestReaderStructuredValuesAndRedaction(t *testing.T) {
	const key = "demo.StructuredConfig"
	const raw = `{
 "nested":{"name":" nested ","token":" nested-token ","content":"aGVs\nbG8=","children":[{"name":" child ","content":"","children":[]}]},
 "groups":{" key ":[{"value":{"name":" group ","token":" group-token ","children":[]}}, {"value":null}]},
 "payload":"aGVs\r\nbG8=",
 "payloads":[{"value":"aGVsbG8="},{"value":null}],
 "payloadMap":{" key ":"aGVsbG8=","empty":"","nil":null},
 "whole":{"value":"whole-secret"},
 "private":{"name":"private-secret","children":[]}
}`
	for _, lifecycle := range []Lifecycle{LifecycleEternal, LifecycleInstant} {
		t.Run(string(lifecycle), func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(ConfigSpec{SkelName: key, Lifecycle: lifecycle, Type: reflect.TypeFor[*readerStructuredConfig]()})
			linker := new(corelink.TestLinker{
				EternalConfigByKey: map[string]string{key: raw},
				InstantConfigByKey: map[string]string{key: raw},
			})
			reader := newReader(linker, registry)
			value := reader.GetByType(reflect.TypeFor[*readerStructuredConfig]()).(*readerStructuredConfig)
			require.Equal(t, " nested ", value.Nested.Name)
			require.Equal(t, " nested-token ", value.Nested.Token)
			require.Equal(t, skel.Binary("hello"), value.Nested.Content)
			require.Equal(t, " child ", value.Nested.Children[0].Name)
			require.Empty(t, value.Nested.Children[0].Content)
			require.NotNil(t, value.Nested.Children[0].Children)
			require.Equal(t, " group ", value.Groups[" key "][0].Value.Name)
			require.Nil(t, value.Groups[" key "][1].Value)
			require.Equal(t, skel.Binary("hello"), value.Payload)
			require.Equal(t, skel.Binary("hello"), *value.Payloads[0].Value)
			require.Nil(t, value.Payloads[1].Value)
			require.Equal(t, skel.Binary("hello"), value.PayloadMap[" key "])
			require.Empty(t, value.PayloadMap["empty"])
			require.Nil(t, value.PayloadMap["nil"])

			result, err := redact.Render(value)
			require.NoError(t, err)
			require.Contains(t, result.JSON, `"token":"<redacted>"`)
			require.Contains(t, result.JSON, `"whole":"<redacted>"`)
			require.Contains(t, result.JSON, `"private":"<redacted>"`)
			for _, secret := range []string{"nested-token", "group-token", "whole-secret", "private-secret", "aGVsbG8="} {
				require.NotContains(t, result.JSON, secret)
			}
			require.Contains(t, result.JSON, `"name":" nested "`)

			value.Nested.Children[0].Name = "mutated"
			value.Groups[" key "][0].Value.Name = "mutated"
			value.Payload[0] = 'X'
			(*value.Payloads[0].Value)[0] = 'X'
			value.PayloadMap[" key "][0] = 'X'
			next := reader.GetByType(reflect.TypeFor[*readerStructuredConfig]()).(*readerStructuredConfig)
			require.Equal(t, " child ", next.Nested.Children[0].Name)
			require.Equal(t, " group ", next.Groups[" key "][0].Value.Name)
			require.Equal(t, skel.Binary("hello"), next.Payload)
			require.Equal(t, skel.Binary("hello"), *next.Payloads[0].Value)
			require.Equal(t, skel.Binary("hello"), next.PayloadMap[" key "])
			require.Equal(t, raw, linker.EternalConfigByKey[key])
			require.Equal(t, raw, linker.InstantConfigByKey[key])
		})
	}
}

func TestReaderRejectsInvalidBinary(t *testing.T) {
	const key = "demo.StructuredConfig"
	for _, lifecycle := range []Lifecycle{LifecycleEternal, LifecycleInstant} {
		for _, raw := range []string{
			`{"payload":"aGVs bG8="}`,
			`{"payload":"aGVs\tbG8="}`,
			`{"nested":{"content":"invalid!"}}`,
			`{"payloads":[{"value":"invalid!"}]}`,
			`{"payloadMap":{"key":"invalid!"}}`,
		} {
			t.Run(string(lifecycle)+"/"+raw, func(t *testing.T) {
				registry := NewRegistry()
				registry.Register(ConfigSpec{SkelName: key, Lifecycle: lifecycle, Type: reflect.TypeFor[*readerStructuredConfig]()})
				reader := newReader(new(corelink.TestLinker{
					EternalConfigByKey: map[string]string{key: raw},
					InstantConfigByKey: map[string]string{key: raw},
				}), registry)
				require.Panics(t, func() { reader.GetByType(reflect.TypeFor[*readerStructuredConfig]()) })
			})
		}
	}
}

type readerOptionalEntry[TValue any] struct {
	Value *TValue `json:"value"`
}

type readerOptionalGenericConfig struct {
	ConfigModel
	Binary         []readerOptionalEntry[skel.Binary]  `json:"binary"`
	NullableBinary []readerOptionalEntry[*skel.Binary] `json:"nullableBinary"`
	Lists          []readerOptionalEntry[[]string]     `json:"lists"`
}

func TestReaderNullableGenericParameterReferences(t *testing.T) {
	const key = "demo.OptionalGenericConfig"
	const raw = `{"binary":[{"value":null},{"value":""},{"value":"aGVsbG8="}],"nullableBinary":[{"value":null},{"value":"aGVsbG8="}],"lists":[{"value":null},{"value":[]}]}`
	for _, lifecycle := range []Lifecycle{LifecycleEternal, LifecycleInstant} {
		t.Run(string(lifecycle), func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(ConfigSpec{SkelName: key, Lifecycle: lifecycle, Type: reflect.TypeFor[*readerOptionalGenericConfig]()})
			reader := newReader(&corelink.TestLinker{EternalConfigByKey: map[string]string{key: raw}, InstantConfigByKey: map[string]string{key: raw}}, registry)
			config := reader.GetByType(reflect.TypeFor[*readerOptionalGenericConfig]()).(*readerOptionalGenericConfig)
			require.Nil(t, config.Binary[0].Value)
			require.NotNil(t, config.Binary[1].Value)
			require.Empty(t, *config.Binary[1].Value)
			require.Equal(t, skel.Binary("hello"), *config.Binary[2].Value)
			require.Nil(t, config.NullableBinary[0].Value)
			require.Equal(t, skel.Binary("hello"), **config.NullableBinary[1].Value)
			require.Nil(t, config.Lists[0].Value)
			require.NotNil(t, config.Lists[1].Value)
			require.Empty(t, *config.Lists[1].Value)
		})
	}
}
