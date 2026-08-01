package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func validJSON() string {
	return `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"network":{"default":"deny"}}`
}

func TestParseValid(t *testing.T) {
	c, err := Parse([]byte(validJSON()))
	if err != nil {
		t.Fatalf("Parse(valid): %v", err)
	}
	if c.Version != 1 || c.Network.Default != "deny" || c.Paths.ConfigDir != "/etc/dev-agent" {
		t.Fatalf("unexpected config: %#v", c)
	}
}

func TestParseRejectsStrictAndSemanticCases(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(string) string
		class  ErrorClass
	}{
		{"duplicate", func(s string) string { return strings.Replace(s, `"version":1`, `"version":1,"version":1`, 1) }, ClassDuplicate},
		{"unknown", func(s string) string {
			return strings.Replace(s, `"network":{"default":"deny"}`, `"network":{"default":"deny","sentinel":"do-not-echo"}`, 1)
		}, ClassUnknown},
		{"version", func(s string) string { return strings.Replace(s, `"version":1`, `"version":2`, 1) }, ClassVersion},
		{"trailing", func(s string) string { return s + " {\"sentinel\":\"do-not-echo\"}" }, ClassTrailing},
		{"relative-path", func(s string) string { return strings.Replace(s, `"/etc/dev-agent"`, `"etc/dev-agent"`, 1) }, ClassSemantic},
		{"unclean-path", func(s string) string { return strings.Replace(s, `"/etc/dev-agent"`, `"/etc/../etc/dev-agent"`, 1) }, ClassSemantic},
		{"duplicate-user", func(s string) string { return strings.Replace(s, `"dev-runtime"`, `"dev-agent"`, 1) }, ClassSemantic},
		{"invalid-user", func(s string) string { return strings.Replace(s, `"dev-runtime"`, `"9runtime"`, 1) }, ClassSemantic},
		{"network", func(s string) string { return strings.Replace(s, `"default":"deny"`, `"default":"allow"`, 1) }, ClassSemantic},
		{"allowlist", func(s string) string {
			return strings.Replace(s, `"default":"deny"`, `"default":"deny","allowlist":[]`, 1)
		}, ClassUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.mutate(validJSON())))
			if err == nil || ClassOf(err) != tc.class {
				t.Fatalf("class=%q want %q err=%v", ClassOf(err), tc.class, err)
			}
		})
	}
}

func TestParseRejectsMissingFields(t *testing.T) {
	for _, fragment := range []string{`{"version":1}`, `{"version":1,"paths":{},"users":{},"network":{}}`} {
		if _, err := Parse([]byte(fragment)); err == nil {
			t.Fatalf("accepted incomplete document %q", fragment)
		}
	}
}

func TestLoadFilePolicy(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(validJSON()), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatalf("safe file rejected: %v", err)
	}
	if err := os.Chmod(p, 0660); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); ClassOf(err) != ClassFilePolicy {
		t.Fatalf("writable file class=%q", ClassOf(err))
	}
	if err := os.Chmod(p, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, make([]byte, MaxFileSize+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); ClassOf(err) != ClassFilePolicy {
		t.Fatalf("oversize file class=%q", ClassOf(err))
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(p, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, err := Load(link); ClassOf(err) != ClassFilePolicy {
		t.Fatalf("symlink class=%q", ClassOf(err))
	}
	if _, err := Load(dir); ClassOf(err) != ClassFilePolicy {
		t.Fatalf("directory class=%q", ClassOf(err))
	}
	fifo := filepath.Join(dir, "config.fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Skipf("FIFO unsupported: %v", err)
	}
	if _, err := Load(fifo); ClassOf(err) != ClassFilePolicy {
		t.Fatalf("FIFO class=%q", ClassOf(err))
	}
}

func TestClassOfNeverLeaksInput(t *testing.T) {
	_, err := Parse([]byte(`{"version":1,"sentinel":"credential-value"}`))
	if err == nil || strings.Contains(err.Error(), "credential-value") {
		t.Fatalf("error leaked input: %v", err)
	}
	if errors.Is(err, nil) {
		t.Fatal("unexpected nil error")
	}
}
