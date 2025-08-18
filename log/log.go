package log

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var logger *log.Logger

func init() {
	styles := log.DefaultStyles()
	styles.Levels[log.DebugLevel].UnsetMaxWidth().Width(5).Foreground(lipgloss.Color(lipgloss.Color("8")))
	styles.Levels[log.InfoLevel].UnsetMaxWidth().Width(5).Foreground(lipgloss.Color(lipgloss.Color("10")))
	styles.Levels[log.WarnLevel].UnsetMaxWidth().Width(5).Foreground(lipgloss.Color(lipgloss.Color("11")))
	styles.Levels[log.ErrorLevel].UnsetMaxWidth().Width(5)
	styles.Levels[log.FatalLevel].UnsetMaxWidth().Width(5)

	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Level:           log.InfoLevel,
	})

	logger.SetStyles(styles)
}

// Global returns the global logger, configured with sensible defaults.
func Default() log.Logger {
	return *logger
}

func Custom(options log.Options) *log.Logger {
	return log.NewWithOptions(os.Stderr, options)
}

// Keep in mind calling the package-level Debug logger will show up as if logger.go is the caller, not the location you called log.Debug from.
func Debug(msg string, keysAndValues ...interface{}) {
	logger.Debug(msg, keysAndValues...)
}

// Keep in mind calling the package-level Info logger will show up as if logger.go is the caller, not the location you called log.Info from.
func Info(msg string, keysAndValues ...interface{}) {
	logger.Info(msg, keysAndValues...)
}

// Keep in mind calling the package-level Warn logger will show up as if logger.go is the caller, not the location you called log.Warn from.
func Warn(msg string, keysAndValues ...interface{}) {
	logger.Warn(msg, keysAndValues...)
}

// Keep in mind calling the package-level Error logger will show up as if logger.go is the caller, not the location you called log.Error from.
func Error(msg string, keysAndValues ...interface{}) {
	logger.Error(msg, keysAndValues...)
}

// Keep in mind calling the package-level Fatal logger will show up as if logger.go is the caller, not the location you called log.Fatal from.
func Fatal(msg string, keysAndValues ...interface{}) {
	logger.Fatal(msg, keysAndValues...)
}

