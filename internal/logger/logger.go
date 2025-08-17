package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type Logger struct {
	prefix string
}

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
)

const (
	SymbolInfo    = "ℹ"
	SymbolSuccess = "✓"
	SymbolWarning = "⚠"
	SymbolError   = "✗"
	SymbolDebug   = "→"
)

var defaultLogger = &Logger{prefix: "SYSTEM"}

func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

func GetLogger() *Logger {
	return defaultLogger
}

func GetAPILogger() *Logger {
	return &Logger{prefix: "API"}
}

func GetTunnelLogger() *Logger {
	return &Logger{prefix: "TUNNEL"}
}

func (l *Logger) formatMessage(level, symbol, color, message string, args ...any) {
	timestamp := time.Now().Format("15:04:05.000")

	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	// Format: [15:04:05.000] ✓ [API] Message
	logLine := fmt.Sprintf("%s[%s]%s %s%s%s %s[%s]%s %s",
		Dim, timestamp, Reset,
		color, symbol, Reset,
		Dim, l.prefix, Reset,
		message,
	)

	// Use standard log to maintain consistency with existing logs
	log.Print(logLine)
}

func (l *Logger) Info(message string, args ...any) {
	l.formatMessage("INFO", SymbolInfo, Blue, message, args...)
}

func (l *Logger) Success(message string, args ...any) {
	l.formatMessage("SUCCESS", SymbolSuccess, Green, message, args...)
}

func (l *Logger) Warning(message string, args ...any) {
	l.formatMessage("WARNING", SymbolWarning, Yellow, message, args...)
}

func (l *Logger) Error(message string, args ...any) {
	l.formatMessage("ERROR", SymbolError, Red, message, args...)
}

func (l *Logger) Debug(message string, args ...any) {
	if os.Getenv("DEBUG") != "" {
		l.formatMessage("DEBUG", SymbolDebug, Gray, message, args...)
	}
}

func (l *Logger) Step(step int, total int, message string, args ...any) {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	stepMsg := fmt.Sprintf("Step %d/%d: %s", step, total, message)
	l.formatMessage("STEP", SymbolDebug, Cyan, stepMsg)
}

func (l *Logger) Request(method, path, clientIP string) {
	message := fmt.Sprintf("%s %s from %s",
		strings.ToUpper(method),
		path,
		clientIP,
	)
	l.formatMessage("REQUEST", SymbolDebug, Purple, message)
}

func (l *Logger) Response(method, path string, statusCode int, duration time.Duration) {
	var color string
	var symbol string

	switch {
	case statusCode >= 200 && statusCode < 300:
		color = Green
		symbol = SymbolSuccess
	case statusCode >= 300 && statusCode < 400:
		color = Blue
		symbol = SymbolInfo
	case statusCode >= 400 && statusCode < 500:
		color = Yellow
		symbol = SymbolWarning
	case statusCode >= 500:
		color = Red
		symbol = SymbolError
	default:
		color = Gray
		symbol = SymbolDebug
	}

	message := fmt.Sprintf("%s %s %d %s",
		strings.ToUpper(method),
		path,
		statusCode,
		duration.Round(time.Millisecond),
	)

	l.formatMessage("RESPONSE", symbol, color, message)
}

func (l *Logger) SSHConnect(user, host string, port int) {
	message := fmt.Sprintf("Connecting to %s@%s:%d", user, host, port)
	l.formatMessage("SSH", SymbolDebug, Cyan, message)
}

func (l *Logger) SSHConnected(user, host string) {
	message := fmt.Sprintf("Connected to %s@%s", user, host)
	l.formatMessage("SSH", SymbolSuccess, Green, message)
}

func (l *Logger) SSHDisconnected(host string) {
	message := fmt.Sprintf("Disconnected from %s", host)
	l.formatMessage("SSH", SymbolInfo, Blue, message)
}

func (l *Logger) SSHCommand(cmd string) {
	message := fmt.Sprintf("Executing: %s", cmd)
	l.formatMessage("CMD", SymbolDebug, Purple, message)
}

func (l *Logger) SSHCommandResult(cmd string, exitCode int, duration time.Duration) {
	var color string
	var symbol string

	if exitCode == 0 {
		color = Green
		symbol = SymbolSuccess
	} else {
		color = Red
		symbol = SymbolError
	}

	message := fmt.Sprintf("Command completed [%d] %s (%s)", exitCode, cmd, duration.Round(time.Millisecond))
	l.formatMessage("CMD", symbol, color, message)
}

func (l *Logger) FileTransfer(operation, local, remote string) {
	message := fmt.Sprintf("%s %s → %s", operation, local, remote)
	l.formatMessage("FILE", SymbolDebug, Cyan, message)
}

func (l *Logger) FileTransferComplete(operation string, err error) {
	if err != nil {
		l.formatMessage("FILE", SymbolError, Red, "%s failed: %v", operation, err)
	} else {
		l.formatMessage("FILE", SymbolSuccess, Green, "%s completed", operation)
	}
}

func (l *Logger) SystemOperation(operation string) {
	l.formatMessage("SYS", SymbolDebug, Yellow, operation)
}

func Info(message string, args ...any) {
	defaultLogger.Info(message, args...)
}

func Success(message string, args ...any) {
	defaultLogger.Success(message, args...)
}

func Warning(message string, args ...any) {
	defaultLogger.Warning(message, args...)
}

func Error(message string, args ...any) {
	defaultLogger.Error(message, args...)
}

func Debug(message string, args ...any) {
	defaultLogger.Debug(message, args...)
}

func Step(step int, total int, message string, args ...any) {
	defaultLogger.Step(step, total, message, args...)
}

func Request(method, path, clientIP string) {
	defaultLogger.Request(method, path, clientIP)
}

func Response(method, path string, statusCode int, duration time.Duration) {
	defaultLogger.Response(method, path, statusCode, duration)
}

func SSHConnect(user, host string, port int) {
	defaultLogger.SSHConnect(user, host, port)
}

func SSHConnected(user, host string) {
	defaultLogger.SSHConnected(user, host)
}

func SSHDisconnected(host string) {
	defaultLogger.SSHDisconnected(host)
}

func SSHCommand(cmd string) {
	defaultLogger.SSHCommand(cmd)
}

func SSHCommandResult(cmd string, exitCode int, duration time.Duration) {
	defaultLogger.SSHCommandResult(cmd, exitCode, duration)
}

func FileTransfer(operation, local, remote string) {
	defaultLogger.FileTransfer(operation, local, remote)
}

func FileTransferComplete(operation string, err error) {
	defaultLogger.FileTransferComplete(operation, err)
}

func SystemOperation(operation string) {
	defaultLogger.SystemOperation(operation)
}
