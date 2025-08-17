package tunnel

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// MockClient is a mock SSH client for testing
type MockClient struct {
	mu          sync.Mutex
	connected   bool
	responses   map[string]*Result
	errors      map[string]error
	uploads     map[string]string
	downloads   map[string]string
	execHistory []string
	tracer      Tracer
}

// NewMockClient creates a new mock client
func NewMockClient() *MockClient {
	return &MockClient{
		responses:   make(map[string]*Result),
		errors:      make(map[string]error),
		uploads:     make(map[string]string),
		downloads:   make(map[string]string),
		execHistory: make([]string, 0),
		tracer:      &NoOpTracer{},
	}
}

// SetTracer sets the tracer for the mock client
func (m *MockClient) SetTracer(tracer Tracer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tracer != nil {
		m.tracer = tracer
	} else {
		m.tracer = &NoOpTracer{}
	}
}

// Connect simulates connection
func (m *MockClient) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tracer.OnConnect("mock-host", "mock-user")

	if err, exists := m.errors["connect"]; exists {
		m.tracer.OnError("connect", err)
		return err
	}

	m.connected = true
	return nil
}

// Close simulates closing connection
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tracer.OnDisconnect("mock-host")
	m.connected = false
	return nil
}

// IsConnected returns connection status
func (m *MockClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// Execute simulates command execution
func (m *MockClient) Execute(cmd string, opts ...ExecOption) (*Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply options
	cfg := &execConfig{
		timeout: 60 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Record execution
	m.execHistory = append(m.execHistory, cmd)
	m.tracer.OnExecute(cmd)

	// Check for errors
	if err, exists := m.errors[cmd]; exists {
		m.tracer.OnExecuteResult(cmd, nil, err)
		return nil, err
	}

	// Check for mock response
	if result, exists := m.responses[cmd]; exists {
		m.tracer.OnExecuteResult(cmd, result, nil)
		return result, nil
	}

	// Check for pattern match
	for pattern, result := range m.responses {
		if strings.Contains(cmd, pattern) {
			m.tracer.OnExecuteResult(cmd, result, nil)
			return result, nil
		}
	}

	// Default response
	defaultResult := &Result{
		Stdout:   fmt.Sprintf("Mock output for: %s", cmd),
		Stderr:   "",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}
	m.tracer.OnExecuteResult(cmd, defaultResult, nil)
	return defaultResult, nil
}

// ExecuteSudo simulates sudo command execution
func (m *MockClient) ExecuteSudo(cmd string, opts ...ExecOption) (*Result, error) {
	return m.Execute("sudo "+cmd, opts...)
}

// Upload simulates file upload
func (m *MockClient) Upload(localPath, remotePath string, opts ...FileOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tracer.OnUpload(localPath, remotePath)

	// Check for errors
	uploadKey := fmt.Sprintf("%s->%s", localPath, remotePath)
	if err, exists := m.errors[uploadKey]; exists {
		m.tracer.OnUploadComplete(localPath, remotePath, err)
		return err
	}

	// Record upload
	m.uploads[localPath] = remotePath
	m.tracer.OnUploadComplete(localPath, remotePath, nil)
	return nil
}

// Download simulates file download
func (m *MockClient) Download(remotePath, localPath string, opts ...FileOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tracer.OnDownload(remotePath, localPath)

	// Check for errors
	downloadKey := fmt.Sprintf("%s->%s", remotePath, localPath)
	if err, exists := m.errors[downloadKey]; exists {
		m.tracer.OnDownloadComplete(remotePath, localPath, err)
		return err
	}

	// Record download
	m.downloads[remotePath] = localPath
	m.tracer.OnDownloadComplete(remotePath, localPath, nil)
	return nil
}

// Ping simulates ping
func (m *MockClient) Ping() error {
	result, err := m.Execute("echo ping", WithTimeout(5*time.Second))
	if err != nil {
		return err
	}
	if !strings.Contains(result.Stdout, "ping") {
		return &Error{
			Type:    ErrorConnection,
			Message: "ping failed",
		}
	}
	return nil
}

// HostInfo simulates getting host info
func (m *MockClient) HostInfo() (string, error) {
	result, err := m.Execute("uname -a", WithTimeout(10*time.Second))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// Mock setup methods

// OnExecute sets up a mock response for a command
func (m *MockClient) OnExecute(cmd string) *MockResponse {
	return &MockResponse{
		client: m,
		cmd:    cmd,
	}
}

// OnConnect sets up mock behavior for connection
func (m *MockClient) OnConnect() *MockConnectionResponse {
	return &MockConnectionResponse{
		client: m,
	}
}

// OnUpload sets up mock behavior for upload
func (m *MockClient) OnUpload(local, remote string) *MockFileResponse {
	return &MockFileResponse{
		client: m,
		key:    fmt.Sprintf("%s->%s", local, remote),
	}
}

// OnDownload sets up mock behavior for download
func (m *MockClient) OnDownload(remote, local string) *MockFileResponse {
	return &MockFileResponse{
		client: m,
		key:    fmt.Sprintf("%s->%s", remote, local),
	}
}

// GetExecutionHistory returns the history of executed commands
func (m *MockClient) GetExecutionHistory() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	history := make([]string, len(m.execHistory))
	copy(history, m.execHistory)
	return history
}

// GetUploads returns all recorded uploads
func (m *MockClient) GetUploads() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	uploads := make(map[string]string)
	for k, v := range m.uploads {
		uploads[k] = v
	}
	return uploads
}

// GetDownloads returns all recorded downloads
func (m *MockClient) GetDownloads() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	downloads := make(map[string]string)
	for k, v := range m.downloads {
		downloads[k] = v
	}
	return downloads
}

// Reset clears all mock data
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connected = false
	m.responses = make(map[string]*Result)
	m.errors = make(map[string]error)
	m.uploads = make(map[string]string)
	m.downloads = make(map[string]string)
	m.execHistory = make([]string, 0)
}

// MockResponse helps build mock responses
type MockResponse struct {
	client *MockClient
	cmd    string
}

// Return sets the return value for the command
func (mr *MockResponse) Return(result *Result, err error) {
	mr.client.mu.Lock()
	defer mr.client.mu.Unlock()

	if err != nil {
		mr.client.errors[mr.cmd] = err
	} else if result != nil {
		mr.client.responses[mr.cmd] = result
	}
}

// ReturnSuccess sets a successful response
func (mr *MockResponse) ReturnSuccess(stdout string) {
	mr.Return(&Result{
		Stdout:   stdout,
		Stderr:   "",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}, nil)
}

// ReturnError sets an error response
func (mr *MockResponse) ReturnError(err error) {
	mr.Return(nil, err)
}

// ReturnFailure sets a command failure response
func (mr *MockResponse) ReturnFailure(stderr string, exitCode int) {
	mr.Return(&Result{
		Stdout:   "",
		Stderr:   stderr,
		ExitCode: exitCode,
		Duration: 100 * time.Millisecond,
	}, nil)
}

// MockConnectionResponse helps build connection responses
type MockConnectionResponse struct {
	client *MockClient
}

// Fail makes the connection fail
func (mcr *MockConnectionResponse) Fail(err error) {
	mcr.client.mu.Lock()
	defer mcr.client.mu.Unlock()
	mcr.client.errors["connect"] = err
}

// Succeed makes the connection succeed
func (mcr *MockConnectionResponse) Succeed() {
	mcr.client.mu.Lock()
	defer mcr.client.mu.Unlock()
	delete(mcr.client.errors, "connect")
}

// MockFileResponse helps build file operation responses
type MockFileResponse struct {
	client *MockClient
	key    string
}

// Fail makes the file operation fail
func (mfr *MockFileResponse) Fail(err error) {
	mfr.client.mu.Lock()
	defer mfr.client.mu.Unlock()
	mfr.client.errors[mfr.key] = err
}

// Succeed makes the file operation succeed
func (mfr *MockFileResponse) Succeed() {
	mfr.client.mu.Lock()
	defer mfr.client.mu.Unlock()
	delete(mfr.client.errors, mfr.key)
}

// NewMockManager creates a manager with a mock client
func NewMockManager() (*Manager, *MockClient) {
	client := NewMockClient()
	manager := NewManager(client)
	return manager, client
}

// TestTracer is a tracer that records all events for testing
type TestTracer struct {
	mu     sync.Mutex
	events []TracerEvent
}

// TracerEvent represents a traced event
type TracerEvent struct {
	Type      string
	Timestamp time.Time
	Data      map[string]interface{}
}

// NewTestTracer creates a new test tracer
func NewTestTracer() *TestTracer {
	return &TestTracer{
		events: make([]TracerEvent, 0),
	}
}

func (t *TestTracer) recordEvent(eventType string, data map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.events = append(t.events, TracerEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	})
}

func (t *TestTracer) OnConnect(host string, user string) {
	t.recordEvent("connect", map[string]interface{}{
		"host": host,
		"user": user,
	})
}

func (t *TestTracer) OnDisconnect(host string) {
	t.recordEvent("disconnect", map[string]interface{}{
		"host": host,
	})
}

func (t *TestTracer) OnExecute(cmd string) {
	t.recordEvent("execute", map[string]interface{}{
		"cmd": cmd,
	})
}

func (t *TestTracer) OnExecuteResult(cmd string, result *Result, err error) {
	data := map[string]interface{}{
		"cmd": cmd,
	}
	if result != nil {
		data["exit_code"] = result.ExitCode
		data["duration"] = result.Duration
	}
	if err != nil {
		data["error"] = err.Error()
	}
	t.recordEvent("execute_result", data)
}

func (t *TestTracer) OnUpload(local, remote string) {
	t.recordEvent("upload", map[string]interface{}{
		"local":  local,
		"remote": remote,
	})
}

func (t *TestTracer) OnUploadComplete(local, remote string, err error) {
	data := map[string]interface{}{
		"local":  local,
		"remote": remote,
	}
	if err != nil {
		data["error"] = err.Error()
	}
	t.recordEvent("upload_complete", data)
}

func (t *TestTracer) OnDownload(remote, local string) {
	t.recordEvent("download", map[string]interface{}{
		"remote": remote,
		"local":  local,
	})
}

func (t *TestTracer) OnDownloadComplete(remote, local string, err error) {
	data := map[string]interface{}{
		"remote": remote,
		"local":  local,
	}
	if err != nil {
		data["error"] = err.Error()
	}
	t.recordEvent("download_complete", data)
}

func (t *TestTracer) OnError(operation string, err error) {
	t.recordEvent("error", map[string]interface{}{
		"operation": operation,
		"error":     err.Error(),
	})
}

// GetEvents returns all recorded events
func (t *TestTracer) GetEvents() []TracerEvent {
	t.mu.Lock()
	defer t.mu.Unlock()

	events := make([]TracerEvent, len(t.events))
	copy(events, t.events)
	return events
}

// GetEventsByType returns events of a specific type
func (t *TestTracer) GetEventsByType(eventType string) []TracerEvent {
	t.mu.Lock()
	defer t.mu.Unlock()

	filtered := make([]TracerEvent, 0)
	for _, event := range t.events {
		if event.Type == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// Clear clears all recorded events
func (t *TestTracer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = make([]TracerEvent, 0)
}
