package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Color codes for console output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
	ColorBold   = "\033[1m"
)

// Logger represents a structured logger with level control and colored output
type Logger struct {
	mu          sync.RWMutex
	level       LogLevel
	output      io.Writer
	enableColor bool
	prefix      string
	fields      map[string]interface{}
}

// LogEntry represents a single log entry with structured data
type LogEntry struct {
	logger    *Logger
	level     LogLevel
	message   string
	fields    map[string]interface{}
	timestamp time.Time
	caller    string
}

// New creates a new logger instance with default settings
func New() *Logger {
	return &Logger{
		level:       InfoLevel,
		output:      os.Stdout,
		enableColor: isTerminal(os.Stdout),
		fields:      make(map[string]interface{}),
	}
}

// NewWithOutput creates a new logger with a specific output writer
func NewWithOutput(output io.Writer) *Logger {
	return &Logger{
		level:       InfoLevel,
		output:      output,
		enableColor: isTerminal(output),
		fields:      make(map[string]interface{}),
	}
}

// SetLevel sets the minimum log level that will be output
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetOutput sets the output destination for the logger
func (l *Logger) SetOutput(output io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = output
	l.enableColor = isTerminal(output)
}

// EnableColor enables or disables colored output
func (l *Logger) EnableColor(enable bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enableColor = enable
}

// SetPrefix sets a prefix for all log messages
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// WithField returns a new logger entry with the specified field
func (l *Logger) WithField(key string, value interface{}) *LogEntry {
	l.mu.RLock()
	fields := make(map[string]interface{})
	for k, v := range l.fields {
		fields[k] = v
	}
	l.mu.RUnlock()

	fields[key] = value

	return &LogEntry{
		logger:    l,
		fields:    fields,
		timestamp: time.Now(),
	}
}

// WithFields returns a new logger entry with multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *LogEntry {
	l.mu.RLock()
	allFields := make(map[string]interface{})
	for k, v := range l.fields {
		allFields[k] = v
	}
	l.mu.RUnlock()

	for k, v := range fields {
		allFields[k] = v
	}

	return &LogEntry{
		logger:    l,
		fields:    allFields,
		timestamp: time.Now(),
	}
}

// WithContext returns a new logger entry with context values
func (l *Logger) WithContext(ctx context.Context) *LogEntry {
	entry := &LogEntry{
		logger:    l,
		fields:    make(map[string]interface{}),
		timestamp: time.Now(),
	}

	// Extract common context values if they exist
	if requestID := ctx.Value("request_id"); requestID != nil {
		entry.fields["request_id"] = requestID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		entry.fields["user_id"] = userID
	}

	return entry
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(DebugLevel, msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(InfoLevel, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(WarnLevel, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(ErrorLevel, msg, args...)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(FatalLevel, msg, args...)
	os.Exit(1)
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, msg string, args ...interface{}) {
	l.mu.RLock()
	if level < l.level {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	entry := &LogEntry{
		logger:    l,
		level:     level,
		message:   fmt.Sprintf(msg, args...),
		fields:    make(map[string]interface{}),
		timestamp: time.Now(),
		caller:    getCaller(),
	}

	entry.write()
}

// LogEntry methods

// WithField adds a field to this log entry
func (e *LogEntry) WithField(key string, value interface{}) *LogEntry {
	e.fields[key] = value
	return e
}

// WithFields adds multiple fields to this log entry
func (e *LogEntry) WithFields(fields map[string]interface{}) *LogEntry {
	for k, v := range fields {
		e.fields[k] = v
	}
	return e
}

// WithError adds an error field to this log entry
func (e *LogEntry) WithError(err error) *LogEntry {
	if err != nil {
		e.fields["error"] = err.Error()
	}
	return e
}

// Debug logs this entry as debug level
func (e *LogEntry) Debug(msg string, args ...interface{}) {
	e.level = DebugLevel
	e.message = fmt.Sprintf(msg, args...)
	e.write()
}

// Info logs this entry as info level
func (e *LogEntry) Info(msg string, args ...interface{}) {
	e.level = InfoLevel
	e.message = fmt.Sprintf(msg, args...)
	e.write()
}

// Warn logs this entry as warning level
func (e *LogEntry) Warn(msg string, args ...interface{}) {
	e.level = WarnLevel
	e.message = fmt.Sprintf(msg, args...)
	e.write()
}

// Error logs this entry as error level
func (e *LogEntry) Error(msg string, args ...interface{}) {
	e.level = ErrorLevel
	e.message = fmt.Sprintf(msg, args...)
	e.write()
}

// Fatal logs this entry as fatal level and exits
func (e *LogEntry) Fatal(msg string, args ...interface{}) {
	e.level = FatalLevel
	e.message = fmt.Sprintf(msg, args...)
	e.write()
	os.Exit(1)
}

// log logs this entry at the specified level (internal method)
func (e *LogEntry) log(level LogLevel, msg string, args ...interface{}) {
	e.level = level
	e.message = fmt.Sprintf(msg, args...)
	e.write()
}

// write outputs the log entry
func (e *LogEntry) write() {
	e.logger.mu.RLock()
	defer e.logger.mu.RUnlock()

	if e.level < e.logger.level {
		return
	}

	// Format the log message
	var output strings.Builder

	// Timestamp
	timestamp := e.timestamp.Format("2006-01-02 15:04:05.000")

	// Level with color
	levelStr := e.formatLevel()

	// Caller info
	caller := e.caller
	if caller == "" {
		caller = getCaller()
	}

	// Prefix
	prefix := ""
	if e.logger.prefix != "" {
		prefix = fmt.Sprintf("[%s] ", e.logger.prefix)
	}

	// Build the main message
	output.WriteString(fmt.Sprintf("%s %s %s%s: %s",
		timestamp, levelStr, prefix, caller, e.message))

	// Add fields if any
	if len(e.fields) > 0 {
		output.WriteString(" |")
		for key, value := range e.fields {
			output.WriteString(fmt.Sprintf(" %s=%v", key, value))
		}
	}

	output.WriteString("\n")

	// Write to output
	fmt.Fprint(e.logger.output, output.String())
}

// formatLevel formats the log level with optional colors
func (e *LogEntry) formatLevel() string {
	level := e.level.String()

	if !e.logger.enableColor {
		return fmt.Sprintf("[%s]", level)
	}

	var color string
	switch e.level {
	case DebugLevel:
		color = ColorGray
	case InfoLevel:
		color = ColorBlue
	case WarnLevel:
		color = ColorYellow
	case ErrorLevel:
		color = ColorRed
	case FatalLevel:
		color = ColorRed + ColorBold
	default:
		color = ColorReset
	}

	return fmt.Sprintf("%s[%s]%s", color, level, ColorReset)
}

// getCaller returns information about the caller
func getCaller() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown"
	}

	// Get just the filename, not the full path
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return fmt.Sprintf("%s:%d", file, line)
}

// isTerminal checks if the output is a terminal (for color support)
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		// Check if it's stdout, stderr, or stdin
		return f == os.Stdout || f == os.Stderr || f == os.Stdin
	}
	return false
}

// Global logger instance
var defaultLogger = New()

// Package-level functions that use the default logger

// SetLevel sets the log level for the default logger
func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// SetOutput sets the output for the default logger
func SetOutput(output io.Writer) {
	defaultLogger.SetOutput(output)
}

// EnableColor enables or disables colors for the default logger
func EnableColor(enable bool) {
	defaultLogger.EnableColor(enable)
}

// SetPrefix sets a prefix for the default logger
func SetPrefix(prefix string) {
	defaultLogger.SetPrefix(prefix)
}

// WithField returns a log entry with a field using the default logger
func WithField(key string, value interface{}) *LogEntry {
	return defaultLogger.WithField(key, value)
}

// WithFields returns a log entry with fields using the default logger
func WithFields(fields map[string]interface{}) *LogEntry {
	return defaultLogger.WithFields(fields)
}

// WithContext returns a log entry with context using the default logger
func WithContext(ctx context.Context) *LogEntry {
	return defaultLogger.WithContext(ctx)
}

// WithError returns a log entry with an error using the default logger
func WithError(err error) *LogEntry {
	return defaultLogger.WithField("error", err.Error())
}

// Debug logs a debug message using the default logger
func Debug(msg string, args ...interface{}) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info message using the default logger
func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message using the default logger
func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message using the default logger
func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}

// Fatal logs a fatal message using the default logger and exits
func Fatal(msg string, args ...interface{}) {
	defaultLogger.Fatal(msg, args...)
}

// NewBasicConsoleLogger creates a basic logger optimized for console output
func NewBasicConsoleLogger(level LogLevel) *Logger {
	logger := New()
	logger.SetLevel(level)
	logger.EnableColor(true)
	return logger
}

// NewFileLogger creates a logger optimized for file output
func NewFileLogger(filename string, level LogLevel) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", filename, err)
	}

	logger := NewWithOutput(file)
	logger.SetLevel(level)
	logger.EnableColor(false) // No colors for file output
	return logger, nil
}

// Performance and progress logging utilities

// Progress logs progress information with percentage
func Progress(msg string, current, total int, args ...interface{}) {
	percentage := float64(current) / float64(total) * 100
	message := fmt.Sprintf(msg, args...)
	WithFields(map[string]interface{}{
		"current":    current,
		"total":      total,
		"percentage": fmt.Sprintf("%.1f%%", percentage),
	}).Info("Progress: %s", message)
}

// Success logs a success message with green color emphasis
func Success(msg string, args ...interface{}) {
	if defaultLogger.enableColor {
		fmt.Fprintf(defaultLogger.output, "%s✓%s ", ColorGreen, ColorReset)
	}
	Info(msg, args...)
}

// Failure logs a failure message with red color emphasis
func Failure(msg string, args ...interface{}) {
	if defaultLogger.enableColor {
		fmt.Fprintf(defaultLogger.output, "%s✗%s ", ColorRed, ColorReset)
	}
	Error(msg, args...)
}

// Step logs a step in a process
func Step(step string, msg string, args ...interface{}) {
	WithField("step", step).Info(msg, args...)
}

// Timer helps measure and log execution time
type Timer struct {
	name      string
	startTime time.Time
	logger    *Logger
}

// StartTimer creates and starts a new timer
func StartTimer(name string) *Timer {
	return &Timer{
		name:      name,
		startTime: time.Now(),
		logger:    defaultLogger,
	}
}

// Stop stops the timer and logs the duration
func (t *Timer) Stop() {
	duration := time.Since(t.startTime)
	t.logger.WithFields(map[string]interface{}{
		"operation": t.name,
		"duration":  duration.String(),
	}).Info("Operation completed")
}

// StopWithMessage stops the timer and logs a custom message with duration
func (t *Timer) StopWithMessage(msg string, args ...interface{}) {
	duration := time.Since(t.startTime)
	t.logger.WithFields(map[string]interface{}{
		"operation": t.name,
		"duration":  duration.String(),
	}).Info(msg, args...)
}
