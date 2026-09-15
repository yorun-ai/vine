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

func (s *configRepoSpy) List() []*AppConfig {
	s.calls = append(s.calls, "List")
	items := make([]*AppConfig, 0, len(s.items))
	for _, item := range s.items {
		value := *item
		items = append(items, &value)
	}
	return items
}

func (s *configRepoSpy) ListSlots() []*AppConfig {
	return s.List()
}

func (s *configRepoSpy) FindByName(name string) (*AppConfig, bool) {
	return s.GetByName(name)
}

func (s *configRepoSpy) GetById(id int) (*AppConfig, bool) {
	s.calls = append(s.calls, "GetById")
	for _, item := range s.items {
		if item.Id != id {
			continue
		}
		value := *item
		return &value, true
	}
	return nil, false
}

func (s *configRepoSpy) GetByName(name string) (*AppConfig, bool) {
	s.calls = append(s.calls, "GetByName:"+name)
	item, ok := s.items[name]
	if !ok {
		return nil, false
	}
	value := *item
	return &value, true
}

func (s *configRepoSpy) Save(item *AppConfig) {
	s.calls = append(s.calls, "Save")
	value := *item
	s.savedItem = &value
	if s.items == nil {
		s.items = map[string]*AppConfig{}
	}
	s.items[item.Name] = &value
}

func (s *configRepoSpy) Remove(id int) bool {
	s.calls = append(s.calls, "Remove")
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
	assert.Equal(t, []string{"List"}, repo.calls)
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

	assert.Equal(t, []string{"GetById"}, repo.calls)
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
	assert.Equal(t, []string{"GetById"}, repo.calls)
}

func TestAppConfigCoreCreate(t *testing.T) {
	repo := &configRepoSpy{}
	core := &AppConfigCore{AppConfigRepo: repo}

	item := core.Create(AppConfigCreation{
		Name:  "db.main",
		Value: `{"connUrl":"postgres://demo"}`,
	})

	assert.Equal(t, []string{
		"GetByName:db.main",
		"Save",
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
	assert.Equal(t, []string{"GetByName:db.main"}, repo.calls)
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
		"GetById",
		"Save",
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
	assert.Equal(t, []string{"GetById"}, repo.calls)
}

func capturePanic(fn func()) (got any) {
	defer func() {
		got = recover()
	}()
	fn()
	return got
}

func TestAppConfigSaveOwnsIdentityAndVersion(t *testing.T) {
	repo := &configRepoSpy{}
	target := &AppConfigCore{AppConfigRepo: repo}
	first := target.Save(AppConfig{Id: 99, Name: "demo", Value: "one", Version: 99})
	require.Equal(t, 0, first.Id)
	require.Equal(t, 1, first.Version)
	require.False(t, first.CreatedAt.IsZero())
	repo.items["demo"].Id = 7
	same := target.Save(AppConfig{Name: "demo", Value: "one"})
	require.Equal(t, 7, same.Id)
	require.Equal(t, first.CreatedAt, same.CreatedAt)
	require.Equal(t, 1, same.Version)
	changed := target.Save(AppConfig{Name: "demo", Value: "two"})
	require.Equal(t, 2, changed.Version)
	require.Equal(t, 2, target.Update(7, AppConfigUpdate{Value: new("two")}).Version)
	require.Equal(t, 2, target.Update(7, AppConfigUpdate{}).Version)
	require.Equal(t, 3, target.Update(7, AppConfigUpdate{Value: new("three")}).Version)
}

func TestAppConfigValidateDoesNotAccessStorage(t *testing.T) {
	target := &AppConfigCore{}
	require.NotPanics(t, func() { target.Validate(AppConfig{Name: "demo"}) })
	require.Panics(t, func() { target.Validate(AppConfig{Name: " "}) })
	repo := &configRepoSpy{}
	target.AppConfigRepo = repo
	require.Panics(t, func() { target.Save(AppConfig{}) })
	require.Empty(t, repo.calls)
}
