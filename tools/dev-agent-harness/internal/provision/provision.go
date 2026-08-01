// Package provision builds the read-only Ubuntu provision manifest.
//
// The package intentionally deals only in values. It never consults the host,
// creates a file, changes permissions, starts a process, or opens a socket.
// A later executor may consume this manifest, but this package is not an
// executor.
package provision

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/config"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
)

const (
	Version     = 1
	ActionCount = 11
	platform    = "ubuntu"
	defaultMode = "0750"
	runtimeMode = "0710"
)

// ErrorClass is a stable, non-sensitive classification for planning errors.
type ErrorClass string

const (
	ClassInvalidRoot   ErrorClass = "invalid-target-root"
	ClassInvalidPath   ErrorClass = "invalid-logical-path"
	ClassInvalidRecord ErrorClass = "invalid-manifest"
	ClassEncode        ErrorClass = "encode"
	ClassWrite         ErrorClass = "write"
)

type provisionError struct{ class ErrorClass }

func (e *provisionError) Error() string { return string(e.class) }

func fail(class ErrorClass) error { return &provisionError{class: class} }

// ClassOf maps an error to the stable diagnostic class used by command.Run.
func ClassOf(err error) ErrorClass {
	var e *provisionError
	if errors.As(err, &e) {
		return e.class
	}
	return ClassWrite
}

// Header is the first JSONL record in a version 1 manifest.
type Header struct {
	Kind        string `json:"kind"`
	Version     int    `json:"version"`
	Platform    string `json:"platform"`
	Default     string `json:"default"`
	TargetRoot  string `json:"target_root"`
	ActionCount int    `json:"action_count"`
}

// UserAction describes one locked, non-login operating-system user.
type UserAction struct {
	Kind       string `json:"kind"`
	Sequence   int    `json:"sequence"`
	Action     string `json:"action"`
	Role       string `json:"role"`
	Name       string `json:"name"`
	Home       string `json:"home"`
	Shell      string `json:"shell"`
	Locked     bool   `json:"locked"`
	CreateHome bool   `json:"create_home"`
}

// DirectoryAction describes a directory desired state. TargetPath is a
// display path under TargetRoot; it is not read or created by this package.
type DirectoryAction struct {
	Kind        string `json:"kind"`
	Sequence    int    `json:"sequence"`
	Action      string `json:"action"`
	LogicalPath string `json:"logical_path"`
	TargetPath  string `json:"target_path"`
	Mode        string `json:"mode"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
}

// ServiceAction describes a service that is deliberately not enabled or
// started. No command, unit, or process is invoked to produce this value.
type ServiceAction struct {
	Kind     string `json:"kind"`
	Sequence int    `json:"sequence"`
	Action   string `json:"action"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Enabled  bool   `json:"enabled"`
	Started  bool   `json:"started"`
}

type manifest struct {
	header      Header
	users       []UserAction
	directories []DirectoryAction
	services    []ServiceAction
}

// MapTarget maps an absolute, clean logical path below root without touching
// the filesystem. Both arguments must be absolute and clean. The resulting
// path is checked again with filepath.Rel so a lexical escape is rejected.
func MapTarget(root, logical string) (string, error) {
	if !validAbsoluteClean(root) {
		return "", fail(ClassInvalidRoot)
	}
	if !validAbsoluteClean(logical) {
		return "", fail(ClassInvalidPath)
	}
	// Removing the leading separator makes logical an ordinary path component
	// for Join; it avoids filepath.Join discarding root on Unix.
	relative := strings.TrimLeft(logical, string(filepath.Separator))
	if relative == "" {
		return "", fail(ClassInvalidPath)
	}
	target := filepath.Join(root, relative)
	rel, err := filepath.Rel(root, target)
	if err != nil || filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fail(ClassInvalidPath)
	}
	if filepath.Clean(target) != target {
		return "", fail(ClassInvalidPath)
	}
	return target, nil
}

func validAbsoluteClean(value string) bool {
	return value != "" && !strings.ContainsRune(value, '\x00') && filepath.IsAbs(value) && filepath.Clean(value) == value
}

// Build constructs and validates the complete in-memory manifest and returns
// canonical JSONL. There is no output side effect until Write is called.
func Build(c *config.Config, targetRoot string) ([]byte, error) {
	m, err := build(c, targetRoot)
	if err != nil {
		return nil, err
	}
	if err := validateManifest(m); err != nil {
		return nil, err
	}
	return encodeManifest(m)
}

// Write builds the complete payload, then invokes w exactly once. A short or
// failed write is returned and is never retried or re-emitted.
func Write(c *config.Config, targetRoot string, w io.Writer) error {
	if w == nil {
		return fail(ClassWrite)
	}
	payload, err := Build(c, targetRoot)
	if err != nil {
		return err
	}
	n, writeErr := w.Write(payload)
	if writeErr != nil {
		return fail(ClassWrite)
	}
	if n != len(payload) {
		return fail(ClassWrite)
	}
	return nil
}

func build(c *config.Config, targetRoot string) (manifest, error) {
	if c == nil {
		return manifest{}, fail(ClassInvalidRecord)
	}
	if !validAbsoluteClean(targetRoot) {
		return manifest{}, fail(ClassInvalidRoot)
	}
	if !validConfig(c) {
		return manifest{}, fail(ClassInvalidRecord)
	}
	userNames := []struct {
		role string
		name string
	}{
		{"agent", c.Users.Agent},
		{"runtime", c.Users.Runtime},
		{"broker", c.Users.Broker},
	}
	users := make([]UserAction, 0, len(userNames))
	for i, item := range userNames {
		users = append(users, UserAction{
			Kind: "action", Sequence: i + 1, Action: "user", Role: item.role,
			Name: item.name, Home: "/nonexistent", Shell: "/usr/sbin/nologin",
			Locked: true, CreateHome: false,
		})
	}
	logical := []struct {
		path  string
		owner string
		group string
		mode  string
	}{
		{c.Paths.ConfigDir, "root", c.Users.Broker, defaultMode},
		{c.Paths.StateDir, c.Users.Broker, c.Users.Broker, defaultMode},
		{c.Paths.RuntimeDir, c.Users.Broker, c.Users.Agent, runtimeMode},
		{filepath.Join(c.Paths.StateDir, "audit"), c.Users.Broker, c.Users.Broker, defaultMode},
		{filepath.Join(c.Paths.ConfigDir, "credentials"), c.Users.Broker, c.Users.Broker, "0700"},
	}
	directories := make([]DirectoryAction, 0, len(logical))
	for i, item := range logical {
		target, err := MapTarget(targetRoot, item.path)
		if err != nil {
			return manifest{}, err
		}
		directories = append(directories, DirectoryAction{
			Kind: "action", Sequence: i + 4, Action: "directory", LogicalPath: item.path,
			TargetPath: target, Mode: item.mode, Owner: item.owner, Group: item.group,
		})
	}
	services := []ServiceAction{
		{Kind: "action", Sequence: 9, Action: "service", Name: "dev-agent-broker", User: c.Users.Broker, Enabled: false, Started: false},
		{Kind: "action", Sequence: 10, Action: "service", Name: "dev-agent-egress", User: c.Users.Broker, Enabled: false, Started: false},
		{Kind: "action", Sequence: 11, Action: "service", Name: "dev-agent-approval", User: c.Users.Broker, Enabled: false, Started: false},
	}
	return manifest{
		header: Header{Kind: "manifest", Version: Version, Platform: platform, Default: "deny", TargetRoot: targetRoot, ActionCount: ActionCount},
		users:  users, directories: directories, services: services,
	}, nil
}

func validConfig(c *config.Config) bool {
	if c.Version != Version || c.Network.Default != "deny" || !validWorkspaceID(c.Identity.WorkspaceID) {
		return false
	}
	if len(c.Egress.GitHubRepositories) < 1 || len(c.Egress.GitHubRepositories) > 32 || len(c.Egress.OpenAIModels) < 1 || len(c.Egress.OpenAIModels) > 32 {
		return false
	}
	if _, err := egresspolicy.New(egresspolicy.Rules{
		GitHubRepositories: c.Egress.GitHubRepositories,
		OpenAIModels:       c.Egress.OpenAIModels,
		MaxBodyBytes:       64 * 1024,
		MaxOutputTokens:    4096,
	}); err != nil {
		return false
	}
	paths := []string{c.Paths.ConfigDir, c.Paths.StateDir, c.Paths.RuntimeDir}
	for _, path := range paths {
		if !validAbsoluteClean(path) {
			return false
		}
	}
	if paths[0] == paths[1] || paths[0] == paths[2] || paths[1] == paths[2] {
		return false
	}
	users := []string{c.Users.Agent, c.Users.Runtime, c.Users.Broker}
	for _, user := range users {
		if !validLinuxUser(user) {
			return false
		}
	}
	return users[0] != users[1] && users[0] != users[2] && users[1] != users[2]
}

func validWorkspaceID(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if i == 0 {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return false
			}
			continue
		}
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func validLinuxUser(value string) bool {
	if len(value) == 0 || len(value) > 32 {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
			continue
		}
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func validateManifest(m manifest) error {
	if m.header.Kind != "manifest" || m.header.Version != Version || m.header.Platform != platform || m.header.Default != "deny" || m.header.ActionCount != ActionCount || !validAbsoluteClean(m.header.TargetRoot) {
		return fail(ClassInvalidRecord)
	}
	if len(m.users) != 3 || len(m.directories) != 5 || len(m.services) != 3 {
		return fail(ClassInvalidRecord)
	}
	for i, action := range m.users {
		roles := [...]string{"agent", "runtime", "broker"}
		if action.Kind != "action" || action.Action != "user" || action.Sequence != i+1 || action.Role != roles[i] || !validLinuxUser(action.Name) || action.Home != "/nonexistent" || action.Shell != "/usr/sbin/nologin" || !action.Locked || action.CreateHome {
			return fail(ClassInvalidRecord)
		}
	}
	if m.users[0].Name == m.users[1].Name || m.users[0].Name == m.users[2].Name || m.users[1].Name == m.users[2].Name {
		return fail(ClassInvalidRecord)
	}
	broker := m.users[2].Name
	for i, action := range m.directories {
		wantMode := defaultMode
		if i == 2 {
			wantMode = runtimeMode
		} else if i == 4 {
			wantMode = "0700"
		}
		if action.Kind != "action" || action.Action != "directory" || action.Sequence != i+4 || !validAbsoluteClean(action.LogicalPath) || action.TargetPath == "" || action.Mode != wantMode || action.Owner == "" || action.Group == "" {
			return fail(ClassInvalidRecord)
		}
		target, err := MapTarget(m.header.TargetRoot, action.LogicalPath)
		if err != nil || target != action.TargetPath {
			return fail(ClassInvalidRecord)
		}
		if i == 0 {
			if action.Owner != "root" || action.Group != broker {
				return fail(ClassInvalidRecord)
			}
		} else if action.Owner != broker || (i != 2 && action.Group != broker) {
			return fail(ClassInvalidRecord)
		}
		if i == 2 && action.Group != m.users[0].Name {
			return fail(ClassInvalidRecord)
		}
	}
	if len(m.directories) == 5 && (filepath.Dir(m.directories[3].LogicalPath) != m.directories[1].LogicalPath || filepath.Base(m.directories[3].LogicalPath) != "audit") {
		return fail(ClassInvalidRecord)
	}
	if len(m.directories) == 5 && (filepath.Dir(m.directories[4].LogicalPath) != m.directories[0].LogicalPath || filepath.Base(m.directories[4].LogicalPath) != "credentials" || m.directories[4].Mode != "0700" || m.directories[4].Owner != broker || m.directories[4].Group != broker) {
		return fail(ClassInvalidRecord)
	}
	wantNames := [...]string{"dev-agent-broker", "dev-agent-egress", "dev-agent-approval"}
	for i, action := range m.services {
		if action.Kind != "action" || action.Action != "service" || action.Sequence != i+9 || action.Name != wantNames[i] || action.User != broker || action.Enabled || action.Started {
			return fail(ClassInvalidRecord)
		}
	}
	return nil
}

func encodeManifest(m manifest) ([]byte, error) {
	var out []byte
	appendLine := func(value any) error {
		line, err := json.Marshal(value)
		if err != nil {
			return err
		}
		out = append(out, line...)
		out = append(out, '\n')
		return nil
	}
	if err := appendLine(m.header); err != nil {
		return nil, fmt.Errorf("%w: header", fail(ClassEncode))
	}
	for _, action := range m.users {
		if err := appendLine(action); err != nil {
			return nil, fmt.Errorf("%w: user", fail(ClassEncode))
		}
	}
	for _, action := range m.directories {
		if err := appendLine(action); err != nil {
			return nil, fmt.Errorf("%w: directory", fail(ClassEncode))
		}
	}
	for _, action := range m.services {
		if err := appendLine(action); err != nil {
			return nil, fmt.Errorf("%w: service", fail(ClassEncode))
		}
	}
	return out, nil
}
