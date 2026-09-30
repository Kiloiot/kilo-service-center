package bssci

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// Fixture log message and field value the observer must capture verbatim.
const (
	testWarnMessage    = "Test warning message"
	testWarnFieldValue = "value"
)

// TestObserverCapture verifies the recording logger captures WARN entries so
// log-assertion tests can rely on it.
func TestObserverCapture(t *testing.T) {
	testLogger := newRecordingLogger()

	ctx := testutil.TestContext()

	testLogger.WarnContext(ctx, testWarnMessage, logger.FieldField, testWarnFieldValue)

	warnLogs := testLogger.getEntriesByLevel("WARN")

	t.Logf("Total logs: %d, Warn logs: %d", len(testLogger.entries), len(warnLogs))

	if len(testLogger.entries) == 0 {
		t.Fatal("Recorder didn't capture any logs!")
	}
	if len(warnLogs) == 0 {
		t.Fatal("Recorder didn't capture the warn message!")
	}
	if warnLogs[0].msg != "Test warning message" {
		t.Fatalf("unexpected message %q", warnLogs[0].msg)
	}
}
