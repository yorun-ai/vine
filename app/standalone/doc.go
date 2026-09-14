// Package standalone runs Vine applications with an in-process Hub, Portal, and Link.
// Without SQLiteFile or PostgresURL it requires SeedHubDataFile or inline SeedHubData
// and loads read-only configuration into memory. Update the seed source and
// restart to apply changes. The two seed sources are mutually exclusive.
package standalone
