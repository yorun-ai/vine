package rdb

import (
	"reflect"

	"uuid"
)

// T returns the reflection type for T without requiring a value of T.
func T[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

// NewUUIDV7String creates a time-ordered UUID string suitable for database primary keys.
func NewUUIDV7String() string {
	return uuid.NewV7().String()
}
