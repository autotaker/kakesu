package gitcredential

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/command"
)

const fixedTestSocket = "/run/dev-agent-harness/egress.sock"

var canonicalTestHandle = "cap_" + strings.Repeat("A", 43)

type endlessReader struct{ bytes int }

func (r *endlessReader) Read(buffer []byte) (int, error) {
	for i := range buffer {
		buffer[i] = 'x'
	}
	r.bytes += len(buffer)
	return len(buffer), nil
}

type shortWriter struct {
	builder strings.Builder
	limit   int
}

func (w *shortWriter) Write(value []byte) (int, error) {
	if len(value) > w.limit {
		value = value[:w.limit]
	}
	return w.builder.Write(value)
}

func TestGetSuccessForBlankAndEOFTermination(t *testing.T) {
	for name, input := range map[string]string{
		"blank":       "path=octo/repo.git\nhost=github.com\nprotocol=https\n\n",
		"eof":         "protocol=https\nhost=github.com:443\npath=octo/repo.git",
		"newline eof": "protocol=https\nhost=github.com\npath=octo/repo.git\n",
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			withClientSeams(t, func(path, repository string) (string, error) {
				calls++
				if path != fixedTestSocket || repository != "octo/repo" {
					t.Fatalf("path/repository reached unexpected values")
				}
				return canonicalTestHandle, nil
			}, nil)
			var stdout, stderr strings.Builder
			code := Run([]string{"get"}, strings.NewReader(input), &stdout, &stderr)
			want := "username=x-access-token\npassword=" + canonicalTestHandle + "\n\n"
			if code != 0 || stdout.String() != want || stderr.Len() != 0 || calls != 1 {
				t.Fatalf("code=%d stdout=%q stderr=%q calls=%d", code, stdout.String(), stderr.String(), calls)
			}
		})
	}
}

func TestGetRejectsAmbiguousContextBeforeControl(t *testing.T) {
	base := "protocol=https\nhost=github.com\npath=octo/repo.git"
	cases := map[string]string{
		"empty":            "",
		"missing protocol": "host=github.com\npath=octo/repo.git",
		"duplicate":        base + "\npath=octo/repo.git",
		"conflicting":      base + "\nhost=example.com",
		"unknown":          base + "\nusername=someone",
		"url":              "url=https://github.com/octo/repo.git",
		"http":             strings.Replace(base, "https", "http", 1),
		"host case":        strings.Replace(base, "github.com", "GitHub.com", 1),
		"host port":        strings.Replace(base, "github.com", "github.com:444", 1),
		"userinfo":         strings.Replace(base, "github.com", "user@github.com", 1),
		"leading slash":    strings.Replace(base, "octo/repo.git", "/octo/repo.git", 1),
		"trailing slash":   strings.Replace(base, "octo/repo.git", "octo/repo.git/", 1),
		"three segments":   strings.Replace(base, "octo/repo.git", "octo/team/repo.git", 1),
		"dot owner":        strings.Replace(base, "octo/repo.git", "./repo.git", 1),
		"dot repo":         strings.Replace(base, "octo/repo.git", "octo/...git", 1),
		"empty segment":    strings.Replace(base, "octo/repo.git", "octo//repo.git", 1),
		"uppercase owner":  strings.Replace(base, "octo/repo.git", "Octo/repo.git", 1),
		"encoded":          strings.Replace(base, "octo/repo.git", "octo%2frepo.git", 1),
		"query":            strings.Replace(base, "octo/repo.git", "octo/repo.git?x=1", 1),
		"fragment":         strings.Replace(base, "octo/repo.git", "octo/repo.git#x", 1),
		"wrong suffix":     strings.Replace(base, ".git", ".bundle", 1),
		"empty value":      strings.Replace(base, "host=github.com", "host=", 1),
		"missing equals":   strings.Replace(base, "host=github.com", "host", 1),
		"nul":              base + "\x00",
		"cr":               strings.Replace(base, "\n", "\r\n", 1),
		"embedded blank":   base + "\n\npassword=secret",
		"third newline":    base + "\n\n\n",
		"over limit":       strings.Repeat("x", maxInputBytes+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			withClientSeams(t, func(string, string) (string, error) {
				calls++
				return canonicalTestHandle, nil
			}, nil)
			var stdout, stderr strings.Builder
			code := Run([]string{"get"}, strings.NewReader(input), &stdout, &stderr)
			if code != 0 || stdout.String() != failClosed || stderr.Len() != 0 || calls != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q calls=%d", code, stdout.String(), stderr.String(), calls)
			}
		})
	}
}

func TestGetControlFailuresAreFailClosedAndNonLeaking(t *testing.T) {
	input := "protocol=https\nhost=github.com\npath=octo/repo.git\n\n"
	for name, issue := range map[string]func(string, string) (string, error){
		"dependency": func(string, string) (string, error) {
			return "", errors.New("secret lower socket repository token")
		},
		"bad handle": func(string, string) (string, error) { return "cap_bad", nil },
	} {
		t.Run(name, func(t *testing.T) {
			withClientSeams(t, issue, nil)
			var stdout, stderr strings.Builder
			if code := Run([]string{"get"}, strings.NewReader(input), &stdout, &stderr); code != 0 || stdout.String() != failClosed || stderr.Len() != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			for _, secret := range []string{"octo/repo", fixedTestSocket, canonicalTestHandle, "lower", "token"} {
				if strings.Contains(stdout.String()+stderr.String(), secret) {
					t.Fatalf("leaked %q", secret)
				}
			}
		})
	}
}

func TestStoreAndUnknownAreBoundedSilentNoOps(t *testing.T) {
	for _, operation := range []string{"store", "future-operation"} {
		t.Run(operation, func(t *testing.T) {
			reader := &endlessReader{}
			calls := 0
			withClientSeams(t, func(string, string) (string, error) { calls++; return "", nil }, func(string, string) error { calls++; return nil })
			var stdout, stderr strings.Builder
			if code := Run([]string{operation}, reader, &stdout, &stderr); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 || calls != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q calls=%d", code, stdout.String(), stderr.String(), calls)
			}
			if reader.bytes != maxInputBytes+1 {
				t.Fatalf("read bytes=%d", reader.bytes)
			}
		})
	}
}

func TestEraseSuccessAndStrictInput(t *testing.T) {
	called := 0
	withClientSeams(t, nil, func(path, handle string) error {
		called++
		if path != fixedTestSocket || handle != canonicalTestHandle {
			t.Fatal("unexpected revoke values")
		}
		return nil
	})
	var stdout, stderr strings.Builder
	input := "password=" + canonicalTestHandle + "\n\n"
	if code := Run([]string{"erase"}, strings.NewReader(input), &stdout, &stderr); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 || called != 1 {
		t.Fatalf("code=%d stdout=%q stderr=%q called=%d", code, stdout.String(), stderr.String(), called)
	}
}

func TestEraseRejectsWithoutOutputOrDial(t *testing.T) {
	cases := map[string]string{
		"empty":      "",
		"missing":    "username=x-access-token\n",
		"bad handle": "password=cap_bad\n",
		"duplicate":  "password=" + canonicalTestHandle + "\npassword=" + canonicalTestHandle,
		"mixed":      "password=" + canonicalTestHandle + "\nprotocol=https",
		"extra":      "password=" + canonicalTestHandle + "\n\nextra=x",
		"cr":         "password=" + canonicalTestHandle + "\r\n",
		"nul":        "password=" + canonicalTestHandle + "\x00",
		"over limit": strings.Repeat("x", maxInputBytes+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			withClientSeams(t, nil, func(string, string) error { calls++; return nil })
			var stdout, stderr strings.Builder
			if code := Run([]string{"erase"}, strings.NewReader(input), &stdout, &stderr); code != 1 || stdout.Len() != 0 || stderr.Len() != 0 || calls != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q calls=%d", code, stdout.String(), stderr.String(), calls)
			}
		})
	}
}

func TestEraseDependencyFailureIsSilent(t *testing.T) {
	withClientSeams(t, nil, func(string, string) error { return errors.New("secret handle lower error") })
	var stdout, stderr strings.Builder
	if code := Run([]string{"erase"}, strings.NewReader("password="+canonicalTestHandle), &stdout, &stderr); code != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestArgumentHelpAndVersionSurface(t *testing.T) {
	originalVersion := command.Version
	command.Version = "test-version"
	defer func() { command.Version = originalVersion }()
	for _, input := range []struct {
		args    []string
		code    int
		wantOut string
		wantErr string
	}{
		{[]string{"--version"}, 0, programName + " test-version\n", ""},
		{[]string{"--help"}, 0, "usage: " + programName + " <get|store|erase>\n", ""},
		{[]string{"-h"}, 0, "usage: " + programName + " <get|store|erase>\n", ""},
		{nil, 2, "", usageDiagnostic},
		{[]string{"get", "extra-secret"}, 2, "", usageDiagnostic},
	} {
		var stdout, stderr strings.Builder
		code := Run(input.args, strings.NewReader("password=secret"), &stdout, &stderr)
		if code != input.code || stdout.String() != input.wantOut || stderr.String() != input.wantErr || strings.Contains(stderr.String(), "extra-secret") {
			t.Fatalf("args=%q code=%d stdout=%q stderr=%q", input.args, code, stdout.String(), stderr.String())
		}
	}
}

func TestSocketIsLinkFixedAndEnvironmentDoesNotOverride(t *testing.T) {
	t.Setenv("DEV_AGENT_EGRESS_SOCKET", "/tmp/attacker.sock")
	t.Setenv("GIT_CREDENTIAL_SOCKET", "/tmp/attacker-two.sock")
	originalSocket := socketPath
	socketPath = "relative.sock"
	defer func() { socketPath = originalSocket }()
	calls := 0
	originalIssue := issueCapability
	issueCapability = func(string, string) (string, error) { calls++; return canonicalTestHandle, nil }
	defer func() { issueCapability = originalIssue }()
	var stdout, stderr strings.Builder
	input := "protocol=https\nhost=github.com\npath=octo/repo.git"
	if code := Run([]string{"get"}, strings.NewReader(input), &stdout, &stderr); code != 0 || stdout.String() != failClosed || stderr.Len() != 0 || calls != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q calls=%d", code, stdout.String(), stderr.String(), calls)
	}
}

func TestSuccessOutputHandlesPartialWrites(t *testing.T) {
	withClientSeams(t, func(string, string) (string, error) { return canonicalTestHandle, nil }, nil)
	writer := &shortWriter{limit: 3}
	input := "protocol=https\nhost=github.com\npath=octo/repo.git"
	if code := Run([]string{"get"}, strings.NewReader(input), writer, io.Discard); code != 0 {
		t.Fatalf("code=%d", code)
	}
	want := "username=x-access-token\npassword=" + canonicalTestHandle + "\n\n"
	if writer.builder.String() != want {
		t.Fatalf("output=%q", writer.builder.String())
	}
}

func withClientSeams(t *testing.T, issue func(string, string) (string, error), revoke func(string, string) error) {
	t.Helper()
	originalSocket, originalIssue, originalRevoke := socketPath, issueCapability, revokeCapability
	socketPath = fixedTestSocket
	if issue != nil {
		issueCapability = issue
	}
	if revoke != nil {
		revokeCapability = revoke
	}
	t.Cleanup(func() {
		socketPath, issueCapability, revokeCapability = originalSocket, originalIssue, originalRevoke
	})
}
