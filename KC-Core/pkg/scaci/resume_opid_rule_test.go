package scaci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Counters a session stored before the connection loss.
const (
	ruleKnownAcOpID  int64 = 10
	ruleIssuedScOpID int64 = -5
)

// SCACI §3.3.1: snAcOpId is the minimum application center operation ID the
// service center must know, snScOpId the maximum service center operation ID
// the application center knows. The previous session is resumed when both
// sides are consistent (§1).
func TestResumeOpIDConflict(t *testing.T) {
	stored := OpIDPair{AC: ruleKnownAcOpID, SC: ruleIssuedScOpID}
	cases := []struct {
		name      string
		snAcOpID  int64
		snScOpID  int64
		resumable bool
	}{
		{name: "both sides agree", snAcOpID: ruleKnownAcOpID, snScOpID: ruleIssuedScOpID, resumable: true},
		{name: "service center processed an operation whose response was lost", snAcOpID: ruleKnownAcOpID - 1, snScOpID: ruleIssuedScOpID, resumable: true},
		{name: "application center missed service center operations", snAcOpID: ruleKnownAcOpID, snScOpID: ruleIssuedScOpID + 2, resumable: true},
		{name: "application center has seen no service center operation", snAcOpID: ruleKnownAcOpID, snScOpID: 0, resumable: true},
		{name: "service center lost an application center operation", snAcOpID: ruleKnownAcOpID + 1, snScOpID: ruleIssuedScOpID, resumable: false},
		{name: "application center knows a service center operation never issued", snAcOpID: ruleKnownAcOpID, snScOpID: ruleIssuedScOpID - 1, resumable: false},
		{name: "negative application center operation ID", snAcOpID: -1, snScOpID: ruleIssuedScOpID, resumable: false},
		{name: "positive service center operation ID", snAcOpID: ruleKnownAcOpID, snScOpID: 1, resumable: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason := ResumeOpIDConflict(tc.snAcOpID, tc.snScOpID, stored)
			assert.Equal(t, tc.resumable, reason == "", reason)
		})
	}
}
