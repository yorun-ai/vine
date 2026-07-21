package rdb

import (
	"reflect"

	"github.com/google/uuid"
)

// T returns the reflect.Type for T and keeps rdb package type references concise.
func T[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

// NewUUIDV7String creates a time-ordered UUID string suitable for database primary keys.
func NewUUIDV7String() string {
	return uuid.Must(uuid.NewV7()).String()
}
