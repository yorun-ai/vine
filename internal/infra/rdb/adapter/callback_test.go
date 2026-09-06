package adapter

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type callbackTestModel struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid;serializer:vine-rdb-uuid"`
	Reference uuid.UUID `gorm:"type:uuid;serializer:vine-rdb-uuid"`
	Hooks     []string  `gorm:"-"`
}

func (m *callbackTestModel) BeforeSave(_ *gorm.DB) error {
	if m.ID == uuid.Nil() {
		return fmt.Errorf("BeforeSave received an empty ID")
	}
	m.Hooks = append(m.Hooks, "save")
	return nil
}

func (m *callbackTestModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil() {
		return fmt.Errorf("BeforeCreate received an empty ID")
	}
	m.Hooks = append(m.Hooks, "create")
	return nil
}

func TestUUIDCallbackBeforeUserHooks(t *testing.T) {
	db, err := openDriverTestDB(t, "sqlite://"+t.TempDir()+"/callbacks.sqlite")
	require.NoError(t, err)
	require.NoError(t, RegisterCreateCallbacks(db))
	require.NoError(t, RegisterCreateCallbacks(db.Session(&gorm.Session{})))
	require.NoError(t, db.AutoMigrate(new(callbackTestModel)))
	supplied := uuid.NewV7()
	rows := []*callbackTestModel{new(callbackTestModel), new(callbackTestModel{ID: supplied})}
	require.NoError(t, db.Create(&rows).Error)
	require.Equal(t, supplied, rows[1].ID)
	require.NotEqual(t, rows[0].ID, rows[1].ID)
	for _, row := range rows {
		require.Equal(t, byte(7), row.ID[6]>>4)
		require.Equal(t, []string{"save", "create"}, row.Hooks)
		require.Equal(t, uuid.Nil(), row.Reference)
	}
	values := []callbackTestModel{{}, {}}
	require.NoError(t, db.CreateInBatches(&values, 1).Error)
	for _, row := range values {
		require.Equal(t, []string{"save", "create"}, row.Hooks)
	}
	skipped := new(callbackTestModel)
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(skipped).Error)
	require.NotEqual(t, uuid.Nil(), skipped.ID)
	require.Empty(t, skipped.Hooks)
}
