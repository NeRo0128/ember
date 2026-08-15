package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

// FormatJSON serializes a log entry as JSON, indented with two spaces when
// pretty is true.
func FormatJSON(entry map[string]interface{}, pretty bool) ([]byte, error) { // todo: add colors distinguishing fields from their values
	if pretty {
		return json.MarshalIndent(entry, "", "  ")
	}
	return json.Marshal(entry)
}

// FormatText renders a log entry as a single line of text: timestamp, level
// (colored when stdout is a terminal), layer, message, caller, and any extra
// fields.
func FormatText(entry map[string]interface{}, level string, pretty bool) string {
	var sb strings.Builder

	// timestamp
	ts, _ := entry["ts"].(string)
	if ts == "" {
		ts = time.Now().Format(time.RFC3339)
	}
	sb.WriteString(ts)

	sb.WriteString(" ")
	if isTerminal() {
		sb.WriteString(applyColor(level, fmt.Sprintf(" [%s]", level)))
	} else {
		sb.WriteString(fmt.Sprintf("[%s]", level))
	}

	// layer
	if layer, ok := entry["layer"].(string); ok && layer != "" {
		sb.WriteString(fmt.Sprintf(" [%s]", strings.ToUpper(layer)))
	} else {
		sb.WriteString(" [UNKNOWN]")
	}

	// message
	if msg, ok := entry["msg"].(string); ok && msg != "" {
		sb.WriteString(fmt.Sprintf(" %s", msg))
	}

	// caller
	if caller, ok := entry["caller"].(string); ok && caller != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", filepath.Base(caller)))
	}

	for k, v := range entry {
		if k == "ts" || k == "lvl" || k == "msg" || k == "layer" || k == `caller` {
			continue
		}
		sb.WriteString(fmt.Sprintf(" %s: %v", k, v))
	}

	return sb.String()
}

func isTerminal() bool {
	fd := int(os.Stdout.Fd())
	return term.IsTerminal(fd)
}

// applyColor wraps text in an ANSI color escape sequence for the given log
// level, or returns it unchanged if the level has no color (only used when
// stdout is a TTY).
func applyColor(level, text string) string {
	switch level {
	case "DEBUG":
		return fmt.Sprintf("\x1b[36m%s\x1b[0m", text)
	case "INFO":
		return fmt.Sprintf("\x1b[32m%s\x1b[0m", text)
	case "WARN":
		return fmt.Sprintf("\x1b[33m%s\x1b[0m", text)
	case "ERROR":
		return fmt.Sprintf("\x1b[31m%s\x1b[0m", text)
	case "FATAL":
		return fmt.Sprintf("\x1b[41m\x1b[37m%s\x1b[0m", text)
	default:
		return text
	}
}

// FormatField serializes a field value for display, JSON-encoding complex
// values (structs, maps) and indenting them when pretty is true.
func FormatField(field any, pretty bool) string {
	j, err := json.Marshal(field)
	if err != nil {
		return fmt.Sprintf("%v", field)
	}
	if pretty {
		var indented bytes.Buffer
		json.Indent(&indented, j, "", "  ")
		return indented.String()
	}
	return string(j)
}

// FormatLog renders a complete log entry as JSON or text depending on its
// "lvl" value: "JSON" uses FormatJSON, anything else uses FormatText.
func FormatLog(entry map[string]any, level string, pretty bool) string {
	if entry["lvl"] == "JSON" {
		j, _ := FormatJSON(entry, pretty)
		return string(j)
	}
	return FormatText(entry, level, pretty)
}
