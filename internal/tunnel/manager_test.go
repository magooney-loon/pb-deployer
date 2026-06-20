package tunnel

import (
	"encoding/base64"
	"strings"
	"testing"
)

// recordingClient is a mock SSHClient that records the commands it is asked to
// run and returns canned successful results.
type recordingClient struct {
	execCmds []string
	sudoCmds []string
}

func (m *recordingClient) Connect() error            { return nil }
func (m *recordingClient) Close() error              { return nil }
func (m *recordingClient) IsConnected() bool         { return true }
func (m *recordingClient) Ping() error               { return nil }
func (m *recordingClient) HostInfo() (string, error) { return "test", nil }
func (m *recordingClient) SetTracer(Tracer)          {}

func (m *recordingClient) Execute(cmd string, opts ...ExecOption) (*Result, error) {
	m.execCmds = append(m.execCmds, cmd)
	if strings.Contains(cmd, "getent passwd") {
		return &Result{Stdout: "/home/deploy\n", ExitCode: 0}, nil
	}
	return &Result{ExitCode: 0}, nil
}

func (m *recordingClient) ExecuteSudo(cmd string, opts ...ExecOption) (*Result, error) {
	m.sudoCmds = append(m.sudoCmds, cmd)
	return &Result{ExitCode: 0}, nil
}

func (m *recordingClient) Upload(localPath, remotePath string, opts ...FileOption) error {
	return nil
}

func (m *recordingClient) Download(remotePath, localPath string, opts ...FileOption) error {
	return nil
}

// TestSetupSSHKeysIsInjectionSafe verifies the authorized_keys content is passed
// via base64 (decoded remotely) rather than interpolated into echo '...', so a
// key containing a single quote can't break out of the shell command.
func TestSetupSSHKeysIsInjectionSafe(t *testing.T) {
	mock := &recordingClient{}
	m := NewManager(mock)

	keys := []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 user@host",
		"ssh-rsa AAAAB3NzaC1yc2E injected'; rm -rf / #",
	}

	if err := m.SetupSSHKeys("deploy", keys); err != nil {
		t.Fatalf("SetupSSHKeys returned error: %v", err)
	}

	var writeCmd string
	for _, c := range mock.sudoCmds {
		if strings.Contains(c, "authorized_keys") && strings.Contains(c, "base64 -d") {
			writeCmd = c
			break
		}
	}
	if writeCmd == "" {
		t.Fatalf("expected a base64-decoded authorized_keys write; sudo cmds: %v", mock.sudoCmds)
	}

	// The raw, attacker-influenced key content must not appear literally.
	if strings.Contains(writeCmd, "rm -rf /") {
		t.Errorf("raw key content leaked into the shell command: %q", writeCmd)
	}

	// The base64 blob must round-trip back to exactly the keys (newline-joined).
	fields := strings.Fields(writeCmd)
	if len(fields) < 2 || fields[0] != "echo" {
		t.Fatalf("unexpected write command shape: %q", writeCmd)
	}
	decoded, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		t.Fatalf("authorized_keys payload is not valid base64: %v", err)
	}
	want := strings.Join(keys, "\n") + "\n"
	if string(decoded) != want {
		t.Errorf("decoded authorized_keys = %q, want %q", string(decoded), want)
	}
}
