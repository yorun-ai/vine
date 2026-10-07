package admin

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

// _AppConfigServiceAppConfigRepo mirrors the repository contract: it returns
// configuration slots with their declaration, status and lifecycle assembled.
type _AppConfigServiceAppConfigRepo struct {
	items       []*core.AppConfig
	descriptors []*skeldesc.Config
	enums       []*skeldesc.Enum
}

func (r *_AppConfigServiceAppConfigRepo) assemble(item *core.AppConfig) *core.AppConfig {
	var descriptor *skeldesc.Config
	for _, candidate := range r.descriptors {
		if candidate.SkelName == item.Name {
			descriptor = candidate
			break
		}
	}
	item.Definition = core.NewAppConfigDefinition(descriptor, r.enums, nil)
	item.Configured = item.Id != 0
	item.Lifecycle = core.AppConfigLifecycleFor(descriptor)
	if item.Configured {
		item.Status = core.AppConfigStatusFor(descriptor, item.Value, r.enums, nil)
	} else {
		item.Status = core.AppConfigStatusUnconfigured
	}
	return item
}

func (r *_AppConfigServiceAppConfigRepo) List() []*core.AppConfig {
	items := make([]*core.AppConfig, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, r.assemble(item))
	}
	return items
}

func (r *_AppConfigServiceAppConfigRepo) ListSlots() []*core.AppConfig {
	slots := r.List()
	seen := map[string]struct{}{}
	for _, slot := range slots {
		seen[slot.Name] = struct{}{}
	}
	for _, descriptor := range r.descriptors {
		if _, ok := seen[descriptor.SkelName]; ok {
			continue
		}
		slots = append(slots, r.assemble(&core.AppConfig{Name: descriptor.SkelName}))
	}
	return slots
}

func (r *_AppConfigServiceAppConfigRepo) FindByName(name string) (*core.AppConfig, bool) {
	if item, ok := r.GetByName(name); ok {
		return item, true
	}
	for _, descriptor := range r.descriptors {
		if descriptor.SkelName == name {
			return r.assemble(&core.AppConfig{Name: name}), true
		}
	}
	return nil, false
}

func (r *_AppConfigServiceAppConfigRepo) GetById(id int) (*core.AppConfig, bool) {
	for _, item := range r.items {
		if item.Id == id {
			return r.assemble(item), true
		}
	}
	return nil, false
}

func (r *_AppConfigServiceAppConfigRepo) GetByName(name string) (*core.AppConfig, bool) {
	for _, item := range r.items {
		if item.Name == name {
			return r.assemble(item), true
		}
	}
	return nil, false
}

func (r *_AppConfigServiceAppConfigRepo) Save(item *core.AppConfig) {
	if item.Id == 0 {
		item.Id = len(r.items) + 1
		*item = *r.assemble(item)
		r.items = append(r.items, item)
		return
	}
	*item = *r.assemble(item)
	for index, current := range r.items {
		if current.Id == item.Id {
			r.items[index] = item
			return
		}
	}
	r.items = append(r.items, item)
}

func (r *_AppConfigServiceAppConfigRepo) Remove(id int) bool {
	for index, item := range r.items {
		if item.Id == id {
			r.items = append(r.items[:index], r.items[index+1:]...)
			return true
		}
	}
	return false
}

func TestAppConfigServiceCreateConfig(t *testing.T) {
	repo := &_AppConfigServiceAppConfigRepo{
		descriptors: []*skeldesc.Config{{
			Name:      "FeatureConfig",
			SkelName:  "demo.user.FeatureConfig",
			Lifecycle: skeldesc.ConfigLifecycleInstant,
		}},
	}
	service := &AppConfigApiServiceServerImpl{AppConfigCore: &core.AppConfigCore{AppConfigRepo: repo}}

	item := service.Create(skeled.AppConfigCreation{
		SkelName: "demo.user.FeatureConfig",
		Value:    `{}`,
	})

	assert.Equal(t, "demo.user.FeatureConfig", item.Key)
	assert.Equal(t, "NORMAL", item.Status)
	assert.Equal(t, 1, item.Id)
	require.NotNil(t, item.Descriptor)
	assert.Equal(t, "FeatureConfig", item.Descriptor.Name)
}

func TestAppConfigServiceCreateRejectsInvalidConfigSkelName(t *testing.T) {
	service := &AppConfigApiServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: &_AppConfigServiceAppConfigRepo{}},
	}

	for _, skelName := range []string{"ddd", "demo.ddd", "demo.user.bad-config", "demo.1user.BadConfig"} {
		assert.Panics(t, func() {
			service.Create(skeled.AppConfigCreation{
				SkelName: skelName,
				Value:    `{}`,
			})
		})
	}
}

func TestAppConfigSkelNameShape(t *testing.T) {
	for name, want := range map[string]bool{
		"demo.user.FeatureConfig": true,
		"demo.Config":             true,
		"Config":                  false,
		"_demo.FeatureConfig":     true,
		"ddd":                     false,
		"demo.ddd":                false,
		"demo.user.bad-config":    false,
		"demo.1user.BadConfig":    false,
		"demo.user.":              false,
		".demo.Config":            false,
	} {
		assert.Equal(t, want, isValidConfigSkelName(name), name)
	}
}

func TestAppConfigServiceRemoveOnlyAllowsUnusedConfig(t *testing.T) {
	repo := &_AppConfigServiceAppConfigRepo{
		items: []*core.AppConfig{
			{Id: 7, Name: "demo.user.SiteConfig", Value: `{}`, Version: 1},
			{Id: 8, Name: "demo.user.LegacyConfig", Value: `{}`, Version: 1},
		},
	}
	repo.descriptors = []*skeldesc.Config{{
		Name:     "SiteConfig",
		SkelName: "demo.user.SiteConfig", Lifecycle: skeldesc.ConfigLifecycleEternal,
	}}
	service := &AppConfigApiServiceServerImpl{AppConfigCore: &core.AppConfigCore{AppConfigRepo: repo}}

	assert.True(t, service.Remove(8))
	_, ok := repo.GetById(8)
	assert.False(t, ok)
	assert.Panics(t, func() {
		service.Remove(7)
	})
}

func TestEditorScalarFormatsMatchRuntime(t *testing.T) {
	data, err := os.ReadFile("../../mod/admin/dashboard/src/features/app/testdata/config-scalar.json")
	require.NoError(t, err)
	var cases []struct {
		Type  string         `json:"type"`
		Value jsontext.Value `json:"value"`
		Valid bool           `json:"valid"`
	}
	require.NoError(t, json.Unmarshal(data, &cases))
	for _, item := range cases {
		t.Run(item.Type+"/"+string(item.Value), func(t *testing.T) {
			var target any
			switch item.Type {
			case "duration":
				target = new(skeltype.Duration)
			case "decimal":
				target = new(skeltype.Decimal)
			case "uuid":
				target = new(skeltype.UUID)
			case "timestamp":
				target = new(skeltype.Timestamp)
			case "localdate":
				target = new(skeltype.LocalDate)
			case "localtime":
				target = new(skeltype.LocalTime)
			case "localdatetime":
				target = new(skeltype.LocalDateTime)
			default:
				t.Fatalf("unknown scalar %s", item.Type)
			}
			err := json.Unmarshal(item.Value, target)
			require.Equal(t, item.Valid, err == nil, "decode error: %v", err)
		})
	}
}

func TestAppConfigServiceGetReturnsFieldSourcesAndDeclaredSlots(t *testing.T) {
	repo := &_AppConfigServiceAppConfigRepo{
		descriptors: []*skeldesc.Config{
			{
				Name:      "FeatureConfig",
				SkelName:  "demo.FeatureConfig",
				Lifecycle: skeldesc.ConfigLifecycleEternal,
				Members: []*skeldesc.Member{
					{Name: "enabled", Type: &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarBoolean}},
				},
			},
			{Name: "OtherConfig", SkelName: "demo.OtherConfig", Lifecycle: skeldesc.ConfigLifecycleInstant},
		},
		items: []*core.AppConfig{{
			Id:           7,
			Name:         "demo.FeatureConfig",
			Value:        `{"enabled":true}`,
			Version:      1,
			FieldSources: core.FieldSources{"/value": {Source: "app/default", Override: "hub"}},
		}},
	}
	service := &AppConfigApiServiceServerImpl{AppConfigCore: &core.AppConfigCore{AppConfigRepo: repo}}

	detail := service.Get("demo.FeatureConfig")
	require.Len(t, detail.FieldSources, 1)
	assert.Equal(t, "/value", detail.FieldSources[0].Path)
	assert.Equal(t, "hub", detail.FieldSources[0].Override)
	assert.Equal(t, "demo.FeatureConfig", detail.Descriptor.SkelName)

	// A configuration an application declares without a stored value resolves by
	// key and carries no provenance.
	declared := service.Get("demo.OtherConfig")
	assert.Zero(t, declared.Id)
	assert.Equal(t, "UNCONFIGURED", declared.Status)
	assert.Equal(t, "instant", declared.Lifecycle)
	assert.Empty(t, declared.FieldSources)

	// The list payload has no field for provenance, so it cannot leak sources.
	items := service.List()
	require.Len(t, items, 2)
	statuses := map[string]string{}
	for _, item := range items {
		statuses[item.Key] = item.Status
	}
	assert.Equal(t, map[string]string{"demo.FeatureConfig": "NORMAL", "demo.OtherConfig": "UNCONFIGURED"}, statuses)
}

func TestStructuredConfigDescriptorMapping(t *testing.T) {
	kind := &skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: "demo.Box", TypeArguments: []*skeldesc.Type{{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarBinary}}}
	definition := core.NewAppConfigDefinition(&skeldesc.Config{Sensitive: true, Members: []*skeldesc.Member{{Name: "box", Type: kind}}, Lifecycle: skeldesc.ConfigLifecycleEternal}, nil,
		[]*skeldesc.Data{{SkelName: "demo.Box", TypeParameters: []string{"T"}, Sensitive: true, Members: []*skeldesc.Member{{Name: "value", Sensitive: true, Example: "aGVsbG8=", Type: &skeldesc.Type{Kind: skeldesc.TypeKindTypeParameter, Name: "T"}}}}})
	result := toServerAppConfigDescriptor(definition)
	require.True(t, result.Sensitive)
	require.Len(t, result.DataTypes, 1)
	require.True(t, result.DataTypes[0].Sensitive)
	require.Equal(t, []string{"T"}, result.DataTypes[0].TypeParameters)
	require.Equal(t, "aGVsbG8=", result.DataTypes[0].Fields[0].Example)
	require.True(t, result.DataTypes[0].Fields[0].Sensitive)
	require.Equal(t, "binary", result.Fields[0].ValueType.TypeArguments[0].Name)
}
