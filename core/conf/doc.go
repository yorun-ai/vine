// Package conf defines application configuration schemas, lifecycles, and readers.
// Readers trim leading and trailing Unicode whitespace from string fields,
// including nullable strings, list elements, and map values. Map keys, JSON
// contents, and named scalar types are preserved. Stored configuration is unchanged.
// Config fields tagged skel:"noTrim" preserve their string values, including
// nullable strings and collection values. Flags can be combined with commas,
// for example skel:"sensitive,noTrim".
package conf
