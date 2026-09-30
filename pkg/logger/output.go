package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Stderr reports for entries the configured output could not take.
const (
	sinkWriteFailureFmt  = "logger: write failed: %v: %s\n"
	sinkEncodeFailureFmt = "logger: encode failed: %v: %v\n"
)

// outputJSON formats the log entry as JSON
func (l *logger) outputJSON(fields map[string]interface{}) {
	data, err := json.Marshal(fields)
	if err != nil {
		fmt.Fprintf(os.Stderr, sinkEncodeFailureFmt, err, fields[entryKeyMessage])
		return
	}
	l.write(string(data))
}

// write sends one formatted entry to the output. A logger has no channel of
// its own left to report a failing sink on, so the entry goes to stderr.
func (l *logger) write(line string) {
	if _, err := fmt.Fprintln(l.output, line); err != nil {
		fmt.Fprintf(os.Stderr, sinkWriteFailureFmt, err, line)
	}
}

// outputText formats the log entry as human-readable text
func (l *logger) outputText(level Level, msg string, fields map[string]interface{}) {
	// Build the log line
	var sb strings.Builder

	// Timestamp
	sb.WriteString(time.Now().Format(time.DateTime))
	sb.WriteString(" ")

	// Level with color
	if l.useColor {
		sb.WriteString(levelColors[level])
	}
	fmt.Fprintf(&sb, "[%-5s]", levelNames[level])
	if l.useColor {
		sb.WriteString(resetColor)
	}
	sb.WriteString(" ")

	// Message
	sb.WriteString(msg)

	// Additional fields
	for key, value := range fields {
		if key == entryKeyTimestamp || key == entryKeyLevel || key == entryKeyMessage {
			continue
		}
		fmt.Fprintf(&sb, " %s=%v", key, value)
	}

	l.write(sb.String())
}
