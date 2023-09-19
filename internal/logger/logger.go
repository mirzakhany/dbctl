package logger

import "log"

// Provider is the interface that wraps the Println method.
type Provider interface {
	Println(v ...any)
}

// LogLevel is the log level type.
type LogLevel int

const (
	// LevelDebug is the debug log level
	LevelDebug LogLevel = iota
	// LevelInfo is the info log level
	LevelInfo
	// LevelWarn is the warn log level
	LevelWarn
	// LevelError is the error log level
	LevelError
)

func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

// Logger is the logger object
type Logger struct {
	provider Provider

	logLvl LogLevel
}

// New creates a new logger with the given provider.
func New(provider Provider, loglvl LogLevel) *Logger {
	return &Logger{provider: provider, logLvl: loglvl}
}

// SetProvider sets the logger provider.
func SetProvider(provider Provider) {
	logger.provider = provider
}

// SetLevel sets the logger level.
func SetLevel(loglvl LogLevel) {
	if loglvl < LevelDebug || loglvl > LevelError {
		return
	}
	logger.logLvl = loglvl
}

// logger is the default logger.
var logger = New(log.Default(), LevelDebug)

// Println calls the Println method of the logger provider.
func Println(v ...any) {
	logger.provider.Println(v...)
}

// Info calls the Println method of the logger provider with INFO prefix.
func Info(v ...any) {
	if logger.logLvl > LevelInfo {
		return
	}
	printWithLevel(LevelInfo, v...)
}

// Debug calls the Println method of the logger provider with DEBUG prefix.
func Debug(v ...any) {
	if logger.logLvl > LevelDebug {
		return
	}
	printWithLevel(LevelDebug, v...)
}

// Warn calls the Println method of the logger provider with WARN prefix.
func Warn(v ...any) {
	if logger.logLvl > LevelWarn {
		return
	}
	printWithLevel(LevelWarn, v...)
}

// Error calls the Println method of the logger provider with ERROR prefix.
func Error(v ...any) {
	if logger.logLvl > LevelError {
		return
	}
	printWithLevel(LevelError, v...)
}

func printWithLevel(lvl LogLevel, v ...any) {
	logger.provider.Println(append([]any{lvl.String() + ":"}, v...)...)
}
