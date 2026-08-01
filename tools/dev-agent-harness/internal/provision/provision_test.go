package provision

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Version: 1,
		Paths: config.Paths{
			ConfigDir:  "/etc/dev-agent-harness",
			StateDir:   "/var/lib/dev-agent-harness",
			RuntimeDir: "/run/dev-agent-harness",
		},
		Users:   config.Users{Agent: "dev-agent", Runtime: "dev-runtime", Broker: "dev-broker"},
		Network: config.Network{Default: "deny"},
	}
}

func jsonLines(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	parts := bytes.Split(data, []byte{'\n'})
	if len(parts) < 2 || len(parts[len(parts)-1]) != 0 {
		t.Fatalf("JSONL does not end in exactly one newline: %q", data)
	}
	lines := make([]map[string]any, 0, len(parts)-1)
	for i, part := range parts[:len(parts)-1] {
		if len(part) == 0 {
			t.Fatalf("empty JSONL record at %d", i)
		}
		var value map[string]any
		if err := json.Unmarshal(part, &value); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
		lines = append(lines, value)
	}
	return lines
}

func TestBuildCanonicalManifest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	c := testConfig()
	one, err := Build(c, root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	two, err := Build(c, root)
	if err != nil {
		t.Fatalf("Build repeat: %v", err)
	}
	if !bytes.Equal(one, two) {
		t.Fatalf("manifest is not deterministic:\n%s\n%s", one, two)
	}
	lines := jsonLines(t, one)
	if len(lines) != 11 {
		t.Fatalf("record count=%d, want 11", len(lines))
	}
	header := lines[0]
	for key, want := range map[string]any{
		"kind": "manifest", "version": float64(1), "platform": "ubuntu",
		"default": "deny", "target_root": root, "action_count": float64(10),
	} {
		if header[key] != want {
			t.Errorf("header[%q]=%v, want %v", key, header[key], want)
		}
	}
	wantActions := []string{"user", "user", "user", "directory", "directory", "directory", "directory", "service", "service", "service"}
	for i, action := range lines[1:] {
		if action["kind"] != "action" || action["sequence"] != float64(i+1) || action["action"] != wantActions[i] {
			t.Errorf("action %d common fields=%v", i+1, action)
		}
	}
	if !bytes.Contains(one, []byte(`"kind":"manifest","version":1,"platform":"ubuntu","default":"deny"`)) {
		t.Fatal("header fields are not canonical and compact")
	}
	want := strings.Join([]string{
		fmt.Sprintf(`{"kind":"manifest","version":1,"platform":"ubuntu","default":"deny","target_root":%q,"action_count":10}`, root),
		`{"kind":"action","sequence":1,"action":"user","role":"agent","name":"dev-agent","home":"/nonexistent","shell":"/usr/sbin/nologin","locked":true,"create_home":false}`,
		`{"kind":"action","sequence":2,"action":"user","role":"runtime","name":"dev-runtime","home":"/nonexistent","shell":"/usr/sbin/nologin","locked":true,"create_home":false}`,
		`{"kind":"action","sequence":3,"action":"user","role":"broker","name":"dev-broker","home":"/nonexistent","shell":"/usr/sbin/nologin","locked":true,"create_home":false}`,
		fmt.Sprintf(`{"kind":"action","sequence":4,"action":"directory","logical_path":"/etc/dev-agent-harness","target_path":%q,"mode":"0750","owner":"root","group":"dev-broker"}`, filepath.Join(root, "etc/dev-agent-harness")),
		fmt.Sprintf(`{"kind":"action","sequence":5,"action":"directory","logical_path":"/var/lib/dev-agent-harness","target_path":%q,"mode":"0750","owner":"dev-broker","group":"dev-broker"}`, filepath.Join(root, "var/lib/dev-agent-harness")),
		fmt.Sprintf(`{"kind":"action","sequence":6,"action":"directory","logical_path":"/run/dev-agent-harness","target_path":%q,"mode":"0710","owner":"dev-broker","group":"dev-agent"}`, filepath.Join(root, "run/dev-agent-harness")),
		fmt.Sprintf(`{"kind":"action","sequence":7,"action":"directory","logical_path":"/var/lib/dev-agent-harness/audit","target_path":%q,"mode":"0750","owner":"dev-broker","group":"dev-broker"}`, filepath.Join(root, "var/lib/dev-agent-harness/audit")),
		`{"kind":"action","sequence":8,"action":"service","name":"dev-agent-broker","user":"dev-broker","enabled":false,"started":false}`,
		`{"kind":"action","sequence":9,"action":"service","name":"dev-agent-egress","user":"dev-broker","enabled":false,"started":false}`,
		`{"kind":"action","sequence":10,"action":"service","name":"dev-agent-approval","user":"dev-broker","enabled":false,"started":false}`,
	}, "\n") + "\n"
	if string(one) != want {
		t.Fatalf("canonical JSONL mismatch:\n got: %s\nwant: %s", one, want)
	}
}

func TestBuildUsersDirectoriesAndServices(t *testing.T) {
	root := filepath.Join(t.TempDir(), "target")
	lines := jsonLines(t, mustBuild(t, testConfig(), root))
	wantUsers := []struct{ role, name string }{
		{"agent", "dev-agent"}, {"runtime", "dev-runtime"}, {"broker", "dev-broker"},
	}
	for i, want := range wantUsers {
		action := lines[i+1]
		if action["role"] != want.role || action["name"] != want.name || action["home"] != "/nonexistent" || action["shell"] != "/usr/sbin/nologin" || action["locked"] != true || action["create_home"] != false {
			t.Errorf("user %d=%v", i, action)
		}
	}
	wantDirs := []struct {
		logical, owner, group string
	}{
		{"/etc/dev-agent-harness", "root", "dev-broker"},
		{"/var/lib/dev-agent-harness", "dev-broker", "dev-broker"},
		{"/run/dev-agent-harness", "dev-broker", "dev-agent"},
		{"/var/lib/dev-agent-harness/audit", "dev-broker", "dev-broker"},
	}
	for i, want := range wantDirs {
		action := lines[i+4]
		mapped, err := MapTarget(root, want.logical)
		if err != nil {
			t.Fatal(err)
		}
		wantMode := "0750"
		if i == 2 {
			wantMode = "0710"
		}
		if action["logical_path"] != want.logical || action["target_path"] != mapped || action["mode"] != wantMode || action["owner"] != want.owner || action["group"] != want.group {
			t.Errorf("directory %d=%v", i, action)
		}
	}
	wantServices := []string{"dev-agent-broker", "dev-agent-egress", "dev-agent-approval"}
	for i, name := range wantServices {
		action := lines[i+8]
		if action["name"] != name || action["user"] != "dev-broker" || action["enabled"] != false || action["started"] != false {
			t.Errorf("service %d=%v", i, action)
		}
	}
}

func mustBuild(t *testing.T, c *config.Config, root string) []byte {
	t.Helper()
	data, err := Build(c, root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return data
}

func TestMapTargetContainmentAndCleanBoundaries(t *testing.T) {
	root := filepath.Join(t.TempDir(), "target")
	valid := []string{"/etc/dev-agent", "/var/lib/dev-agent/audit", "/run/dev-agent"}
	for _, logical := range valid {
		got, err := MapTarget(root, logical)
		if err != nil || !strings.HasPrefix(got, root+string(filepath.Separator)) {
			t.Errorf("MapTarget(%q)=%q, %v", logical, got, err)
		}
	}
	if got, err := MapTarget(root, root); err != nil || got != filepath.Join(root, strings.TrimLeft(root, string(filepath.Separator))) {
		t.Fatalf("same text in separate coordinate spaces mapped incorrectly: got=%q err=%v", got, err)
	}
	for _, tc := range []struct {
		name, root, logical string
	}{
		{"empty root", "", "/etc"},
		{"relative root", "target", "/etc"},
		{"unclean root", root + "/..", "/etc"},
		{"nul root", root + "\x00bad", "/etc"},
		{"empty logical", root, ""},
		{"relative logical", root, "etc"},
		{"unclean logical", root, "/var/../etc"},
		{"nul logical", root, "/etc\x00bad"},
		{"root logical", root, "/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := MapTarget(tc.root, tc.logical); err == nil {
				t.Fatalf("accepted root escape/boundary root=%q logical=%q", tc.root, tc.logical)
			}
		})
	}
}

func TestBuildRejectsDirectInvalidConfigValues(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	cases := []struct {
		name   string
		mutate func(*config.Config)
	}{
		{"version", func(c *config.Config) { c.Version = 2 }},
		{"network", func(c *config.Config) { c.Network.Default = "allow" }},
		{"duplicate path", func(c *config.Config) { c.Paths.StateDir = c.Paths.ConfigDir }},
		{"relative path", func(c *config.Config) { c.Paths.ConfigDir = "etc/dev-agent" }},
		{"unclean path", func(c *config.Config) { c.Paths.ConfigDir = "/etc/../etc/dev-agent" }},
		{"invalid user", func(c *config.Config) { c.Users.Agent = "9agent" }},
		{"duplicate user", func(c *config.Config) { c.Users.Runtime = c.Users.Agent }},
		{"empty root", func(c *config.Config) { _ = c }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := testConfig()
			tc.mutate(c)
			targetRoot := root
			if tc.name == "empty root" {
				targetRoot = ""
			}
			if data, err := Build(c, targetRoot); err == nil || len(data) != 0 {
				t.Fatalf("accepted invalid config: data=%q err=%v", data, err)
			}
		})
	}
}

type recordingWriter struct {
	bytes.Buffer
	calls int
	err   error
	short bool
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.short {
		n := len(p) / 2
		_, _ = w.Buffer.Write(p[:n])
		return n, nil
	}
	if w.err != nil {
		n := len(p) / 3
		_, _ = w.Buffer.Write(p[:n])
		return n, w.err
	}
	return w.Buffer.Write(p)
}

func TestWriteSingleCallAndWriterFailures(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	var good recordingWriter
	if err := Write(testConfig(), root, &good); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if good.calls != 1 || len(jsonLines(t, good.Bytes())) != 11 {
		t.Fatalf("successful writer calls=%d bytes=%d", good.calls, good.Len())
	}
	for _, tc := range []struct {
		name  string
		short bool
		err   error
	}{
		{"short", true, nil},
		{"error", false, errors.New("sentinel writer error")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &recordingWriter{short: tc.short, err: tc.err}
			err := Write(testConfig(), root, w)
			if err == nil || ClassOf(err) != ClassWrite || w.calls != 1 {
				t.Fatalf("err=%v class=%q calls=%d", err, ClassOf(err), w.calls)
			}
		})
	}
	if err := Write(testConfig(), root, nil); err == nil {
		t.Fatal("nil writer unexpectedly accepted")
	}
}

func TestManifestMutationGuards(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	base, err := build(testConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		mutate func(*manifest)
	}{
		{"order", func(m *manifest) { m.users[0].Sequence = 2 }},
		{"disabled-stopped", func(m *manifest) { m.services[0].Enabled = true }},
		{"containment", func(m *manifest) { m.directories[0].TargetPath = "/outside" }},
		{"owner", func(m *manifest) { m.directories[0].Owner = m.users[2].Name }},
		{"audit", func(m *manifest) { m.directories[3].LogicalPath = "/tmp/audit" }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			m := base
			m.users = append([]UserAction(nil), base.users...)
			m.directories = append([]DirectoryAction(nil), base.directories...)
			m.services = append([]ServiceAction(nil), base.services...)
			tc.mutate(&m)
			if err := validateManifest(m); err == nil {
				t.Fatal("mutation bypassed validator")
			}
		})
	}
}

func TestBuildDoesNotTouchTargetRoot(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(root, "sentinel")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(testConfig(), root); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("target root sentinel changed")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel" {
		t.Fatalf("target root tree changed: entries=%v err=%v", entries, err)
	}
}

func TestVerifyAcceptsOnlyCanonicalBytes(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "target")
	path := filepath.Join(dir, "manifest.jsonl")
	want := mustBuild(t, testConfig(), root)
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Verify(testConfig(), path, root); err != nil {
		t.Fatalf("canonical manifest rejected: %v", err)
	}
	otherConfig := testConfig()
	otherConfig.Users.Agent = "other-agent"
	if err := Verify(otherConfig, path, root); ClassOf(err) != ClassManifestMismatch {
		t.Fatalf("different config class=%q err=%v", ClassOf(err), err)
	}
	if err := Verify(testConfig(), path, filepath.Join(dir, "other-target")); ClassOf(err) != ClassManifestMismatch {
		t.Fatalf("different target root class=%q err=%v", ClassOf(err), err)
	}
	for _, tc := range []struct {
		name string
		data func([]byte) []byte
	}{
		{"append", func(data []byte) []byte { return append(append([]byte(nil), data...), ' ') }},
		{"change", func(data []byte) []byte {
			mutated := append([]byte(nil), data...)
			mutated[len(mutated)-2] = 'x'
			return mutated
		}},
		{"delete", func(data []byte) []byte { return append([]byte(nil), data[:len(data)-1]...) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, tc.data(want), 0600); err != nil {
				t.Fatal(err)
			}
			if err := Verify(testConfig(), path, root); ClassOf(err) != ClassManifestMismatch {
				t.Fatalf("class=%q err=%v, want %q", ClassOf(err), err, ClassManifestMismatch)
			}
		})
	}
}

func TestVerifyManifestFilePolicy(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "target")
	path := filepath.Join(dir, "manifest.jsonl")
	want := mustBuild(t, testConfig(), root)
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0660); err != nil {
		t.Fatal(err)
	}
	if err := Verify(testConfig(), path, root); ClassOf(err) != ClassManifestFilePolicy {
		t.Fatalf("writable file class=%q err=%v", ClassOf(err), err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "manifest-link")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := Verify(testConfig(), link, root); ClassOf(err) != ClassManifestFilePolicy {
		t.Fatalf("symlink class=%q err=%v", ClassOf(err), err)
	}
	if err := Verify(testConfig(), dir, root); ClassOf(err) != ClassManifestFilePolicy {
		t.Fatalf("directory class=%q err=%v", ClassOf(err), err)
	}
	if err := os.WriteFile(path, make([]byte, MaxManifestSize+1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Verify(testConfig(), path, root); ClassOf(err) != ClassManifestFilePolicy {
		t.Fatalf("oversize class=%q err=%v", ClassOf(err), err)
	}
	if err := os.WriteFile(path, make([]byte, MaxManifestSize), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Verify(testConfig(), path, root); ClassOf(err) != ClassManifestMismatch {
		t.Fatalf("boundary-size class=%q err=%v", ClassOf(err), err)
	}
}

func TestVerifyRejectsReadTimeMetadataChange(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "target")
	path := filepath.Join(dir, "manifest.jsonl")
	if err := os.WriteFile(path, mustBuild(t, testConfig(), root), 0600); err != nil {
		t.Fatal(err)
	}
	manifestReadBeforeHook = func(f *os.File) { _ = f.Chmod(0640) }
	defer func() { manifestReadBeforeHook = nil }()
	if err := Verify(testConfig(), path, root); ClassOf(err) != ClassManifestFilePolicy {
		t.Fatalf("metadata change class=%q err=%v", ClassOf(err), err)
	}
}

func TestVerifyMapsClosedDescriptorReadToManifestRead(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "target")
	path := filepath.Join(dir, "manifest.jsonl")
	if err := os.WriteFile(path, mustBuild(t, testConfig(), root), 0600); err != nil {
		t.Fatal(err)
	}
	manifestReadBeforeHook = func(f *os.File) { _ = f.Close() }
	defer func() { manifestReadBeforeHook = nil }()
	if err := Verify(testConfig(), path, root); ClassOf(err) != ClassManifestRead {
		t.Fatalf("closed descriptor class=%q err=%v, want %q", ClassOf(err), err, ClassManifestRead)
	}
}

func TestVerifySuccessDoesNotMutateInputsOrTargetRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "target")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(root, "sentinel")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "manifest.jsonl")
	want := mustBuild(t, testConfig(), root)
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	manifestBefore, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	manifestInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	manifestMode := manifestInfo.Mode()
	sentinelBefore, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	entriesBefore, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(testConfig(), path, root); err != nil {
		t.Fatalf("canonical manifest rejected: %v", err)
	}
	manifestAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	sentinelAfter, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	entriesAfter, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(manifestBefore, manifestAfter) || manifestMode != afterInfo.Mode() {
		t.Fatalf("manifest changed: bytes=%t mode=%v/%v", !bytes.Equal(manifestBefore, manifestAfter), manifestMode, afterInfo.Mode())
	}
	if !bytes.Equal(sentinelBefore, sentinelAfter) {
		t.Fatal("target root sentinel changed")
	}
	if len(entriesBefore) != len(entriesAfter) {
		t.Fatalf("target root listing changed: before=%d after=%d", len(entriesBefore), len(entriesAfter))
	}
	for i := range entriesBefore {
		if entriesBefore[i].Name() != entriesAfter[i].Name() {
			t.Fatalf("target root listing changed: before=%q after=%q", entriesBefore[i].Name(), entriesAfter[i].Name())
		}
	}
}

func TestVerifyDoesNotReopenManifestPath(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "target")
	path := filepath.Join(dir, "manifest.jsonl")
	want := mustBuild(t, testConfig(), root)
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	manifestReadBeforeHook = func(_ *os.File) {
		backup := path + ".old"
		if err := os.Rename(path, backup); err != nil {
			t.Fatalf("rename manifest: %v", err)
		}
		if err := os.WriteFile(path, []byte("replacement"), 0600); err != nil {
			t.Fatalf("replace manifest: %v", err)
		}
	}
	defer func() { manifestReadBeforeHook = nil }()
	if err := Verify(testConfig(), path, root); err != nil {
		t.Fatalf("descriptor contents were not used: %v", err)
	}
}
