package tunnel

import (
	"strings"
	"testing"

	"pb-deployer/internal/logger"
)

// TestExecuteNotConnectedReturnsNonNilResult guards the fix for the systemic
// nil-deref: Execute must never return a nil *Result, even on error, so callers
// that read result.Stderr/ExitCode in an error branch can't panic.
func TestExecuteNotConnectedReturnsNonNilResult(t *testing.T) {
	c := &Client{tracer: &NoOpTracer{}, logger: logger.GetTunnelLogger()}

	result, err := c.Execute("echo hi")
	if err == nil {
		t.Fatal("expected error when not connected")
	}
	if result == nil {
		t.Fatal("Execute returned a nil result on error — callers can nil-deref")
	}
	if result.ExitCode != -1 {
		t.Errorf("expected sentinel ExitCode -1, got %d", result.ExitCode)
	}
	// Accessing fields in the error branch must be safe.
	_ = result.Stderr
}

func TestGetConnNilWhenClosed(t *testing.T) {
	c := &Client{}
	if c.getConn() == nil {
		// zero client has no conn — fine, but make the closed path explicit too
	}

	c.closed = true
	if c.getConn() != nil {
		t.Error("getConn must return nil once the client is closed")
	}
}

func TestEnvEnabled(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"Yes", true},
		{"on", true},
		{"0", false},
		{"false", false},
		{"", false},
		{"nope", false},
		{"  true  ", true},
	}

	const key = "PB_DEPLOYER_TEST_ENV"
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			t.Setenv(key, tt.val)
			if got := envEnabled(key); got != tt.want {
				t.Errorf("envEnabled(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestSSHDebugAndInsecureGating(t *testing.T) {
	// Default (unset) must be secure: debug off, insecure off.
	t.Setenv("PB_DEPLOYER_SSH_DEBUG", "")
	t.Setenv("PB_DEPLOYER_INSECURE_SSH", "")
	if sshDebugEnabled() {
		t.Error("sshDebugEnabled should default to false")
	}
	if insecureSSHEnabled() {
		t.Error("insecureSSHEnabled should default to false")
	}
	if DevelopmentAuthConfig().DebugAuth {
		t.Error("DevelopmentAuthConfig.DebugAuth should be false unless PB_DEPLOYER_SSH_DEBUG is set")
	}

	t.Setenv("PB_DEPLOYER_SSH_DEBUG", "1")
	t.Setenv("PB_DEPLOYER_INSECURE_SSH", "yes")
	if !sshDebugEnabled() {
		t.Error("sshDebugEnabled should be true when PB_DEPLOYER_SSH_DEBUG=1")
	}
	if !insecureSSHEnabled() {
		t.Error("insecureSSHEnabled should be true when PB_DEPLOYER_INSECURE_SSH=yes")
	}
	if !DevelopmentAuthConfig().DebugAuth {
		t.Error("DevelopmentAuthConfig.DebugAuth should follow PB_DEPLOYER_SSH_DEBUG")
	}
}

func TestShellescape(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "'plain'"},
		{"", "''"},
		{"two words", "'two words'"},
		{"it's", `'it'\''s'`},
		{"a'b'c", `'a'\''b'\''c'`},
		{"$(rm -rf /)", "'$(rm -rf /)'"},
	}

	for _, tt := range tests {
		if got := shellescape(tt.in); got != tt.want {
			t.Errorf("shellescape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestBuildCommandEscapesEnv ensures env values can't break out of the export
// statement (the value is single-quote escaped via shellescape).
func TestBuildCommandEscapesEnv(t *testing.T) {
	c := &Client{}
	cfg := &execConfig{env: map[string]string{"FOO": "a'b; rm -rf /"}}

	got := c.buildCommand("echo done", cfg)

	if !strings.Contains(got, `export FOO='a'\''b; rm -rf /';`) {
		t.Errorf("env value not safely escaped in built command: %q", got)
	}
	if !strings.HasSuffix(got, "echo done") {
		t.Errorf("built command should end with the user command, got %q", got)
	}
}
