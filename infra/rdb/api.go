package rdb

import (
	"reflect"

	internalrdb "go.yorun.ai/vine/internal/infra/rdb"
	"gorm.io/gorm"
)

// Option configures a relational database component.
type Option = internalrdb.Option

// TypeAdder adds a model type to a database specification.
type TypeAdder = internalrdb.TypeAdder

// DatabaseSpec describes a named database and its registered models.
type DatabaseSpec = internalrdb.DatabaseSpec

// Database exposes the underlying GORM connection and transaction helpers.
type Database = internalrdb.Database

// Model provides the common identifier and timestamp fields for persisted records.
type Model = internalrdb.Model

// DeletableModel extends Model with soft-deletion metadata.
type DeletableModel = internalrdb.DeletableModel

// ModelConstraint is implemented by model pointer types accepted by Dao and Query.
type ModelConstraint = internalrdb.ModelConstraint

// Patch describes selected field updates for a record.
type Patch = internalrdb.Patch

// Dao provides typed create, read, update, and delete operations for M.
type Dao[M ModelConstraint] = internalrdb.Dao[M]

// Query builds and executes typed queries for M.
type Query[M ModelConstraint] = internalrdb.Query[M]

// T returns the reflection type for T without requiring a value of T.
func T[T any]() reflect.Type {
	return internalrdb.T[T]()
}

// NewDao creates a typed data access object backed by gdb.
func NewDao[M ModelConstraint](gdb *gorm.DB) Dao[M] {
	return internalrdb.NewDao[M](gdb)
}

// NewUUIDV7String returns a time-ordered UUID version 7 as a string.
func NewUUIDV7String() string {
	return internalrdb.NewUUIDV7String()
}
