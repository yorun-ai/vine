package admin

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

// _AppConfigServiceAppConfigRepo mirrors the repository contract: it returns
// configuration slots with their declaration, status and lifecycle assembled.
type _AppConfigServiceAppConfigRepo struct {
	items   []*core.AppConfig
	schemas []*skel.ConfigSchema
	enums   []*skel.EnumSchema
}

func (r *_AppConfigServiceAppConfigRepo) assemble(item *core.AppConfig) *core.AppConfig {
	var schema *skel.ConfigSchema
	for _, candidate := range r.schemas {
		if candidate.SkelName == item.Name {
			schema = candidate
			break
		}
	}
	item.Definition = core.NewAppConfigDefinition(schema, r.enums)
	item.Configured = item.Id != 0
	item.Lifecycle = core.AppConfigLifecycleFor(schema)
	if item.Configured {
		item.Status = core.AppConfigStatusFor(schema, item.Value, r.enums)
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
	for _, schema := range r.schemas {
		if _, ok := seen[schema.SkelName]; ok {
			continue
		}
		slots = append(slots, r.assemble(&core.AppConfig{Name: schema.SkelName}))
	}
	return slots
}

func (r *_AppConfigServiceAppConfigRepo) FindByName(name string) (*core.AppConfig, bool) {
	if item, ok := r.GetByName(name); ok {
		return item, true
	}
	for _, schema := range r.schemas {
		if schema.SkelName == name {
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

type _AppConfigServiceSchemaRepo struct {
	configSchemas []*skel.ConfigSchema
	enumSchemas   []*skel.EnumSchema
}

func (*_AppConfigServiceSchemaRepo) SaveDomainSchemas(string, string, []*skel.DomainSchema) {
}

func (*_AppConfigServiceSchemaRepo) SaveDomainSchemasJSON(string, string, []skel.JSON) {
}

func (*_AppConfigServiceSchemaRepo) ReleaseDomainSchemas(string, string) {}

func (*_AppConfigServiceSchemaRepo) ListDomainSchemaViews() []core.DomainSchemaView {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListVineHubSchemaViews() []core.DomainSchemaView {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListActorSchemaVersions() []core.SchemaVersion[*skel.ActorSchema] {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListConfigSchemaVersions() []core.SchemaVersion[*skel.ConfigSchema] {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListDataSchemaVersions() []core.SchemaVersion[*skel.DataSchema] {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListEnumSchemaVersions() []core.SchemaVersion[*skel.EnumSchema] {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListEventSchemaVersions() []core.SchemaVersion[*skel.EventSchema] {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListResourceSchemaVersions() []core.SchemaVersion[*skel.ResourceSchema] {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListServiceSchemaVersions() []core.SchemaVersion[*skel.ServiceSchema] {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListTaskSchemaVersions() []core.SchemaVersion[*skel.TaskSchema] {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListWebSchemaVersions() []core.SchemaVersion[*skel.WebSchema] {
	return nil
}

func (r *_AppConfigServiceSchemaRepo) ListAppConfigSchemas() []*skel.ConfigSchema {
	return r.configSchemas
}

func (*_AppConfigServiceSchemaRepo) ListActorSchemas() []*skel.ActorSchema {
	return nil
}

func (r *_AppConfigServiceSchemaRepo) ListEnumSchemas() []*skel.EnumSchema {
	return r.enumSchemas
}

func (*_AppConfigServiceSchemaRepo) ListServiceSchemas() []*skel.ServiceSchema {
	return nil
}

func (*_AppConfigServiceSchemaRepo) ListWebSchemas() []*skel.WebSchema {
	return nil
}

func TestAppConfigServiceCreateConfig(t *testing.T) {
	repo := &_AppConfigServiceAppConfigRepo{
		schemas: []*skel.ConfigSchema{{
			Name:      "FeatureConfig",
			SkelName:  "demo.user.FeatureConfig",
			Lifecycle: "INSTANT",
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
	require.NotNil(t, item.Schema)
	assert.Equal(t, "FeatureConfig", item.Schema.Name)
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
	repo.schemas = []*skel.ConfigSchema{{
		Name:     "SiteConfig",
		SkelName: "demo.user.SiteConfig",
	}}
	service := &AppConfigApiServiceServerImpl{AppConfigCore: &core.AppConfigCore{AppConfigRepo: repo}}

	assert.True(t, service.Remove(8))
	_, ok := repo.GetById(8)
	assert.False(t, ok)
	assert.Panics(t, func() {
		service.Remove(7)
	})
}

func findAppConfigItemForTest(items []skeled.AppConfigListItem, key string) *skeled.AppConfigListItem {
	for i := range items {
		if items[i].Key == key {
			return &items[i]
		}
	}
	return nil
}

func TestEditorScalarFormatsMatchRuntime(t *testing.T) {
	data, err := os.ReadFile("../../../dashboard/src/features/app/testdata/config-scalar.json")
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
				target = new(skel.Duration)
			case "decimal":
				target = new(skel.Decimal)
			case "uuid":
				target = new(skel.UUID)
			case "timestamp":
				target = new(skel.Timestamp)
			case "localdate":
				target = new(skel.LocalDate)
			case "localtime":
				target = new(skel.LocalTime)
			case "localdatetime":
				target = new(skel.LocalDateTime)
			default:
				t.Fatalf("unknown scalar %s", item.Type)
			}
			err := json.Unmarshal(item.Value, target)
			require.Equal(t, item.Valid, err == nil, "decode error: %v", err)
		})
	}
}

func (r *_AppConfigServiceSchemaRepo) GetWebSchema(skelName string) *skel.WebSchema {
	for _, schema := range r.ListWebSchemas() {
		if schema.SkelName == skelName {
			return schema
		}
	}
	return nil
}

func TestAppConfigServiceGetReturnsFieldSourcesAndDeclaredSlots(t *testing.T) {
	repo := &_AppConfigServiceAppConfigRepo{
		schemas: []*skel.ConfigSchema{
			{
				Name:      "FeatureConfig",
				SkelName:  "demo.FeatureConfig",
				Lifecycle: "ETERNAL",
				Members: []*skel.MemberSchema{
					{Name: "enabled", Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarBool}},
				},
			},
			{Name: "OtherConfig", SkelName: "demo.OtherConfig", Lifecycle: "INSTANT"},
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
	assert.Equal(t, "demo.FeatureConfig", detail.Schema.SkelName)

	// A configuration an application declares without a stored value resolves by
	// key and carries no provenance.
	declared := service.Get("demo.OtherConfig")
	assert.Zero(t, declared.Id)
	assert.Equal(t, "UNCONFIGURED", declared.Status)
	assert.Equal(t, "INSTANT", declared.Lifecycle)
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
