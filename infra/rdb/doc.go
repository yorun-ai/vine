// Package rdb integrates relational databases, models, queries, and data access objects with Vine.
//
// UUID primary keys are generated before user creation hooks on RDB connections
// and connections passed to NewDao, including when user hooks are skipped.
// With OnConflict DoNothing, a skipped insert can retain its generated UUID.
// Neither a populated ID nor IsNew returning false proves that insertion succeeded.
// Dao.Create returns the candidate model; use GORM RowsAffected to detect a skipped insert.
// Prefer UModel for new models. Its uuid.UUID primary key is generated as UUIDv7
// and bound by the RDB driver adapter. RDB connections store UUID columns as
// PostgreSQL uuid or SQLite TEXT.
// On RDB-managed connections, custom uuid.UUID and *uuid.UUID fields need no
// type or serializer tags: PostgreSQL uses uuid and SQLite uses TEXT automatically.
// Explicit type and serializer tags take precedence. The vine-rdb-uuid serializer
// remains available for explicitly tagged fields.
// A nil *uuid.UUID stores SQL NULL; a pointer to uuid.Nil() stores the zero UUID.
// Reading SQL NULL into a uuid.UUID value produces uuid.Nil().
// RDB connections also convert uuid.UUID SQL parameters automatically, including
// UUID slices expanded by GORM in expressions such as Where("id IN ?", ids).
// DAO queries also normalize UUID primary-key shorthand and map conditions.
//
// Direct GORM calls bypass DAO normalization. GORM can expand uuid.UUID as an
// array before the driver sees it. For a UUID id, use these forms:
//
//   - Instead of db.First(&row, id), use db.Where("id = ?", id).First(&row).
//   - In map conditions, use map[string]any{"id": id.String()}; for UUID lists,
//     prefer db.Where("id IN ?", ids) or convert the list to []string.
//   - For a scalar placeholder immediately following an opening parenthesis,
//     pass id.String(), for example db.Exec("INSERT INTO items (id) VALUES (?)", id.String())
//     or db.Where("id = (?)", id.String()). This also applies to Raw expressions.
//
// On RDB-managed connections, db.Where("id = ?", id), db.Where("id IN ?", ids),
// and db.Where("id IN (?)", ids) support UUID arguments directly. Here ids is a
// []uuid.UUID; a single UUID must not be used in place of the list.
// DAO equivalents such as dao.First(id) and dao.Query(map[string]any{"id": id})
// normalize UUID arguments automatically.
package rdb
