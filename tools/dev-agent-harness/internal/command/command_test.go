package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run("example", []string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run returned %d", code)
	}
	if !strings.HasPrefix(stdout.String(), "example ") || stderr.Len() != 0 {
		t.Fatalf("unexpected output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestOperationalInvocationFailsClosed(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run("example", nil, &stdout, &stderr); code == 0 {
		t.Fatal("operational invocation unexpectedly succeeded")
	}
	if !strings.Contains(stderr.String(), "refusing to start") {
		t.Fatalf("missing refusal: %q", stderr.String())
	}
}

func TestSetupCheckConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "harness.json")
	input := `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"network":{"default":"deny"}}`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run("dev-agent-harness-setup", []string{"check-config", "--config", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("check-config returned %d: %q", code, stderr.String())
	}
	if stdout.String() != "config version=1 network.default=deny validated\n" || stderr.Len() != 0 {
		t.Fatalf("unexpected output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), path) || strings.Contains(stdout.String(), "dev-agent") {
		t.Fatal("summary leaked configuration values")
	}

	if err := os.WriteFile(path, []byte(strings.Replace(input, `"deny"`, `"allow"`, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run("dev-agent-harness-setup", []string{"check-config", "--config", path}, &stdout, &stderr); code == 0 || stdout.Len() != 0 {
		t.Fatalf("invalid check-config result code=%d stdout=%q", code, stdout.String())
	}
	if strings.Contains(stderr.String(), path) || strings.Contains(stderr.String(), "allow") {
		t.Fatalf("diagnostic leaked input: %q", stderr.String())
	}
}

func TestSetupCheckConfigArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run("dev-agent-harness-setup", []string{"check-config"}, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "invalid check-config arguments") {
		t.Fatalf("unexpected result code=%d stderr=%q", code, stderr.String())
	}
}
