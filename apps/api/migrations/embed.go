// Package migrations holds the database schema as goose SQL migrations.
// The API embeds them to migrate at startup, and sqlc reads the same
// files as its schema.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
