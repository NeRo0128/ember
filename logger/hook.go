package logger

import "slices"

// Hook is an interceptor invoked after a log entry is written. Use it for
// side effects such as Slack notifications, Prometheus metrics, auditing, or
// alerts. Hooks must not block for long periods.
type Hook interface {
	// Levels returns the log levels this hook reacts to.
	Levels() []Level
	// Fire runs with the full Entry. It should return quickly and may return
	// an error to report a failure.
	Fire(entry Entry) error
}

// hookLevelsContain reports whether target is present in the given levels.
func hookLevelsContain(levels []Level, target Level) bool {
	return slices.Contains(levels, target)
}
