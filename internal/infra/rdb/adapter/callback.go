package adapter

import (
	"reflect"
	"sync"
	"uuid"

	"gorm.io/gorm"
)

var createCallbacksMu sync.Mutex

// RegisterCreateCallbacks installs UUID generation once per GORM callback registry.
// Call it before using an externally managed connection concurrently.
func RegisterCreateCallbacks(db *gorm.DB) error {
	createCallbacksMu.Lock()
	defer createCallbacksMu.Unlock()
	const name = "vine-infra-rdb:uuid"
	callbacks := db.Callback().Create()
	if callbacks.Get(name) != nil {
		return nil
	}
	return callbacks.Before("gorm:before_create").Register(name, assignUUIDPrimaryKeys)
}

func assignUUIDPrimaryKeys(db *gorm.DB) {
	if db.Error != nil || db.Statement.Schema == nil {
		return
	}
	var assign func(reflect.Value)
	assign = func(value reflect.Value) {
		for value.IsValid() && value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			value = value.Elem()
		}
		if !value.IsValid() {
			return
		}
		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < value.Len(); i++ {
				assign(value.Index(i))
			}
		case reflect.Struct:
			for _, field := range db.Statement.Schema.PrimaryFields {
				if field.FieldType != reflect.TypeFor[uuid.UUID]() {
					continue
				}
				target := field.ReflectValueOf(db.Statement.Context, value)
				if !target.IsZero() {
					continue
				}
				if !target.CanSet() {
					db.AddError(gorm.ErrInvalidValue)
					return
				}
				target.Set(reflect.ValueOf(uuid.NewV7()))
			}
		}
	}
	assign(db.Statement.ReflectValue)
}
