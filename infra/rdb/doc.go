// Package rdb integrates relational databases, models, queries, and data access objects with Vine.
//
// UUID primary keys are generated before user creation hooks on RDB connections
// and connections passed to NewDao, including when user hooks are skipped.
// Prefer UModel for new models. Its uuid.UUID primary key is generated as UUIDv7
// and serialized using the registered GORM "vine-rdb-uuid" serializer. RDB connections
// store UUID columns as PostgreSQL uuid or SQLite TEXT.
// For custom uuid.UUID or *uuid.UUID fields, use gorm:"type:uuid;serializer:vine-rdb-uuid".
// A nil *uuid.UUID stores SQL NULL; a pointer to uuid.Nil() stores the zero UUID.
// Reading SQL NULL into a uuid.UUID value produces uuid.Nil().
// RDB connections also convert uuid.UUID SQL parameters automatically, including
// UUID slices expanded by GORM in expressions such as Where("id IN ?", ids).
// DAO queries also normalize UUID primary-key shorthand and map conditions.
// For direct GORM calls, use explicit SQL predicates: primary-key shorthand
// and map conditions may interpret the UUID array as a list before binding.
package rdb
