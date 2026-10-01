package postgres

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// userSchemaFilter restricts catalog queries to the schemas migrations own.
const userSchemaFilter = `n.nspname NOT IN ('pg_catalog', 'information_schema') AND n.nspname NOT LIKE 'pg_toast%'`

// catalogShapeQueries each return one text line per catalog object. Columns
// are compared by name, type, nullability, default and comment; view
// definitions carry their column order.
var catalogShapeQueries = []string{
	`SELECT format('column %s.%s.%s %s notnull=%s default=%s comment=%s', n.nspname, c.relname, a.attname,
	        format_type(a.atttypid, a.atttypmod), a.attnotnull, pg_get_expr(d.adbin, d.adrelid), col_description(c.oid, a.attnum))
	 FROM pg_attribute a
	 JOIN pg_class c ON c.oid = a.attrelid
	 JOIN pg_namespace n ON n.oid = c.relnamespace
	 LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
	 WHERE a.attnum > 0 AND NOT a.attisdropped AND c.relkind IN ('r', 'p', 'v', 'm')
	   AND c.relname <> 'schema_migrations' AND ` + userSchemaFilter,
	`SELECT format('table %s.%s kind=%s comment=%s', n.nspname, c.relname, c.relkind, obj_description(c.oid, 'pg_class'))
	 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
	 WHERE c.relkind IN ('r', 'p', 'v', 'm', 'S') AND c.relname <> 'schema_migrations' AND ` + userSchemaFilter,
	`SELECT format('constraint %s.%s %s', con.conrelid::regclass, con.conname, pg_get_constraintdef(con.oid))
	 FROM pg_constraint con JOIN pg_namespace n ON n.oid = con.connamespace
	 WHERE con.conrelid <> 0 AND con.conrelid::regclass::text <> 'schema_migrations' AND ` + userSchemaFilter,
	`SELECT format('index %s.%s %s', n.nspname, c.relname, pg_get_indexdef(c.oid))
	 FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid JOIN pg_namespace n ON n.oid = c.relnamespace
	 WHERE i.indrelid::regclass::text <> 'schema_migrations' AND ` + userSchemaFilter,
	`SELECT format('trigger %s.%s %s', t.tgrelid::regclass, t.tgname, pg_get_triggerdef(t.oid))
	 FROM pg_trigger t JOIN pg_class c ON c.oid = t.tgrelid JOIN pg_namespace n ON n.oid = c.relnamespace
	 WHERE NOT t.tgisinternal AND ` + userSchemaFilter,
	`SELECT format('view %s.%s %s', n.nspname, c.relname, pg_get_viewdef(c.oid))
	 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
	 WHERE c.relkind IN ('v', 'm') AND ` + userSchemaFilter,
	`SELECT format('function %s.%s(%s) %s', n.nspname, p.proname, pg_get_function_identity_arguments(p.oid), pg_get_functiondef(p.oid))
	 FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
	 WHERE p.prokind = 'f' AND ` + userSchemaFilter,
	`SELECT format('sequence %s.%s owned-by=%s', n.nspname, c.relname,
	        (SELECT d.refobjid::regclass || '.' || a.attname FROM pg_depend d
	         JOIN pg_attribute a ON a.attrelid = d.refobjid AND a.attnum = d.refobjsubid
	         WHERE d.objid = c.oid AND d.deptype = 'a'))
	 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
	 WHERE c.relkind = 'S' AND ` + userSchemaFilter,
	`SELECT format('type %s.%s %s', n.nspname, t.typname, string_agg(e.enumlabel, ',' ORDER BY e.enumsortorder))
	 FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace JOIN pg_enum e ON e.enumtypid = t.oid
	 WHERE ` + userSchemaFilter + ` GROUP BY n.nspname, t.typname`,
}

// captureCatalogShape lists every schema object migrations create, one line
// per object, sorted.
func captureCatalogShape(t *testing.T, db *sqlx.DB) []string {
	t.Helper()
	var shape []string
	for _, query := range catalogShapeQueries {
		var lines []string
		require.NoError(t, db.Select(&lines, query))
		shape = append(shape, lines...)
	}
	sort.Strings(shape)
	return shape
}

// shapeDifference returns the lines of want that got lacks.
func shapeDifference(want, got []string) []string {
	present := make(map[string]bool, len(got))
	for _, line := range got {
		present[line] = true
	}
	var missing []string
	for _, line := range want {
		if !present[line] {
			missing = append(missing, line)
		}
	}
	return missing
}

// TestDownMigrationsRestoreThePriorCatalog applies each migration and its down
// migration and requires the catalog to match the shape before the up
// migration: columns, constraints, indexes, triggers, views, functions,
// sequences and enum types.
func TestDownMigrationsRestoreThePriorCatalog(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration test in short mode")
	}
	for _, version := range []uint{145, 154, 155, 163} {
		t.Run(fmt.Sprintf("%06d", version), func(t *testing.T) {
			db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
			defer cleanup()
			m := newMigrator(t, db)

			require.NoError(t, m.Migrate(version-1))
			before := captureCatalogShape(t, db)
			require.NoError(t, m.Migrate(version))
			require.NoError(t, m.Migrate(version-1))
			after := captureCatalogShape(t, db)

			assert.Empty(t, strings.Join(shapeDifference(before, after), "\n"), "lost by the down migration")
			assert.Empty(t, strings.Join(shapeDifference(after, before), "\n"), "left behind by the down migration")
		})
	}
}
