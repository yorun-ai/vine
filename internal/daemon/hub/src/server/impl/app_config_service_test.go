package impl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type _AppConfigServiceAppConfigRepo struct {
	items []*core.AppConfig
}

func (r *_AppConfigServiceAppConfigRepo) ListItems() []*core.AppConfig {
	return r.items
}

func (r *_AppConfigServiceAppConfigRepo) GetItemById(id int) (*core.AppConfig, bool) {
	for _, item := range r.items {
		if item.Id == id {
			return item, true
		}
	}
	return nil, false
}

func (r *_AppConfigServiceAppConfigRepo) GetItemByName(name string) (*core.AppConfig, bool) {
	for _, item := range r.items {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

func (r *_AppConfigServiceAppConfigRepo) SaveItem(item *core.AppConfig) {
	if item.Id == 0 {
		item.Id = len(r.items) + 1
		r.items = append(r.items, item)
		return
	}
	for index, current := range r.items {
		if current.Id == item.Id {
			r.items[index] = item
			return
		}
	}
	r.items = append(r.items, item)
}

func (r *_AppConfigServiceAppConfigRepo) RemoveItem(id int) bool {
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

func TestAppConfigServiceListReturnsConfigSchemaDescriptionsAndFields(t *testing.T) {
	service := &AppConfigServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{
			AppConfigRepo: &_AppConfigServiceAppConfigRepo{
				items: []*core.AppConfig{{
					Id:      7,
					Name:    "demo.user.SiteConfig",
					Value:   `{"title":"Demo","defaultStatus":"ACTIVE","statusMessages":{"ACTIVE":"Ready"}}`,
					Version: 1,
				}},
			},
		},
		SchemaRepo: &_AppConfigServiceSchemaRepo{
			enumSchemas: []*skel.EnumSchema{
				{
					Name:     "UserStatus",
					SkelName: "demo.admin.UserStatus",
					Items: []*skel.EnumItemSchema{
						{Name: "DISABLED", Description: "禁用"},
					},
				},
				{
					Name:     "UserStatus",
					SkelName: "demo.user.UserStatus",
					Items: []*skel.EnumItemSchema{
						{Name: "ACTIVE", Description: "启用"},
						{Name: "PENDING", Description: "待审核"},
					},
				},
			},
			configSchemas: []*skel.ConfigSchema{{
				Name:        "SiteConfig",
				SkelName:    "demo.user.SiteConfig",
				Description: "站点配置",
				Lifecycle:   "ETERNAL",
				Members: []*skel.MemberSchema{
					{
						Name:        "title",
						Description: "页面标题",
						Type: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
					{
						Name: "defaultStatus",
						Type: &skel.TypeSchema{
							Kind:     skel.TypeKindEnum,
							Name:     "UserStatus",
							SkelName: "demo.user.UserStatus",
						},
					},
					{
						Name: "statusMessages",
						Type: &skel.TypeSchema{
							Kind: skel.TypeKindMap,
							Key: &skel.TypeSchema{
								Kind:     skel.TypeKindEnum,
								Name:     "UserStatus",
								SkelName: "demo.user.UserStatus",
							},
							Value: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
				},
			}},
		},
	}

	items := service.List()

	require.Len(t, items, 1)
	assert.Equal(t, 7, items[0].Id)
	assert.Equal(t, "NORMAL", items[0].Status)
	require.NotNil(t, items[0].Schema)
	assert.Equal(t, "demo.user.SiteConfig", items[0].Schema.SkelName)
	assert.Equal(t, "SiteConfig", items[0].Schema.Name)
	require.NotNil(t, items[0].Schema.Description)
	assert.Equal(t, "站点配置", *items[0].Schema.Description)
	assert.Equal(t, "ETERNAL", items[0].Schema.Lifecycle)
	require.Len(t, items[0].Schema.Fields, 3)
	assert.Equal(t, "title", items[0].Schema.Fields[0].Name)
	assert.Equal(t, "string", items[0].Schema.Fields[0].Type)
	require.NotNil(t, items[0].Schema.Fields[0].Description)
	assert.Equal(t, "页面标题", *items[0].Schema.Fields[0].Description)
	assert.Equal(t, "defaultStatus", items[0].Schema.Fields[1].Name)
	assert.Equal(t, "demo.user.UserStatus", items[0].Schema.Fields[1].Type)
	assert.Nil(t, items[0].Schema.Fields[1].Description)
	require.Len(t, items[0].Schema.Fields[1].EnumItems, 2)
	assert.Equal(t, "ACTIVE", items[0].Schema.Fields[1].EnumItems[0].Name)
	require.NotNil(t, items[0].Schema.Fields[1].EnumItems[0].Description)
	assert.Equal(t, "启用", *items[0].Schema.Fields[1].EnumItems[0].Description)
	assert.Equal(t, "statusMessages", items[0].Schema.Fields[2].Name)
	assert.Equal(t, "map<demo.user.UserStatus, string>", items[0].Schema.Fields[2].Type)
	require.Len(t, items[0].Schema.Fields[2].EnumItems, 2)
	assert.Equal(t, "ACTIVE", items[0].Schema.Fields[2].EnumItems[0].Name)
}

func TestAppConfigServiceListIncludesUnusedAndUnconfiguredConfigs(t *testing.T) {
	service := &AppConfigServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{
			AppConfigRepo: &_AppConfigServiceAppConfigRepo{
				items: []*core.AppConfig{
					{Id: 7, Name: "demo.user.SiteConfig", Value: `{}`, Version: 1},
					{Id: 8, CreatedAt: time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC), Name: "demo.user.LegacyConfig", Value: `{"enabled":true}`, Version: 1},
					{Id: 9, CreatedAt: time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC), Name: "demo.user.NewerConfig", Value: `{"enabled":true}`, Version: 1},
					{Id: 10, Name: "demo.user.BrokenConfig", Value: `{"enabled":"yes"}`, Version: 1},
				},
			},
		},
		SchemaRepo: &_AppConfigServiceSchemaRepo{
			configSchemas: []*skel.ConfigSchema{
				{Name: "SiteConfig", SkelName: "demo.user.SiteConfig", Lifecycle: "ETERNAL"},
				{
					Name:      "BrokenConfig",
					SkelName:  "demo.user.BrokenConfig",
					Lifecycle: "INSTANT",
					Members: []*skel.MemberSchema{{
						Name: "enabled",
						Type: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarBool,
						},
					}},
				},
				{Name: "FeatureConfig", SkelName: "demo.user.FeatureConfig", Lifecycle: "INSTANT"},
			},
		},
	}

	items := service.List()

	require.Len(t, items, 5)
	assert.Equal(t, "demo.user.BrokenConfig", items[0].Key)
	assert.Equal(t, "MISMATCH", items[0].Status)
	assert.Equal(t, "demo.user.FeatureConfig", items[1].Key)
	assert.Equal(t, "UNCONFIGURED", items[1].Status)
	assert.Equal(t, 0, items[1].Id)
	assert.Equal(t, "", items[1].Value)
	require.NotNil(t, items[1].Schema)
	assert.Equal(t, "FeatureConfig", items[1].Schema.Name)
	assert.Equal(t, "demo.user.NewerConfig", items[2].Key)
	assert.Equal(t, "UNUSED", items[2].Status)
	assert.Nil(t, items[2].Schema)
	assert.Equal(t, "demo.user.LegacyConfig", items[3].Key)
	assert.Equal(t, "UNUSED", items[3].Status)
	assert.Equal(t, "demo.user.SiteConfig", items[4].Key)
	assert.Equal(t, "NORMAL", items[4].Status)
}

func TestAppConfigServiceListMatchesConfigSchemaByFullSkelName(t *testing.T) {
	service := &AppConfigServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{
			AppConfigRepo: &_AppConfigServiceAppConfigRepo{
				items: []*core.AppConfig{{
					Id:      7,
					Name:    "lca.core.RedisConfig",
					Value:   `{"endpoint":"redis://redis.example.com:6379/0"}`,
					Version: 1,
				}},
			},
		},
		SchemaRepo: &_AppConfigServiceSchemaRepo{
			configSchemas: []*skel.ConfigSchema{
				{
					Name:      "RedisConfig",
					SkelName:  "uic.RedisConfig",
					Lifecycle: "ETERNAL",
					Members: []*skel.MemberSchema{{
						Name: "addr",
						Type: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					}},
				},
				{
					Name:      "RedisConfig",
					SkelName:  "lca.core.RedisConfig",
					Lifecycle: "ETERNAL",
					Members: []*skel.MemberSchema{{
						Name: "endpoint",
						Type: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					}},
				},
			},
		},
	}

	items := service.List()

	require.Len(t, items, 2)
	config := findAppConfigItemForTest(items, "lca.core.RedisConfig")
	require.NotNil(t, config)
	assert.Equal(t, "NORMAL", config.Status)
	require.NotNil(t, config.Schema)
	assert.Equal(t, "lca.core.RedisConfig", config.Schema.SkelName)
	require.Len(t, config.Schema.Fields, 1)
	assert.Equal(t, "endpoint", config.Schema.Fields[0].Name)
	unconfigured := findAppConfigItemForTest(items, "uic.RedisConfig")
	require.NotNil(t, unconfigured)
	assert.Equal(t, "UNCONFIGURED", unconfigured.Status)
}

func TestAppConfigServiceListDoesNotMatchConfigSchemaByShortName(t *testing.T) {
	service := &AppConfigServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{
			AppConfigRepo: &_AppConfigServiceAppConfigRepo{
				items: []*core.AppConfig{{
					Id:      7,
					Name:    "RedisConfig",
					Value:   `{"endpoint":"redis://localhost:6379/0"}`,
					Version: 1,
				}},
			},
		},
		SchemaRepo: &_AppConfigServiceSchemaRepo{
			configSchemas: []*skel.ConfigSchema{{
				Name:      "RedisConfig",
				SkelName:  "lca.core.RedisConfig",
				Lifecycle: "ETERNAL",
				Members: []*skel.MemberSchema{{
					Name: "endpoint",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				}},
			}},
		},
	}

	items := service.List()

	require.Len(t, items, 2)
	shortConfig := findAppConfigItemForTest(items, "RedisConfig")
	require.NotNil(t, shortConfig)
	assert.Equal(t, "UNUSED", shortConfig.Status)
	assert.Nil(t, shortConfig.Schema)
	fullConfig := findAppConfigItemForTest(items, "lca.core.RedisConfig")
	require.NotNil(t, fullConfig)
	assert.Equal(t, "UNCONFIGURED", fullConfig.Status)
}

func TestAppConfigServiceCreateConfig(t *testing.T) {
	repo := &_AppConfigServiceAppConfigRepo{}
	service := &AppConfigServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: repo},
		SchemaRepo: &_AppConfigServiceSchemaRepo{
			configSchemas: []*skel.ConfigSchema{{
				Name:      "FeatureConfig",
				SkelName:  "demo.user.FeatureConfig",
				Lifecycle: "INSTANT",
			}},
		},
	}

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
	service := &AppConfigServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: &_AppConfigServiceAppConfigRepo{}},
		SchemaRepo:    &_AppConfigServiceSchemaRepo{},
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

func TestAppConfigServiceRemoveOnlyAllowsUnusedConfig(t *testing.T) {
	repo := &_AppConfigServiceAppConfigRepo{
		items: []*core.AppConfig{
			{Id: 7, Name: "demo.user.SiteConfig", Value: `{}`, Version: 1},
			{Id: 8, Name: "demo.user.LegacyConfig", Value: `{}`, Version: 1},
		},
	}
	service := &AppConfigServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: repo},
		SchemaRepo: &_AppConfigServiceSchemaRepo{
			configSchemas: []*skel.ConfigSchema{{
				Name:     "SiteConfig",
				SkelName: "demo.user.SiteConfig",
			}},
		},
	}

	assert.True(t, service.Remove(8))
	_, ok := repo.GetItemById(8)
	assert.False(t, ok)
	assert.Panics(t, func() {
		service.Remove(7)
	})
}

func findAppConfigItemForTest(items []skeled.AppConfigItem, key string) *skeled.AppConfigItem {
	for i := range items {
		if items[i].Key == key {
			return &items[i]
		}
	}
	return nil
}
