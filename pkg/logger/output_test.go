package logger

import (
	"errors"
	"io"
	"math"
	"os"
	"strings"
	"testing"
)

var errSinkClosed = errors.New("sink closed")

// closedSink is an output whose writes fail.
type closedSink struct{}

func (closedSink) Write([]byte) (int, error) { return 0, errSinkClosed }

const testSinkMessage = "entry that must not vanish"

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stderr
	os.Stderr = writer
	fn()
	os.Stderr = original
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	captured, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(captured)
}

func TestWrite_AFailingSinkReportsTheEntryOnStderr(t *testing.T) {
	for _, format := range []string{formatJSON, levelNameInfo} {
		t.Run(format, func(t *testing.T) {
			l := &logger{level: InfoLevel, format: format, output: closedSink{}}

			stderr := captureStderr(t, func() { l.log(InfoLevel, testSinkMessage) })

			if !strings.Contains(stderr, errSinkClosed.Error()) || !strings.Contains(stderr, testSinkMessage) {
				t.Fatalf("stderr %q must name the sink failure and carry the entry", stderr)
			}
		})
	}
}

func TestOutputJSON_AnUnencodableEntryIsReportedOnStderr(t *testing.T) {
	l := &logger{level: InfoLevel, format: formatJSON, output: closedSink{}}

	stderr := captureStderr(t, func() { l.log(InfoLevel, testSinkMessage, FieldValue, math.NaN()) })

	if !strings.Contains(stderr, testSinkMessage) {
		t.Fatalf("stderr %q must carry the message of the entry that could not be encoded", stderr)
	}
}
