package logger

import "time"

// Entry is a fully-resolved log record, handed to Hooks via Fire. It contains
// everything captured at the call site: time, level, message, layer, caller,
// and any structured fields.
type Entry struct {
	// Time is when the entry was created.
	Time time.Time
	// Level is the entry's severity.
	Level Level
	// Message is the human-readable log message.
	Message string
	// Layer is the layer name the logger was derived with, if any.
	Layer string
	// Caller is the caller location (file:line), when enabled.
	Caller string
	// Fields are the structured fields attached to the entry.
	Fields []Field
}
