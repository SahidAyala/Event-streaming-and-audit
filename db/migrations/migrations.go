// Package migrations embeds all versioned SQL migration files.
// Import this package to get access to the migration filesystem.
package migrations

import "embed"

// FS contains all *.sql files from this directory, sorted by filename.
// Files must be named NNN_description.sql (e.g. 001_create_events.sql)
// so lexicographic sort equals execution order.
//
//go:embed *.sql
var FS embed.FS
