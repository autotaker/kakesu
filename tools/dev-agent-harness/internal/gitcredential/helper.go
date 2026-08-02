// Package gitcredential implements the bounded Git credential-helper surface.
package gitcredential

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/command"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/controlclient"
)

const (
	programName     = "git-credential-dev-agent"
	maxInputBytes   = 4 << 10
	failClosed      = "quit=true\n\n"
	usageDiagnostic = "git-credential-dev-agent: expected exactly one operation\n"
)

// socketPath is set only by the helper's target-specific linker flag. It has
// no environment, CLI, credential-input, config, or working-directory source.
var socketPath string

var (
	issueCapability  = controlclient.Issue
	revokeCapability = controlclient.Revoke
)

// Run executes one Git credential-helper operation.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "--version" {
		_, _ = fmt.Fprintf(stdout, "%s %s\n", programName, command.Version)
		return 0
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, _ = fmt.Fprintf(stdout, "usage: %s <get|store|erase>\n", programName)
		return 0
	}
	if len(args) != 1 {
		_, _ = io.WriteString(stderr, usageDiagnostic)
		return 2
	}
	switch args[0] {
	case "get":
		return runGet(stdin, stdout)
	case "erase":
		return runErase(stdin)
	case "store":
		discardBounded(stdin)
		return 0
	default:
		discardBounded(stdin)
		return 0
	}
}

func runGet(stdin io.Reader, stdout io.Writer) int {
	fields, ok := readCredential(stdin)
	if !ok {
		writeFailClosed(stdout)
		return 0
	}
	repository, ok := getRepository(fields)
	if !ok || !fixedAbsoluteSocket() {
		writeFailClosed(stdout)
		return 0
	}
	handle, err := issueCapability(socketPath, repository)
	if err != nil || !canonicalHandle(handle) {
		writeFailClosed(stdout)
		return 0
	}
	output := "username=x-access-token\npassword=" + handle + "\n\n"
	if !writeAll(stdout, output) {
		return 1
	}
	return 0
}

func runErase(stdin io.Reader) int {
	fields, ok := readCredential(stdin)
	if !ok || !fixedAbsoluteSocket() {
		return 1
	}
	handle, ok := eraseHandle(fields)
	if !ok || revokeCapability(socketPath, handle) != nil {
		return 1
	}
	return 0
}

func readCredential(reader io.Reader) (map[string]string, bool) {
	if reader == nil {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxInputBytes+1))
	if err != nil || len(data) > maxInputBytes || bytes.IndexByte(data, 0) >= 0 || bytes.IndexByte(data, '\r') >= 0 {
		return nil, false
	}
	if blank := bytes.Index(data, []byte("\n\n")); blank >= 0 {
		if blank+2 != len(data) {
			return nil, false
		}
		data = data[:blank]
	} else if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	fields := make(map[string]string)
	if len(data) == 0 {
		return fields, true
	}
	for _, raw := range bytes.Split(data, []byte{'\n'}) {
		if len(raw) == 0 {
			return nil, false
		}
		equals := bytes.IndexByte(raw, '=')
		if equals <= 0 || equals == len(raw)-1 {
			return nil, false
		}
		key, value := string(raw[:equals]), string(raw[equals+1:])
		if _, duplicate := fields[key]; duplicate {
			return nil, false
		}
		fields[key] = value
	}
	return fields, true
}

func getRepository(fields map[string]string) (string, bool) {
	if len(fields) != 3 || fields["protocol"] != "https" {
		return "", false
	}
	host := fields["host"]
	if host != "github.com" && host != "github.com:443" {
		return "", false
	}
	path, present := fields["path"]
	if !present || !strings.HasSuffix(path, ".git") || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return "", false
	}
	repository := strings.TrimSuffix(path, ".git")
	if !canonicalRepository(repository) || path != repository+".git" {
		return "", false
	}
	return repository, true
}

func eraseHandle(fields map[string]string) (string, bool) {
	if len(fields) != 1 {
		return "", false
	}
	handle, present := fields["password"]
	return handle, present && canonicalHandle(handle)
}

func discardBounded(reader io.Reader) {
	if reader == nil {
		return
	}
	_, _ = io.CopyN(io.Discard, reader, maxInputBytes+1)
}

func writeFailClosed(writer io.Writer) {
	_ = writeAll(writer, failClosed)
}

func writeAll(writer io.Writer, value string) bool {
	if writer == nil {
		return false
	}
	data := []byte(value)
	for len(data) > 0 {
		n, err := writer.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil || n == 0 {
			return false
		}
	}
	return true
}

func fixedAbsoluteSocket() bool {
	return filepath.IsAbs(socketPath) && filepath.Clean(socketPath) == socketPath
}

func canonicalRepository(repository string) bool {
	parts := strings.Split(repository, "/")
	return len(parts) == 2 && canonicalRepositoryPart(parts[0]) && canonicalRepositoryPart(parts[1])
}

func canonicalRepositoryPart(part string) bool {
	if part == "" || part == "." || part == ".." || !lowerAlphaNumeric(part[0]) {
		return false
	}
	for i := 1; i < len(part); i++ {
		if c := part[i]; !lowerAlphaNumeric(c) && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func lowerAlphaNumeric(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

func canonicalHandle(handle string) bool {
	if len(handle) != len("cap_")+43 || !strings.HasPrefix(handle, "cap_") {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(handle, "cap_"))
	return err == nil && len(raw) == 32 && "cap_"+base64.RawURLEncoding.EncodeToString(raw) == handle
}
