// Package conf defines application configuration schemas, lifecycles, and readers.
// Readers trim leading and trailing Unicode whitespace from string fields,
// including nullable strings, list elements, and map values. Map keys, JSON
// contents, and named scalar types are preserved. Stored configuration is unchanged.
package conf
