package logger

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ConsoleLogger provides enhanced console output with SSH-specific formatting
type ConsoleLogger struct {
	*Logger
	width       int
	interactive bool
}

// NewConsoleLoggerInstance creates a logger optimized for console interaction
func NewConsoleLoggerInstance() *ConsoleLogger {
	logger := NewBasicConsoleLogger(InfoLevel)
	logger.EnableColor(true)
	logger.SetPrefix("pb-deployer")

	return &ConsoleLogger{
		Logger:      logger,
		width:       80,
		interactive: isTerminal(os.Stdout),
	}
}

// SetWidth sets the console width for formatting
func (c *ConsoleLogger) SetWidth(width int) {
	c.width = width
}

// Banner prints a styled banner message
func (c *ConsoleLogger) Banner(msg string) {
	if !c.interactive {
		c.Info("=== %s ===", msg)
		return
	}

	fmt.Fprintf(c.output, "\n")
	c.printLine("=")
	fmt.Fprintf(c.output, "%s%s%s%s%s\n",
		ColorBold, ColorBlue, center(msg, c.width-4), ColorReset, ColorReset)
	c.printLine("=")
	fmt.Fprintf(c.output, "\n")
}

// Section prints a section header
func (c *ConsoleLogger) Section(title string) {
	if !c.interactive {
		c.Info("--- %s ---", title)
		return
	}

	fmt.Fprintf(c.output, "\n%s%s%s%s\n", ColorBold, ColorCyan, title, ColorReset)
	c.printLine("-")
}

// SSHConnect logs SSH connection attempts with status
func (c *ConsoleLogger) SSHConnect(host string, port int, username string, status string) {
	address := fmt.Sprintf("%s:%d", host, port)

	var statusColor, statusSymbol string
	switch status {
	case "connecting":
		statusColor, statusSymbol = ColorYellow, "⟳"
	case "connected":
		statusColor, statusSymbol = ColorGreen, "✓"
	case "failed":
		statusColor, statusSymbol = ColorRed, "✗"
	case "timeout":
		statusColor, statusSymbol = ColorYellow, "⏱"
	default:
		statusColor, statusSymbol = ColorGray, "•"
	}

	if c.enableColor {
		fmt.Fprintf(c.output, "%s%s%s SSH %s@%s %s\n",
			statusColor, statusSymbol, ColorReset, username, address, status)
	} else {
		fmt.Fprintf(c.output, "[%s] SSH %s@%s %s\n", statusSymbol, username, address, status)
	}
}

// SSHCommand logs SSH command execution
func (c *ConsoleLogger) SSHCommand(host string, username string, command string, success bool) {
	symbol := "✓"
	color := ColorGreen
	if !success {
		symbol = "✗"
		color = ColorRed
	}

	// Truncate long commands for display
	displayCmd := command
	if len(displayCmd) > 60 {
		displayCmd = displayCmd[:57] + "..."
	}

	if c.enableColor {
		fmt.Fprintf(c.output, "%s%s%s %s@%s: %s\n",
			color, symbol, ColorReset, username, host, displayCmd)
	} else {
		fmt.Fprintf(c.output, "[%s] %s@%s: %s\n", symbol, username, host, displayCmd)
	}
}

// SSHProgress logs SSH operation progress with visual progress bar
func (c *ConsoleLogger) SSHProgress(operation string, current, total int, message string) {
	if !c.interactive {
		percentage := float64(current) / float64(total) * 100
		c.Info("Progress [%.1f%%] %s: %s", percentage, operation, message)
		return
	}

	percentage := float64(current) / float64(total) * 100
	barWidth := 30
	filledWidth := int(float64(barWidth) * percentage / 100)

	// Create progress bar
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

	fmt.Fprintf(c.output, "\r%s%s%s [%s] %.1f%% %s: %s%s",
		ColorBold, ColorBlue, operation, bar, percentage, ColorReset, message,
		strings.Repeat(" ", 10)) // Clear any remaining text
}

// SSHStep logs individual steps in SSH operations
func (c *ConsoleLogger) SSHStep(step string, status string, message string, details string) {
	var statusColor, statusSymbol string
	switch status {
	case "running":
		statusColor, statusSymbol = ColorYellow, "⟳"
	case "success":
		statusColor, statusSymbol = ColorGreen, "✓"
	case "failed":
		statusColor, statusSymbol = ColorRed, "✗"
	case "warning":
		statusColor, statusSymbol = ColorYellow, "⚠"
	default:
		statusColor, statusSymbol = ColorGray, "•"
	}

	if c.enableColor {
		fmt.Fprintf(c.output, "%s%s%s %s: %s\n",
			statusColor, statusSymbol, ColorReset, step, message)
	} else {
		fmt.Fprintf(c.output, "[%s] %s: %s\n", statusSymbol, step, message)
	}

	if details != "" {
		c.printIndented(details, 4)
	}
}

// ServerStatus prints server status with visual indicators
func (c *ConsoleLogger) ServerStatus(serverName string, host string, setupComplete bool, securityLocked bool, connected bool) {
	if !c.interactive {
		c.Info("Server %s (%s): setup=%v security=%v connected=%v",
			serverName, host, setupComplete, securityLocked, connected)
		return
	}

	fmt.Fprintf(c.output, "\n%s%s%s %s (%s)%s\n",
		ColorBold, ColorBlue, "Server:", serverName, host, ColorReset)

	// Setup status
	setupSymbol, setupColor := "✗", ColorRed
	if setupComplete {
		setupSymbol, setupColor = "✓", ColorGreen
	}

	// Security status
	securitySymbol, securityColor := "✗", ColorRed
	if securityLocked {
		securitySymbol, securityColor = "🔒", ColorGreen
	}

	// Connection status
	connSymbol, connColor := "✗", ColorRed
	if connected {
		connSymbol, connColor = "✓", ColorGreen
	}

	fmt.Fprintf(c.output, "  %s%s%s Setup Complete\n", setupColor, setupSymbol, ColorReset)
	fmt.Fprintf(c.output, "  %s%s%s Security Locked\n", securityColor, securitySymbol, ColorReset)
	fmt.Fprintf(c.output, "  %s%s%s Connected\n", connColor, connSymbol, ColorReset)
}

// AppStatus prints application status with service information
func (c *ConsoleLogger) AppStatus(appName string, domain string, status string, version string, serviceName string) {
	if !c.interactive {
		c.Info("App %s: status=%s version=%s domain=%s service=%s",
			appName, status, version, domain, serviceName)
		return
	}

	fmt.Fprintf(c.output, "\n%s%sApp:%s %s%s\n",
		ColorBold, ColorPurple, ColorReset, appName, ColorReset)

	// Status with appropriate color
	var statusColor string
	switch status {
	case "online":
		statusColor = ColorGreen
	case "offline":
		statusColor = ColorRed
	default:
		statusColor = ColorYellow
	}

	fmt.Fprintf(c.output, "  Status: %s%s%s\n", statusColor, status, ColorReset)
	if version != "" {
		fmt.Fprintf(c.output, "  Version: %s\n", version)
	}
	if domain != "" {
		fmt.Fprintf(c.output, "  Domain: %s%s%s\n", ColorCyan, domain, ColorReset)
	}
	if serviceName != "" {
		fmt.Fprintf(c.output, "  Service: %s\n", serviceName)
	}
}

// DeploymentStatus prints deployment status with progress
func (c *ConsoleLogger) DeploymentStatus(deploymentID string, appName string, version string, status string, progress int) {
	if !c.interactive {
		c.Info("Deployment %s: app=%s version=%s status=%s progress=%d%%",
			deploymentID, appName, version, status, progress)
		return
	}

	var statusColor, statusSymbol string
	switch status {
	case "pending":
		statusColor, statusSymbol = ColorYellow, "⏳"
	case "running":
		statusColor, statusSymbol = ColorBlue, "🚀"
	case "success":
		statusColor, statusSymbol = ColorGreen, "✅"
	case "failed":
		statusColor, statusSymbol = ColorRed, "❌"
	default:
		statusColor, statusSymbol = ColorGray, "•"
	}

	fmt.Fprintf(c.output, "%s%s Deployment:%s %s → %s %s(%s)%s\n",
		ColorBold, statusSymbol, ColorReset, appName, version, statusColor, status, ColorReset)

	if status == "running" && progress > 0 {
		c.ProgressBar(progress, 100, 40)
	}
}

// ProgressBar displays a visual progress bar
func (c *ConsoleLogger) ProgressBar(current, total, width int) {
	if !c.interactive {
		percentage := float64(current) / float64(total) * 100
		c.Info("Progress: %.1f%% (%d/%d)", percentage, current, total)
		return
	}

	percentage := float64(current) / float64(total) * 100
	filledWidth := int(float64(width) * percentage / 100)

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", width-filledWidth)

	fmt.Fprintf(c.output, "  [%s%s%s] %.1f%%\n",
		ColorGreen, bar, ColorReset, percentage)
}

// CommandOutput formats and displays command output
func (c *ConsoleLogger) CommandOutput(output string, maxLines int) {
	if output == "" {
		return
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > maxLines {
		// Show first few and last few lines
		showFirst := maxLines / 2
		showLast := maxLines - showFirst - 1

		for i := 0; i < showFirst; i++ {
			c.printIndented(lines[i], 2)
		}

		c.printIndented(fmt.Sprintf("... (%d lines omitted) ...", len(lines)-maxLines+1), 2)

		for i := len(lines) - showLast; i < len(lines); i++ {
			c.printIndented(lines[i], 2)
		}
	} else {
		for _, line := range lines {
			c.printIndented(line, 2)
		}
	}
}

// ErrorBox prints an error in a formatted box
func (c *ConsoleLogger) ErrorBox(title string, message string, suggestions []string) {
	if !c.interactive {
		c.Error("%s: %s", title, message)
		for _, suggestion := range suggestions {
			c.Info("Suggestion: %s", suggestion)
		}
		return
	}

	boxWidth := c.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	fmt.Fprintf(c.output, "\n%s", ColorRed)
	c.printBoxLine("┌", "─", "┐", boxWidth)
	c.printBoxContent(fmt.Sprintf("ERROR: %s", title), boxWidth)
	c.printBoxLine("├", "─", "┤", boxWidth)
	c.printBoxContent(message, boxWidth)

	if len(suggestions) > 0 {
		c.printBoxLine("├", "─", "┤", boxWidth)
		c.printBoxContent("Suggestions:", boxWidth)
		for _, suggestion := range suggestions {
			c.printBoxContent(fmt.Sprintf("• %s", suggestion), boxWidth)
		}
	}

	c.printBoxLine("└", "─", "┘", boxWidth)
	fmt.Fprintf(c.output, "%s\n", ColorReset)
}

// SuccessBox prints a success message in a formatted box
func (c *ConsoleLogger) SuccessBox(title string, message string) {
	if !c.interactive {
		c.Info("SUCCESS: %s - %s", title, message)
		return
	}

	boxWidth := c.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	fmt.Fprintf(c.output, "\n%s", ColorGreen)
	c.printBoxLine("┌", "─", "┐", boxWidth)
	c.printBoxContent(fmt.Sprintf("✓ SUCCESS: %s", title), boxWidth)
	c.printBoxLine("├", "─", "┤", boxWidth)
	c.printBoxContent(message, boxWidth)
	c.printBoxLine("└", "─", "┘", boxWidth)
	fmt.Fprintf(c.output, "%s\n", ColorReset)
}

// Table prints data in a formatted table
func (c *ConsoleLogger) Table(headers []string, rows [][]string) {
	if !c.interactive || len(headers) == 0 {
		// Fallback to simple text output
		for _, row := range rows {
			c.Info(strings.Join(row, " | "))
		}
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Print table header
	fmt.Fprintf(c.output, "\n%s", ColorBold)
	c.printTableRow(headers, colWidths)
	fmt.Fprintf(c.output, "%s", ColorReset)

	// Print separator
	separator := make([]string, len(headers))
	for i, width := range colWidths {
		separator[i] = strings.Repeat("─", width)
	}
	c.printTableRow(separator, colWidths)

	// Print data rows
	for _, row := range rows {
		c.printTableRow(row, colWidths)
	}
	fmt.Fprintf(c.output, "\n")
}

// SSH-specific logging methods

// SSHOperation logs the start of an SSH operation
func (c *ConsoleLogger) SSHOperation(operation string, server string, username string) func() {
	start := time.Now()

	if c.enableColor {
		fmt.Fprintf(c.output, "%s⟳%s Starting %s on %s@%s...\n",
			ColorYellow, ColorReset, operation, username, server)
	} else {
		fmt.Fprintf(c.output, "[⟳] Starting %s on %s@%s...\n", operation, username, server)
	}

	// Return a completion function
	return func() {
		duration := time.Since(start)
		if c.enableColor {
			fmt.Fprintf(c.output, "%s✓%s Completed %s in %s\n",
				ColorGreen, ColorReset, operation, duration.Round(time.Millisecond))
		} else {
			fmt.Fprintf(c.output, "[✓] Completed %s in %s\n", operation, duration.Round(time.Millisecond))
		}
	}
}

// SSHError logs SSH-specific errors with context
func (c *ConsoleLogger) SSHError(operation string, server string, username string, err error) {
	if c.enableColor {
		fmt.Fprintf(c.output, "%s✗%s %s failed on %s@%s: %s%s%s\n",
			ColorRed, ColorReset, operation, username, server, ColorRed, err.Error(), ColorReset)
	} else {
		fmt.Fprintf(c.output, "[✗] %s failed on %s@%s: %s\n", operation, username, server, err.Error())
	}
}

// ConnectionTest displays connection test results in a formatted way
func (c *ConsoleLogger) ConnectionTest(serverName string, host string, port int, results map[string]interface{}) {
	c.Section(fmt.Sprintf("Connection Test: %s (%s:%d)", serverName, host, port))

	if tcp, ok := results["tcp_connection"].(map[string]interface{}); ok {
		if success, ok := tcp["success"].(bool); ok {
			symbol := "✓"
			color := ColorGreen
			if !success {
				symbol = "✗"
				color = ColorRed
			}

			latency := ""
			if lat, ok := tcp["latency"].(string); ok {
				latency = fmt.Sprintf(" (%s)", lat)
			}

			if c.enableColor {
				fmt.Fprintf(c.output, "  %s%s%s TCP Connection%s\n", color, symbol, ColorReset, latency)
			} else {
				fmt.Fprintf(c.output, "  [%s] TCP Connection%s\n", symbol, latency)
			}
		}
	}

	// Display SSH connection results
	sshConnections := []struct {
		key   string
		label string
	}{
		{"root_ssh_connection", "Root SSH"},
		{"app_ssh_connection", "App SSH"},
	}

	for _, conn := range sshConnections {
		if ssh, ok := results[conn.key].(map[string]interface{}); ok {
			if success, ok := ssh["success"].(bool); ok {
				symbol := "✓"
				color := ColorGreen
				if !success {
					symbol = "✗"
					color = ColorRed
				}

				username := ""
				if user, ok := ssh["username"].(string); ok {
					username = fmt.Sprintf(" (%s)", user)
				}

				authMethod := ""
				if auth, ok := ssh["auth_method"].(string); ok {
					authMethod = fmt.Sprintf(" [%s]", auth)
				}

				if c.enableColor {
					fmt.Fprintf(c.output, "  %s%s%s %s%s%s\n", color, symbol, ColorReset, conn.label, username, authMethod)
				} else {
					fmt.Fprintf(c.output, "  [%s] %s%s%s\n", symbol, conn.label, username, authMethod)
				}

				// Show error if present
				if !success {
					if errMsg, ok := ssh["error"].(string); ok {
						c.printIndented(fmt.Sprintf("Error: %s", errMsg), 6)
					}
				}
			}
		}
	}

	// Overall status
	if status, ok := results["overall_status"].(string); ok {
		var statusColor string
		switch status {
		case "healthy", "healthy_secured":
			statusColor = ColorGreen
		case "unreachable":
			statusColor = ColorRed
		default:
			statusColor = ColorYellow
		}

		fmt.Fprintf(c.output, "\n%sOverall Status:%s %s%s%s\n",
			ColorBold, ColorReset, statusColor, status, ColorReset)
	}
}

// Spinner provides a simple text-based spinner for operations
type Spinner struct {
	logger  *ConsoleLogger
	message string
	frames  []string
	stop    chan struct{}
	done    chan struct{}
}

// NewSpinner creates a new spinner with the given message
func (c *ConsoleLogger) NewSpinner(message string) *Spinner {
	if !c.interactive {
		c.Info("%s...", message)
		return &Spinner{logger: c, message: message}
	}

	return &Spinner{
		logger:  c,
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	if !s.logger.interactive {
		return
	}

	go func() {
		defer close(s.done)

		frame := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				fmt.Fprintf(s.logger.output, "\r%s%s%s %s",
					ColorYellow, s.frames[frame], ColorReset, s.message)
				frame = (frame + 1) % len(s.frames)
			}
		}
	}()
}

// Stop stops the spinner and clears the line
func (s *Spinner) Stop() {
	if !s.logger.interactive {
		return
	}

	close(s.stop)
	<-s.done

	// Clear the spinner line
	fmt.Fprintf(s.logger.output, "\r%s\r", strings.Repeat(" ", len(s.message)+4))
}

// StopWithMessage stops the spinner and displays a final message
func (s *Spinner) StopWithMessage(success bool, message string) {
	if !s.logger.interactive {
		if success {
			s.logger.Info("✓ %s", message)
		} else {
			s.logger.Error("✗ %s", message)
		}
		return
	}

	s.Stop()

	symbol := "✓"
	color := ColorGreen
	if !success {
		symbol = "✗"
		color = ColorRed
	}

	fmt.Fprintf(s.logger.output, "%s%s%s %s\n", color, symbol, ColorReset, message)
}

// Helper methods

// printLine prints a line of characters across the console width
func (c *ConsoleLogger) printLine(char string) {
	fmt.Fprintf(c.output, "%s\n", strings.Repeat(char, c.width))
}

// printIndented prints text with indentation
func (c *ConsoleLogger) printIndented(text string, indent int) {
	spaces := strings.Repeat(" ", indent)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		fmt.Fprintf(c.output, "%s%s\n", spaces, line)
	}
}

// printBoxLine prints a box border line
func (c *ConsoleLogger) printBoxLine(left, middle, right string, width int) {
	fmt.Fprintf(c.output, "%s%s%s\n", left, strings.Repeat(middle, width-2), right)
}

// printBoxContent prints content inside a box with proper padding
func (c *ConsoleLogger) printBoxContent(content string, width int) {
	padding := width - 4 - len(content)
	if padding < 0 {
		// Content is too long, truncate it
		content = content[:width-7] + "..."
		padding = 0
	}

	fmt.Fprintf(c.output, "│ %s%s │\n", content, strings.Repeat(" ", padding))
}

// printTableRow prints a table row with proper column alignment
func (c *ConsoleLogger) printTableRow(cells []string, widths []int) {
	for i, cell := range cells {
		if i < len(widths) {
			fmt.Fprintf(c.output, "%-*s", widths[i], cell)
			if i < len(cells)-1 {
				fmt.Fprintf(c.output, " │ ")
			}
		}
	}
	fmt.Fprintf(c.output, "\n")
}

// center centers text within a given width
func center(text string, width int) string {
	if len(text) >= width {
		return text
	}

	padding := width - len(text)
	leftPad := padding / 2
	rightPad := padding - leftPad

	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}

// Global console logger instance
var Console = NewConsoleLoggerInstance()

// Package-level console functions

// Banner prints a banner using the global console logger
func Banner(msg string) {
	Console.Banner(msg)
}

// Section prints a section header using the global console logger
func Section(title string) {
	Console.Section(title)
}

// SSHConnect logs SSH connection status using the global console logger
func SSHConnect(host string, port int, username string, status string) {
	Console.SSHConnect(host, port, username, status)
}

// SSHCommand logs SSH command execution using the global console logger
func SSHCommand(host string, username string, command string, success bool) {
	Console.SSHCommand(host, username, command, success)
}

// SSHProgress logs SSH progress using the global console logger
func SSHProgress(operation string, current, total int, message string) {
	Console.SSHProgress(operation, current, total, message)
}

// SSHStep logs SSH steps using the global console logger
func SSHStep(step string, status string, message string, details string) {
	Console.SSHStep(step, status, message, details)
}

// SSHOperation starts an SSH operation timer using the global console logger
func SSHOperation(operation string, server string, username string) func() {
	return Console.SSHOperation(operation, server, username)
}

// SSHError logs SSH errors using the global console logger
func SSHError(operation string, server string, username string, err error) {
	Console.SSHError(operation, server, username, err)
}

// ServerStatus displays server status using the global console logger
func ServerStatus(serverName string, host string, setupComplete bool, securityLocked bool, connected bool) {
	Console.ServerStatus(serverName, host, setupComplete, securityLocked, connected)
}

// AppStatus displays app status using the global console logger
func AppStatus(appName string, domain string, status string, version string, serviceName string) {
	Console.AppStatus(appName, domain, status, version, serviceName)
}

// DeploymentStatus displays deployment status using the global console logger
func DeploymentStatus(deploymentID string, appName string, version string, status string, progress int) {
	Console.DeploymentStatus(deploymentID, appName, version, status, progress)
}

// ErrorBox displays an error box using the global console logger
func ErrorBox(title string, message string, suggestions []string) {
	Console.ErrorBox(title, message, suggestions)
}

// SuccessBox displays a success box using the global console logger
func SuccessBox(title string, message string) {
	Console.SuccessBox(title, message)
}

// Table displays a table using the global console logger
func Table(headers []string, rows [][]string) {
	Console.Table(headers, rows)
}

// NewSpinner creates a new spinner using the global console logger
func NewSpinner(message string) *Spinner {
	return Console.NewSpinner(message)
}

// ProgressBar displays a progress bar using the global console logger
func ProgressBar(current, total, width int) {
	Console.ProgressBar(current, total, width)
}

// CommandOutput displays command output using the global console logger
func CommandOutput(output string, maxLines int) {
	Console.CommandOutput(output, maxLines)
}
