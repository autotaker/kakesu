// Package config implements the versioned, deny-by-default harness
// configuration boundary.  It deliberately has no knowledge of services,
// credentials, or the network: loading a configuration is read-only.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
)

const MaxFileSize = 64 * 1024

type ErrorClass string

const (
	ClassIO         ErrorClass = "io"
	ClassFilePolicy ErrorClass = "file-policy"
	ClassParse      ErrorClass = "parse"
	ClassDuplicate  ErrorClass = "duplicate"
	ClassUnknown    ErrorClass = "unknown-field"
	ClassVersion    ErrorClass = "version"
	ClassTrailing   ErrorClass = "trailing-data"
	ClassSemantic   ErrorClass = "semantic"
)

// Error is intentionally safe to print: its text never contains a path or
// decoder error (which could include an input fragment).
type Error struct {
	Class ErrorClass
}

func (e *Error) Error() string { return string(e.Class) }

func fail(class ErrorClass) error { return &Error{Class: class} }

type Config struct {
	Version  int      `json:"version"`
	Paths    Paths    `json:"paths"`
	Users    Users    `json:"users"`
	Identity Identity `json:"identity"`
	Network  Network  `json:"network"`
	Egress   Egress   `json:"egress"`
}

type Paths struct {
	ConfigDir  string `json:"config_dir"`
	StateDir   string `json:"state_dir"`
	RuntimeDir string `json:"runtime_dir"`
}

type Users struct {
	Agent   string `json:"agent"`
	Runtime string `json:"runtime"`
	Broker  string `json:"broker"`
}

type Identity struct {
	WorkspaceID string `json:"workspace_id"`
}

type Network struct {
	Default string `json:"default"`
}

// Egress contains the two strict provider allowlists used by the egress
// service.  Values are copied at the configuration boundary and are never
// interpreted as paths or URLs by this package.
type Egress struct {
	GitHubRepositories []string `json:"github_repositories"`
	OpenAIModels       []string `json:"openai_models"`
}

var linuxUser = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,31}$`)

// Load opens and validates path using one descriptor.  In particular, it
// never stats a path and then re-opens it, and does not follow a final symlink.
func Load(path string) (*Config, error) {
	if path == "" {
		return nil, fail(ClassFilePolicy)
	}
	// O_NONBLOCK ensures a FIFO or other special node cannot make a
	// configuration check wait for an external writer before its type is
	// rejected by fstat.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fail(ClassFilePolicy)
	}
	f := os.NewFile(uintptr(fd), "harness-config")
	if f == nil {
		_ = syscall.Close(fd)
		return nil, fail(ClassIO)
	}
	defer f.Close()

	before, err := f.Stat()
	if err != nil {
		return nil, fail(ClassFilePolicy)
	}
	if !before.Mode().IsRegular() || before.Mode().Perm()&0o022 != 0 || before.Size() > MaxFileSize {
		return nil, fail(ClassFilePolicy)
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxFileSize+1))
	if err != nil {
		return nil, fail(ClassIO)
	}
	after, err := f.Stat()
	if err != nil {
		return nil, fail(ClassFilePolicy)
	}
	if !os.SameFile(before, after) || !after.Mode().IsRegular() || after.Mode().Perm()&0o022 != 0 ||
		after.Size() != before.Size() || after.Size() > MaxFileSize || int64(len(data)) != after.Size() {
		return nil, fail(ClassFilePolicy)
	}
	return Parse(data)
}

// Parse validates a complete V1 JSON document.  The token pass is separate
// from typed decoding because encoding/json otherwise silently accepts a
// duplicate object key.
func Parse(data []byte) (*Config, error) {
	if len(data) == 0 || len(data) > MaxFileSize {
		return nil, fail(ClassParse)
	}
	scanner := json.NewDecoder(bytes.NewReader(data))
	if err := scanValue(scanner); err != nil {
		var duplicate duplicateError
		if errors.As(err, &duplicate) {
			return nil, fail(ClassDuplicate)
		}
		return nil, fail(ClassParse)
	}
	if _, err := scanner.Token(); err != io.EOF {
		if err == nil {
			return nil, fail(ClassTrailing)
		}
		return nil, fail(ClassTrailing)
	}

	var c Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		var syntax *json.SyntaxError
		if errors.As(err, &syntax) {
			return nil, fail(ClassParse)
		}
		// Unknown-field text is intentionally not exposed to callers.
		if bytes.Contains([]byte(err.Error()), []byte("unknown field")) {
			return nil, fail(ClassUnknown)
		}
		if c.Version != 1 {
			return nil, fail(ClassVersion)
		}
		return nil, fail(ClassSemantic)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fail(ClassTrailing)
	}
	if c.Version != 1 {
		return nil, fail(ClassVersion)
	}
	if err := validate(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func scanValue(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("object key")
			}
			if _, exists := seen[name]; exists {
				return duplicateError{}
			}
			seen[name] = struct{}{}
			if err := scanValue(d); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	case '[':
		for d.More() {
			if err := scanValue(d); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	default:
		return fmt.Errorf("unexpected delimiter")
	}
}

type duplicateError struct{}

func (duplicateError) Error() string { return "duplicate key" }

func validate(c *Config) error {
	if err := validatePath(c.Paths.ConfigDir); err != nil {
		return err
	}
	if err := validatePath(c.Paths.StateDir); err != nil {
		return err
	}
	if err := validatePath(c.Paths.RuntimeDir); err != nil {
		return err
	}
	if c.Paths.ConfigDir == c.Paths.StateDir || c.Paths.ConfigDir == c.Paths.RuntimeDir || c.Paths.StateDir == c.Paths.RuntimeDir {
		return fail(ClassSemantic)
	}
	if err := validateUser(c.Users.Agent); err != nil {
		return err
	}
	if err := validateUser(c.Users.Runtime); err != nil {
		return err
	}
	if err := validateUser(c.Users.Broker); err != nil {
		return err
	}
	if c.Users.Agent == c.Users.Runtime || c.Users.Agent == c.Users.Broker || c.Users.Runtime == c.Users.Broker {
		return fail(ClassSemantic)
	}
	if !validateIdentifier(c.Identity.WorkspaceID) {
		return fail(ClassSemantic)
	}
	if c.Network.Default != "deny" {
		return fail(ClassSemantic)
	}
	if len(c.Egress.GitHubRepositories) < 1 || len(c.Egress.GitHubRepositories) > 32 || len(c.Egress.OpenAIModels) < 1 || len(c.Egress.OpenAIModels) > 32 {
		return fail(ClassSemantic)
	}
	if _, err := egresspolicy.New(egresspolicy.Rules{
		GitHubRepositories: c.Egress.GitHubRepositories,
		OpenAIModels:       c.Egress.OpenAIModels,
		MaxBodyBytes:       64 * 1024,
		MaxOutputTokens:    4096,
	}); err != nil {
		return fail(ClassSemantic)
	}
	// Keep caller-owned slices from becoming aliases of the loaded config.
	c.Egress.GitHubRepositories = append([]string(nil), c.Egress.GitHubRepositories...)
	c.Egress.OpenAIModels = append([]string(nil), c.Egress.OpenAIModels...)
	return nil
}

func validateIdentifier(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for index := 0; index < len(value); index++ {
		char := value[index]
		if index == 0 {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
				return false
			}
			continue
		}
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '.' || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func validatePath(value string) error {
	if value == "" || !filepath.IsAbs(value) || filepath.Clean(value) != value {
		return fail(ClassSemantic)
	}
	return nil
}

func validateUser(value string) error {
	if !linuxUser.MatchString(value) {
		return fail(ClassSemantic)
	}
	return nil
}

// ClassOf maps all errors to the stable machine-readable class used by the
// setup command, including unexpected internal errors.
func ClassOf(err error) ErrorClass {
	var e *Error
	if errors.As(err, &e) {
		return e.Class
	}
	return ClassIO
}
