package tracer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ConsoleFormatter formats trace data for console output
type ConsoleFormatter struct {
	EnableColors     bool
	ShowTimestamp    bool
	ShowCaller       bool
	TimestampFormat  string
	IndentSize       int
	MaxFieldValueLen int
}

// NewConsoleFormatter creates a new console formatter
func NewConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{
		EnableColors:     true,
		ShowTimestamp:    true,
		ShowCaller:       false,
		TimestampFormat:  "15:04:05.000",
		IndentSize:       2,
		MaxFieldValueLen: 100,
	}
}

// Format formats a span for console output
func (f *ConsoleFormatter) Format(span *SpanData) ([]byte, error) {
	var buf bytes.Buffer

	// Format timestamp
	if f.ShowTimestamp {
		timestamp := span.StartTime.Format(f.TimestampFormat)
		if f.EnableColors {
			buf.WriteString(f.color(ColorGray))
		}
		buf.WriteString(timestamp)
		buf.WriteString(" ")
		if f.EnableColors {
			buf.WriteString(f.color(ColorReset))
		}
	}

	// Format status
	statusColor := f.getStatusColor(span.Status)
	statusSymbol := f.getStatusSymbol(span.Status)

	if f.EnableColors {
		buf.WriteString(f.color(statusColor))
	}
	buf.WriteString(statusSymbol)
	buf.WriteString(" ")
	if f.EnableColors {
		buf.WriteString(f.color(ColorReset))
	}

	// Format operation
	if f.EnableColors {
		buf.WriteString(f.color(ColorBold))
	}
	buf.WriteString(span.Operation)
	if f.EnableColors {
		buf.WriteString(f.color(ColorReset))
	}

	// Format duration
	if !span.EndTime.IsZero() {
		buf.WriteString(" ")
		if f.EnableColors {
			buf.WriteString(f.color(ColorCyan))
		}
		buf.WriteString(fmt.Sprintf("(%s)", span.Duration.Round(time.Microsecond)))
		if f.EnableColors {
			buf.WriteString(f.color(ColorReset))
		}
	}

	// Format error if present
	if span.Error != nil {
		buf.WriteString(" ")
		if f.EnableColors {
			buf.WriteString(f.color(ColorRed))
		}
		buf.WriteString(fmt.Sprintf("error=%s", span.Error.Error()))
		if f.EnableColors {
			buf.WriteString(f.color(ColorReset))
		}
	}

	// Format fields
	if len(span.Fields) > 0 {
		buf.WriteString(" ")
		buf.WriteString(f.formatFields(span.Fields))
	}

	buf.WriteString("\n")

	// Format events
	for _, event := range span.Events {
		eventBytes, _ := f.FormatEvent(event)
		buf.WriteString(string(eventBytes))
	}

	return buf.Bytes(), nil
}

// FormatEvent formats an event for console output
func (f *ConsoleFormatter) FormatEvent(event Event) ([]byte, error) {
	var buf bytes.Buffer

	// Indent for event
	buf.WriteString(strings.Repeat(" ", f.IndentSize))

	// Event timestamp
	if f.ShowTimestamp {
		timestamp := event.Timestamp.Format(f.TimestampFormat)
		if f.EnableColors {
			buf.WriteString(f.color(ColorGray))
		}
		buf.WriteString(timestamp)
		buf.WriteString(" ")
		if f.EnableColors {
			buf.WriteString(f.color(ColorReset))
		}
	}

	// Event symbol
	if f.EnableColors {
		buf.WriteString(f.color(ColorYellow))
	}
	buf.WriteString("→")
	buf.WriteString(" ")
	if f.EnableColors {
		buf.WriteString(f.color(ColorReset))
	}

	// Event name
	buf.WriteString(event.Name)

	// Event fields
	if len(event.Fields) > 0 {
		buf.WriteString(" ")
		buf.WriteString(f.formatFields(event.Fields))
	}

	buf.WriteString("\n")

	return buf.Bytes(), nil
}

func (f *ConsoleFormatter) formatFields(fields Fields) string {
	if len(fields) == 0 {
		return ""
	}

	var parts []string
	for k, v := range fields {
		value := f.formatValue(v)
		if f.EnableColors {
			parts = append(parts, fmt.Sprintf("%s%s%s=%s",
				f.color(ColorBlue), k, f.color(ColorReset), value))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", k, value))
		}
	}

	return strings.Join(parts, " ")
}

func (f *ConsoleFormatter) formatValue(v any) string {
	var str string
	switch val := v.(type) {
	case string:
		str = val
	case error:
		str = val.Error()
	case time.Time:
		str = val.Format(time.RFC3339)
	case time.Duration:
		str = val.String()
	default:
		str = fmt.Sprintf("%v", v)
	}

	// Truncate long values
	if f.MaxFieldValueLen > 0 && len(str) > f.MaxFieldValueLen {
		str = str[:f.MaxFieldValueLen-3] + "..."
	}

	// Quote strings with spaces
	if strings.Contains(str, " ") {
		str = fmt.Sprintf("%q", str)
	}

	return str
}

func (f *ConsoleFormatter) getStatusColor(status Status) string {
	switch status {
	case StatusOK:
		return ColorGreen
	case StatusError:
		return ColorRed
	case StatusCanceled:
		return ColorYellow
	case StatusTimeout:
		return ColorYellow
	default:
		return ColorGray
	}
}

func (f *ConsoleFormatter) getStatusSymbol(status Status) string {
	switch status {
	case StatusOK:
		return "✓"
	case StatusError:
		return "✗"
	case StatusCanceled:
		return "⊘"
	case StatusTimeout:
		return "⏱"
	default:
		return "•"
	}
}

func (f *ConsoleFormatter) color(code string) string {
	if !f.EnableColors {
		return ""
	}
	return code
}

// Color codes for console output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
	ColorBold   = "\033[1m"
)

// JSONFormatter formats trace data as JSON
type JSONFormatter struct {
	Pretty bool
	Indent string
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(pretty bool) *JSONFormatter {
	return &JSONFormatter{
		Pretty: pretty,
		Indent: "  ",
	}
}

// Format formats a span as JSON
func (f *JSONFormatter) Format(span *SpanData) ([]byte, error) {
	data := f.spanToMap(span)

	if f.Pretty {
		return json.MarshalIndent(data, "", f.Indent)
	}
	return json.Marshal(data)
}

// FormatEvent formats an event as JSON
func (f *JSONFormatter) FormatEvent(event Event) ([]byte, error) {
	data := f.eventToMap(event)

	if f.Pretty {
		return json.MarshalIndent(data, "", f.Indent)
	}
	return json.Marshal(data)
}

func (f *JSONFormatter) spanToMap(span *SpanData) map[string]any {
	data := map[string]any{
		"trace_id":   fmt.Sprintf("%x", span.TraceID),
		"span_id":    fmt.Sprintf("%x", span.SpanID),
		"operation":  span.Operation,
		"start_time": span.StartTime.Format(time.RFC3339Nano),
		"status":     span.Status.String(),
	}

	if span.ParentSpanID != (SpanID{}) {
		data["parent_span_id"] = fmt.Sprintf("%x", span.ParentSpanID)
	}

	if !span.EndTime.IsZero() {
		data["end_time"] = span.EndTime.Format(time.RFC3339Nano)
		data["duration_ms"] = span.Duration.Milliseconds()
	}

	if span.Error != nil {
		data["error"] = span.Error.Error()
	}

	if len(span.Fields) > 0 {
		data["fields"] = span.Fields
	}

	if len(span.Events) > 0 {
		events := make([]map[string]any, len(span.Events))
		for i, event := range span.Events {
			events[i] = f.eventToMap(event)
		}
		data["events"] = events
	}

	return data
}

func (f *JSONFormatter) eventToMap(event Event) map[string]any {
	data := map[string]any{
		"name":      event.Name,
		"timestamp": event.Timestamp.Format(time.RFC3339Nano),
	}

	if len(event.Fields) > 0 {
		data["fields"] = event.Fields
	}

	return data
}

// CompactFormatter formats trace data in a compact single-line format
type CompactFormatter struct {
	Separator string
}

// NewCompactFormatter creates a new compact formatter
func NewCompactFormatter() *CompactFormatter {
	return &CompactFormatter{
		Separator: " | ",
	}
}

// Format formats a span in compact format
func (f *CompactFormatter) Format(span *SpanData) ([]byte, error) {
	parts := []string{
		span.StartTime.Format("15:04:05.000"),
		span.Status.String(),
		span.Operation,
	}

	if span.Duration > 0 {
		parts = append(parts, span.Duration.String())
	}

	// Add key fields
	if host, ok := span.Fields["ssh.host"]; ok {
		parts = append(parts, fmt.Sprintf("host=%v", host))
	}
	if user, ok := span.Fields["ssh.user"]; ok {
		parts = append(parts, fmt.Sprintf("user=%v", user))
	}

	if span.Error != nil {
		parts = append(parts, fmt.Sprintf("error=%s", span.Error.Error()))
	}

	return []byte(strings.Join(parts, f.Separator) + "\n"), nil
}

// FormatEvent formats an event in compact format
func (f *CompactFormatter) FormatEvent(event Event) ([]byte, error) {
	parts := []string{
		event.Timestamp.Format("15:04:05.000"),
		"EVENT",
		event.Name,
	}

	// Add selected fields
	for k, v := range event.Fields {
		if k == "error" || strings.HasPrefix(k, "ssh.") {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}

	return []byte(strings.Join(parts, f.Separator) + "\n"), nil
}

// MultiFormatter chains multiple formatters
type MultiFormatter struct {
	formatters []Formatter
}

// NewMultiFormatter creates a formatter that outputs to multiple formatters
func NewMultiFormatter(formatters ...Formatter) *MultiFormatter {
	return &MultiFormatter{
		formatters: formatters,
	}
}

// Format formats using all formatters
func (f *MultiFormatter) Format(span *SpanData) ([]byte, error) {
	var buf bytes.Buffer
	for _, formatter := range f.formatters {
		data, err := formatter.Format(span)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	return buf.Bytes(), nil
}

// FormatEvent formats an event using all formatters
func (f *MultiFormatter) FormatEvent(event Event) ([]byte, error) {
	var buf bytes.Buffer
	for _, formatter := range f.formatters {
		data, err := formatter.FormatEvent(event)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	return buf.Bytes(), nil
}

// WriterExporter exports spans to an io.Writer using a formatter
type WriterExporter struct {
	writer    io.Writer
	formatter Formatter
	mu        sync.Mutex
}

// NewWriterExporter creates a new writer exporter
func NewWriterExporter(w io.Writer, formatter Formatter) *WriterExporter {
	if formatter == nil {
		formatter = NewConsoleFormatter()
	}
	return &WriterExporter{
		writer:    w,
		formatter: formatter,
	}
}

// Export exports a span
func (e *WriterExporter) Export(ctx context.Context, span *SpanData) error {
	data, err := e.formatter.Format(span)
	if err != nil {
		return fmt.Errorf("failed to format span: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	_, err = e.writer.Write(data)
	return err
}

// Flush flushes the writer if it supports flushing
func (e *WriterExporter) Flush(ctx context.Context) error {
	if flusher, ok := e.writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// Shutdown closes the writer if it supports closing
func (e *WriterExporter) Shutdown(ctx context.Context) error {
	if closer, ok := e.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
