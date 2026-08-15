package ember

import "github.com/NeRo0128/ember/logger"

// defaultLogger is the package-level logger backing the global log functions.
// It defaults to InfoLevel and text output to os.Stdout.
var defaultLogger = logger.NewLogger()

// LogSuccess logs an informational success message via the default logger.
func LogSuccess(msg string) {
	defaultLogger.Info(msg)
}

// LogInfo logs an informational message via the default logger.
func LogInfo(msg string) {
	defaultLogger.Info(msg)
}

// LogWarning logs a warning message via the default logger.
func LogWarning(msg string) {
	defaultLogger.Warn(msg)
}

// LogError logs an error message via the default logger.
func LogError(msg string) {
	defaultLogger.Error(msg)
}
