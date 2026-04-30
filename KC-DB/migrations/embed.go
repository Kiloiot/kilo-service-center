// Package migrations provides embedded SQL migration files for KiloCenter database schema management.
package migrations

import "embed"

// MigrationsFS embeds all SQL migration files for use with golang-migrate.
//
//go:embed *.sql
var MigrationsFS embed.FS
