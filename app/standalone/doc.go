// Package standalone runs Vine applications with an in-process Hub, Portal, and Link.
// Without SQLiteFile or PostgresURL it requires a seed YAML file and loads
// read-only configuration into memory; edit the file and restart to apply changes.
package standalone
