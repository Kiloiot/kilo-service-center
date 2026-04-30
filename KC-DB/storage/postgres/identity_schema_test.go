package postgres

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// identityTables lists the tables that migration 000116 moved to the identity schema.
var identityTables = []string{
	"users",
	"organizations",
	"organization_members",
	"api_keys",
	"refresh_tokens",
}

// TestIdentitySchemaOwnership verifies that the identity schema split is intact:
// each identity table exists as a real base table in the "identity" schema,
// and a compatibility view exists in the "public" schema.
func TestIdentitySchemaOwnership(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping identity schema test in short mode (requires database)")
	}

	requireDB := os.Getenv("CI") == "true"
	db := SetupEnvDBOrSkip(t, requireDB)

	for _, table := range identityTables {
		table := table
		t.Run("identity."+table+"_is_base_table", func(t *testing.T) {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM information_schema.tables
					WHERE table_schema = 'identity' AND table_name = $1 AND table_type = 'BASE TABLE'
				)`, table).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "identity.%s should be a base table in the identity schema", table)
		})

		t.Run("public."+table+"_is_view", func(t *testing.T) {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM information_schema.views
					WHERE table_schema = 'public' AND table_name = $1
				)`, table).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "public.%s should be a compatibility view", table)
		})
	}
}
