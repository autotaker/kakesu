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
	return `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"identity":{"workspace_id":"workspace-1"},"network":{"default":"deny"},"egress":{"github_repositories":["octo/repo"],"openai_models":["gpt-4o-mini"]}}`
}

func TestParseValid(t *testing.T) {
	c, err := Parse([]byte(validJSON()))
	if err != nil {
		t.Fatalf("Parse(valid): %v", err)
	}
	if c.Version != 1 || c.Network.Default != "deny" || c.Paths.ConfigDir != "/etc/dev-agent" || c.Identity.WorkspaceID != "workspace-1" || len(c.Egress.GitHubRepositories) != 1 || len(c.Egress.OpenAIModels) != 1 {
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
		{"identity-unknown", func(s string) string {
			return strings.Replace(s, `"identity":{"workspace_id":"workspace-1"}`, `"identity":{"workspace_id":"workspace-1","sentinel":"do-not-echo"}`, 1)
		}, ClassUnknown},
		{"identity-duplicate", func(s string) string {
			return strings.Replace(s, `"identity":{"workspace_id":"workspace-1"}`, `"identity":{"workspace_id":"workspace-1","workspace_id":"other"}`, 1)
		}, ClassDuplicate},
		{"version", func(s string) string { return strings.Replace(s, `"version":1`, `"version":2`, 1) }, ClassVersion},
		{"trailing", func(s string) string { return s + " {\"sentinel\":\"do-not-echo\"}" }, ClassTrailing},
		{"relative-path", func(s string) string { return strings.Replace(s, `"/etc/dev-agent"`, `"etc/dev-agent"`, 1) }, ClassSemantic},
		{"unclean-path", func(s string) string { return strings.Replace(s, `"/etc/dev-agent"`, `"/etc/../etc/dev-agent"`, 1) }, ClassSemantic},
		{"duplicate-user", func(s string) string { return strings.Replace(s, `"dev-runtime"`, `"dev-agent"`, 1) }, ClassSemantic},
		{"invalid-user", func(s string) string { return strings.Replace(s, `"dev-runtime"`, `"9runtime"`, 1) }, ClassSemantic},
		{"network", func(s string) string { return strings.Replace(s, `"default":"deny"`, `"default":"allow"`, 1) }, ClassSemantic},
		{"egress-missing-repository", func(s string) string {
			return strings.Replace(s, `"github_repositories":["octo/repo"]`, `"github_repositories":[]`, 1)
		}, ClassSemantic},
		{"egress-duplicate", func(s string) string {
			return strings.Replace(s, `"github_repositories":["octo/repo"]`, `"github_repositories":["octo/repo","octo/repo"]`, 1)
		}, ClassSemantic},
		{"egress-invalid-repository", func(s string) string { return strings.Replace(s, `"octo/repo"`, `"Octo/repo"`, 1) }, ClassSemantic},
		{"egress-invalid-model", func(s string) string { return strings.Replace(s, `"gpt-4o-mini"`, `"gpt 4o"`, 1) }, ClassSemantic},
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
	for _, fragment := range []string{`{"version":1}`, `{"version":1,"paths":{},"users":{},"identity":{},"network":{}}`, `{"version":1,"paths":{},"users":{},"network":{}}`} {
		if _, err := Parse([]byte(fragment)); err == nil {
			t.Fatalf("accepted incomplete document %q", fragment)
		}
	}
}

func TestWorkspaceIdentifierBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		value string
		valid bool
	}{
		{"one-byte", "a", true},
		{"max-byte", strings.Repeat("a", 128), true},
		{"too-long", strings.Repeat("a", 129), false},
		{"empty", "", false},
		{"leading-digit", "1workspace", true},
		{"leading-dot", ".workspace", false},
		{"trailing-symbols", "workspace._-1", true},
		{"space", "workspace id", false},
		{"unicode", "workspace-é", false},
		{"slash", "workspace/id", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := strings.Replace(validJSON(), "workspace-1", tc.value, 1)
			c, err := Parse([]byte(input))
			if tc.valid {
				if err != nil || c.Identity.WorkspaceID != tc.value {
					t.Fatalf("workspace=%q rejected: c=%#v err=%v", tc.value, c, err)
				}
				return
			}
			if c != nil || ClassOf(err) != ClassSemantic {
				t.Fatalf("workspace=%q accepted: c=%#v err=%v", tc.value, c, err)
			}
		})
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

func TestEgressAllowlistBoundariesAndCopies(t *testing.T) {
	base := validJSON()
	tooManyRepositories := make([]string, 33)
	for i := range tooManyRepositories {
		tooManyRepositories[i] = `"octo/repo` + string(rune('a'+i%26)) + `"`
	}
	tooMany := `[` + strings.Join(tooManyRepositories, ",") + `]`
	cases := []struct {
		name   string
		mutate func(string) string
	}{
		{"missing-egress", func(s string) string {
			return strings.Replace(s, `,"egress":{"github_repositories":["octo/repo"],"openai_models":["gpt-4o-mini"]}`, "", 1)
		}},
		{"empty-repositories", func(s string) string {
			return strings.Replace(s, `"github_repositories":["octo/repo"]`, `"github_repositories":[]`, 1)
		}},
		{"empty-models", func(s string) string {
			return strings.Replace(s, `"openai_models":["gpt-4o-mini"]`, `"openai_models":[]`, 1)
		}},
		{"too-many-repositories", func(s string) string { return strings.Replace(s, `["octo/repo"]`, tooMany, 1) }},
		{"too-many-models", func(s string) string {
			return strings.Replace(s, `"openai_models":["gpt-4o-mini"]`, `"openai_models":["gpt-4o-mini","m1","m2","m3","m4","m5","m6","m7","m8","m9","m10","m11","m12","m13","m14","m15","m16","m17","m18","m19","m20","m21","m22","m23","m24","m25","m26","m27","m28","m29","m30","m31","m32"]`, 1)
		}},
		{"duplicate-repository", func(s string) string {
			return strings.Replace(s, `"github_repositories":["octo/repo"]`, `"github_repositories":["octo/repo","octo/repo"]`, 1)
		}},
		{"duplicate-model", func(s string) string {
			return strings.Replace(s, `"openai_models":["gpt-4o-mini"]`, `"openai_models":["gpt-4o-mini","gpt-4o-mini"]`, 1)
		}},
		{"invalid-repository", func(s string) string { return strings.Replace(s, `"octo/repo"`, `"Octo/repo"`, 1) }},
		{"invalid-model", func(s string) string { return strings.Replace(s, `"gpt-4o-mini"`, `"gpt 4o"`, 1) }},
		{"unknown-egress-field", func(s string) string {
			return strings.Replace(s, `"openai_models":["gpt-4o-mini"]`, `"openai_models":["gpt-4o-mini"],"sentinel":"hidden"`, 1)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if c, err := Parse([]byte(tc.mutate(base))); c != nil || err == nil {
				t.Fatalf("accepted invalid egress c=%#v err=%v", c, err)
			}
		})
	}
	repositories := []string{"octo/repo"}
	models := []string{"gpt-4o-mini"}
	c := &Config{Version: 1, Paths: Paths{ConfigDir: "/etc/a", StateDir: "/var/lib/a", RuntimeDir: "/run/a"}, Users: Users{Agent: "agent", Runtime: "runtime", Broker: "broker"}, Identity: Identity{WorkspaceID: "w"}, Network: Network{Default: "deny"}, Egress: Egress{GitHubRepositories: repositories, OpenAIModels: models}}
	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	repositories[0], models[0] = "changed/repo", "changed-model"
	if c.Egress.GitHubRepositories[0] != "octo/repo" || c.Egress.OpenAIModels[0] != "gpt-4o-mini" {
		t.Fatalf("allowlist aliases caller data: %#v", c.Egress)
	}
}
