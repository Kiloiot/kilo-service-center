package bssci

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// A row must name how a base station's copy is sequenced: a row without a
// role would let a new base-station command skip the operation ID rules of
// rev1 §5.2 / classic §3.2, so the registry refuses it at startup.
func TestCommandRegistryRefusesARowWithoutRole(t *testing.T) {
	table := []CommandSpec{{Command: mioty.CmdPing, Handler: (*Server).handlePing}}

	_, err := newCommandRegistry(table, commandDirectionMap)

	require.Error(t, err, "a row without a role must be refused")
	require.ErrorIs(t, err, errCommandSpecWithoutRole)
}

// A role must match the direction of its command: only a command the service
// center alone sends goes unsequenced.
func TestCommandRegistryRefusesARoleItsDirectionContradicts(t *testing.T) {
	for name, spec := range map[string]CommandSpec{
		"base station command marked service-center only": {Command: mioty.CmdAttach, Role: roleServiceCenterOnly},
		"service center command given a sequenced role":   {Command: mioty.CmdDLDataQueue, Role: roleResponds},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := newCommandRegistry([]CommandSpec{spec}, commandDirectionMap)

			require.ErrorIs(t, err, errCommandSpecRoleContradictsDirection)
		})
	}
}
