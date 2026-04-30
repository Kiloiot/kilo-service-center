package bssci

import (
	"context"
	"testing"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// Quick test to verify observer captures WARN logs
func TestObserverCapture(t *testing.T) {
	observedCore, observedLogs := observer.New(zap.WarnLevel)
	testLogger := logger.FromZap(zap.New(observedCore))

	ctx := context.Background()

	// Log a WARN
	testLogger.WarnContext(ctx, "Test warning message", "field", "value")

	// Check if captured
	allLogs := observedLogs.All()
	warnLogs := observedLogs.FilterMessage("Test warning message").All()

	t.Logf("Total logs: %d, Warn logs: %d", len(allLogs), len(warnLogs))

	if len(allLogs) == 0 {
		t.Fatal("Observer didn't capture any logs!")
	}
	if len(warnLogs) == 0 {
		t.Fatal("Observer didn't capture the warn message!")
	}
}
