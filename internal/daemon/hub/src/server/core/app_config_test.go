package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
)

type configRepoSpy struct {
	calls []string

	items map[string]*AppConfig

	savedItem *AppConfig
}

func (s *configRepoSpy) ListItems() []*AppConfig {
	s.calls = append(s.calls, "ListItems")
	items := make([]*AppConfig, 0, len(s.items))
	for _, item := range s.items {
		value := *item
		items = append(items, &value)
	}
	return items
}

func (s *configRepoSpy) GetItemById(id int) (*AppConfig, bool) {
	s.calls = append(s.calls, "GetItemById")
	for _, item := range s.items {
		if item.Id != id {
			continue
		}
		value := *item
		return &value, true
	}
	return nil, false
}

func (s *configRepoSpy) GetItemByName(name string) (*AppConfig, bool) {
	s.calls = append(s.calls, "GetItemByName:"+name)
	item, ok := s.items[name]
	if !ok {
		return nil, false
	}
	value := *item
	return &value, true
}

func (s *configRepoSpy) SaveItem(item *AppConfig) {
	s.calls = append(s.calls, "SaveItem")
	value := *item
	s.savedItem = &value
	if s.items == nil {
		s.items = map[string]*AppConfig{}
	}
	s.items[item.Name] = &value
}

func (s *configRepoSpy) RemoveItem(id int) bool {
	s.calls = append(s.calls, "RemoveItem")
	for name, item := range s.items {
		if item.Id != id {
			continue
		}
		delete(s.items, name)
		return true
	}
	return false
}

func TestAppConfigCoreList(t *testing.T) {
	repo := &configRepoSpy{
		items: map[string]*AppConfig{
			"db.main": {
				Id:      1,
				Name:    "db.main",
				Value:   `{"connUrl":"postgres://demo"}`,
				Version: 1,
			},
		},
	}
	core := &AppConfigCore{AppConfigRepo: repo}

	items := core.List()

	assert.Len(t, items, 1)
	assert.Equal(t, []string{"ListItems"}, repo.calls)
	assert.Equal(t, "db.main", items[0].Name)
}

func TestAppConfigCoreGet(t *testing.T) {
	repo := &configRepoSpy{
		items: map[string]*AppConfig{
			"feature.flag": {
				Id:      2,
				Name:    "feature.flag",
				Value:   `{"enabled":true}`,
				Version: 2,
			},
		},
	}
	core := &AppConfigCore{AppConfigRepo: repo}

	item := core.Get(2)

	assert.Equal(t, []string{"GetItemById"}, repo.calls)
	assert.Equal(t, 2, item.Version)
	assert.Equal(t, `{"enabled":true}`, item.Value)
}

func TestAppConfigCoreGetMissing(t *testing.T) {
	repo := &configRepoSpy{}
	core := &AppConfigCore{AppConfigRepo: repo}

	panicValue := capturePanic(func() {
		core.Get(404)
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetItemById"}, repo.calls)
}

func TestAppConfigCoreCreate(t *testing.T) {
	repo := &configRepoSpy{}
	core := &AppConfigCore{AppConfigRepo: repo}

	item := core.Create(AppConfigCreation{
		Name:  "db.main",
		Value: `{"connUrl":"postgres://demo"}`,
	})

	assert.Equal(t, []string{
		"GetItemByName:db.main",
		"SaveItem",
	}, repo.calls)
	require.NotNil(t, repo.savedItem)
	assert.Equal(t, 1, repo.savedItem.Version)
	assert.Equal(t, item, repo.savedItem)
}

func TestAppConfigCoreCreateExisting(t *testing.T) {
	repo := &configRepoSpy{
		items: map[string]*AppConfig{
			"db.main": {Name: "db.main"},
		},
	}
	core := &AppConfigCore{AppConfigRepo: repo}

	panicValue := capturePanic(func() {
		core.Create(AppConfigCreation{Name: "db.main"})
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetItemByName:db.main"}, repo.calls)
}

func TestAppConfigCoreUpdate(t *testing.T) {
	newValue := `{"enabled":false}`
	repo := &configRepoSpy{
		items: map[string]*AppConfig{
			"feature.flag": {
				Id:      3,
				Name:    "feature.flag",
				Value:   `{"enabled":true}`,
				Version: 3,
			},
		},
	}
	core := &AppConfigCore{AppConfigRepo: repo}

	item := core.Update(3, AppConfigUpdate{
		Value: &newValue,
	})

	assert.Equal(t, []string{
		"GetItemById",
		"SaveItem",
	}, repo.calls)
	require.NotNil(t, repo.savedItem)
	assert.Equal(t, 4, item.Version)
	assert.Equal(t, newValue, item.Value)
	assert.Equal(t, item, repo.savedItem)
}

func TestAppConfigCoreUpdateMissing(t *testing.T) {
	repo := &configRepoSpy{}
	core := &AppConfigCore{AppConfigRepo: repo}

	panicValue := capturePanic(func() {
		core.Update(404, AppConfigUpdate{})
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetItemById"}, repo.calls)
}

func capturePanic(fn func()) (got any) {
	defer func() {
		got = recover()
	}()
	fn()
	return got
}
