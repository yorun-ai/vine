package adapter

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
	"reflect"
	"sync"
	"testing"
	"uuid"
)

type serializerTestModel struct {
	Id uuid.UUID `gorm:"type:uuid;serializer:uuid"`
}

func TestUUIDSerializerScan(t *testing.T) {
	s, err := schema.Parse(new(serializerTestModel), new(sync.Map), schema.NamingStrategy{})
	require.NoError(t, err)
	field := s.LookUpField("Id")
	id := uuid.NewV7()
	for _, value := range []any{id.String(), []byte(id.String()), nil} {
		row := new(serializerTestModel{Id: id})
		require.NoError(t, (_UUIDSerializer{}).Scan(context.Background(), field, reflect.ValueOf(row).Elem(), value))
		if value == nil {
			assert.Equal(t, uuid.Nil(), row.Id)
		} else {
			assert.Equal(t, id, row.Id)
		}
	}
	for _, value := range []any{"invalid", []byte("invalid"), 42} {
		row := new(serializerTestModel{Id: id})
		require.Error(t, (_UUIDSerializer{}).Scan(context.Background(), field, reflect.ValueOf(row).Elem(), value))
		assert.Equal(t, id, row.Id)
	}
}

type nullableUUIDTestModel struct {
	ID  int
	Ref *uuid.UUID `gorm:"type:uuid;serializer:uuid"`
}

func TestUUIDSerializerNullableScan(t *testing.T) {
	s, err := schema.Parse(new(nullableUUIDTestModel), new(sync.Map), schema.NamingStrategy{})
	require.NoError(t, err)
	field := s.LookUpField("Ref")
	id := uuid.NewV7()
	for _, value := range []any{id.String(), []byte(id.String()), uuid.Nil().String(), nil} {
		original := uuid.NewV7()
		row := new(nullableUUIDTestModel{Ref: &original})
		require.NoError(t, (_UUIDSerializer{}).Scan(context.Background(), field, reflect.ValueOf(row).Elem(), value))
		if value == nil {
			require.Nil(t, row.Ref)
		} else {
			require.NotNil(t, row.Ref)
			if value == uuid.Nil().String() {
				assert.Equal(t, uuid.Nil(), *row.Ref)
			} else {
				assert.Equal(t, id, *row.Ref)
			}
		}
	}
	row := new(nullableUUIDTestModel{Ref: &id})
	require.Error(t, (_UUIDSerializer{}).Scan(context.Background(), field, reflect.ValueOf(row).Elem(), "invalid"))
	assert.Equal(t, &id, row.Ref)
}

func TestUUIDSerializerNullableRoundTrip(t *testing.T) {
	db, err := openDriverTestDB(t, "sqlite://"+t.TempDir()+"/nullable.sqlite")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(new(nullableUUIDTestModel)))
	id, zero := uuid.NewV7(), uuid.Nil()
	rows := []*nullableUUIDTestModel{{Ref: nil}, {Ref: &id}, {Ref: &zero}}
	require.NoError(t, db.Create(&rows).Error)
	var loaded []nullableUUIDTestModel
	require.NoError(t, db.Order("id").Find(&loaded).Error)
	require.Len(t, loaded, 3)
	require.Nil(t, loaded[0].Ref)
	require.Equal(t, &id, loaded[1].Ref)
	require.Equal(t, &zero, loaded[2].Ref)
	var nullCount int64
	require.NoError(t, db.Model(new(nullableUUIDTestModel)).Where("ref IS NULL").Count(&nullCount).Error)
	require.Equal(t, int64(1), nullCount)
	// Save must distinguish a nil pointer from a pointer to the zero UUID.
	rows[1].Ref = nil
	require.NoError(t, db.Save(rows[1]).Error)
	row := nullableUUIDTestModel{Ref: &id}
	require.NoError(t, db.First(&row, rows[1].ID).Error)
	require.Nil(t, row.Ref)
	rows[1].Ref = &zero
	require.NoError(t, db.Save(rows[1]).Error)
	require.NoError(t, db.First(&row, rows[1].ID).Error)
	require.Equal(t, &zero, row.Ref)
}
