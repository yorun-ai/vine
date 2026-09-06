// Package rdb integrates relational databases, models, queries, and data access objects with Vine.
//
// Prefer UModel for new models. Its uuid.UUID primary key is generated as UUIDv7
// and serialized using the registered GORM "uuid" serializer. RDB connections
// store UUID columns as PostgreSQL uuid or SQLite TEXT.
// For custom uuid.UUID or *uuid.UUID fields, use gorm:"type:uuid;serializer:uuid".
// A nil *uuid.UUID stores SQL NULL; a pointer to uuid.Nil() stores the zero UUID.
// Reading SQL NULL into a uuid.UUID value produces uuid.Nil().
// RDB connections also convert uuid.UUID SQL parameters automatically, including
// UUID slices expanded by GORM in expressions such as Where("id IN ?", ids).
// Use explicit SQL predicates for UUIDs rather than GORM primary-key shorthand
// or map conditions, which may interpret the UUID array as a list before binding.
package rdb
