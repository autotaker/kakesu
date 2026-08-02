package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/launchsession"
)

func TestLauncherExactRunAndStdio(t *testing.T) {
	original := runLaunchSession
	defer func() { runLaunchSession = original }()
	var gotContext context.Context
	var gotRequest launchsession.Request
	runLaunchSession = func(ctx context.Context, request launchsession.Request) launchsession.Result {
		gotContext, gotRequest = ctx, request
		return launchsession.Result{}
	}
	ctx := context.WithValue(context.Background(), struct{}{}, "context sentinel")
	stdin := strings.NewReader("stdin sentinel")
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	args := []string{"run", "--repository", "octo/repo", "--", "agent", "argument with spaces", "", "--repository"}
	if code := RunContextIO(ctx, "dev-agent-launcher", args, stdin, stdout, stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if gotContext != ctx || gotRequest.Repository != "octo/repo" || !reflect.DeepEqual(gotRequest.Argv, args[4:]) ||
		gotRequest.Stdin != stdin || gotRequest.Stdout != stdout || gotRequest.Stderr != stderr || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("handoff changed ctx/request=%#v stdout=%q stderr=%q", gotRequest, stdout.String(), stderr.String())
	}
	args[4] = "mutated"
	if gotRequest.Argv[0] != "agent" {
		t.Fatal("launcher retained caller argv backing array")
	}
}

func TestLauncherRejectsNonExactArgumentsBeforeSession(t *testing.T) {
	original := runLaunchSession
	defer func() { runLaunchSession = original }()
	calls := 0
	runLaunchSession = func(context.Context, launchsession.Request) launchsession.Result {
		calls++
		return launchsession.Result{}
	}
	cases := [][]string{
		nil,
		{"run"},
		{"run", "--repository", "octo/repo", "--"},
		{"run", "--repository", "octo/repo", "agent"},
		{"run", "--repo", "octo/repo", "--", "agent"},
		{"run", "octo/repo", "--repository", "--", "agent"},
		{"run", "--repository", "octo/repo", "--repository", "octo/other", "--", "agent"},
		{"run", "--repository", "Octo/repo", "--", "agent"},
		{"run", "--repository", "octo//repo", "--", "agent"},
		{"run", "--repository", "octo/repo", "--", ""},
		{"run", "--repository", "octo/repo", "--", "a\x00b"},
		{"run", "--repository", "octo/repo", "--", "agent", "x\x00y"},
		{"--help", "extra"},
		{"--version", "extra"},
		{"unknown", "--repository", "octo/repo", "--", "agent"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		if code := RunContextIO(context.Background(), "dev-agent-launcher", args, strings.NewReader(""), &stdout, &stderr); code != 2 || stdout.Len() != 0 || stderr.String() != "dev-agent-launcher: invalid arguments\n" {
			t.Fatalf("args=%q code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
	if calls != 0 {
		t.Fatalf("session called %d times for usage errors", calls)
	}
}

func TestLauncherHelpVersionAndFixedSessionFailure(t *testing.T) {
	original := runLaunchSession
	defer func() { runLaunchSession = original }()
	for _, argument := range []string{"--help", "-h"} {
		var stdout, stderr bytes.Buffer
		if code := RunContextIO(context.Background(), "dev-agent-launcher", []string{argument}, nil, &stdout, &stderr); code != 0 || stdout.String() != "usage: dev-agent-launcher run --repository owner/repo -- COMMAND [ARG...]\n" || stderr.Len() != 0 {
			t.Fatalf("help %q code=%d stdout=%q stderr=%q", argument, code, stdout.String(), stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := RunContextIO(context.Background(), "dev-agent-launcher", []string{"--version"}, nil, &stdout, &stderr); code != 0 || !strings.HasPrefix(stdout.String(), "dev-agent-launcher ") || stderr.Len() != 0 {
		t.Fatalf("version code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	protected := []string{"opaque-handle", "octo/repo", testSocketDiagnostic, "secret-command", "lower error"}
	runLaunchSession = func(context.Context, launchsession.Request) launchsession.Result {
		return launchsession.Result{ExitCode: launchsession.FailureExitCode, SessionFailed: true}
	}
	stdout.Reset()
	stderr.Reset()
	if code := RunContextIO(context.Background(), "dev-agent-launcher", []string{"run", "--repository", "octo/repo", "--", "secret-command"}, nil, &stdout, &stderr); code != launchsession.FailureExitCode || stdout.Len() != 0 || stderr.String() != "dev-agent-launcher: session failed\n" {
		t.Fatalf("failure code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, secret := range protected {
		if strings.Contains(stderr.String(), secret) {
			t.Fatalf("diagnostic leaked %q: %q", secret, stderr.String())
		}
	}

	runLaunchSession = func(context.Context, launchsession.Request) launchsession.Result {
		return launchsession.Result{ExitCode: 37}
	}
	stdout.Reset()
	stderr.Reset()
	if code := RunContextIO(context.Background(), "dev-agent-launcher", []string{"run", "--repository", "octo/repo", "--", "agent"}, nil, &stdout, &stderr); code != 37 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("child exit code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	runLaunchSession = func(context.Context, launchsession.Request) launchsession.Result {
		return launchsession.Result{ExitCode: launchsession.FailureExitCode}
	}
	stdout.Reset()
	stderr.Reset()
	if code := RunContextIO(context.Background(), "dev-agent-launcher", []string{"run", "--repository", "octo/repo", "--", "agent"}, nil, &stdout, &stderr); code != launchsession.FailureExitCode || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("ordinary child 125 code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

const testSocketDiagnostic = "/run/private/control.sock"

func TestEgressServeArgumentsAndFixedFailure(t *testing.T) {
	old := runEgressService
	defer func() { runEgressService = old }()
	var gotContext context.Context
	var gotPath string
	runEgressService = func(ctx context.Context, path string) error {
		gotContext, gotPath = ctx, path
		return errors.New("secret path and cause")
	}
	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := RunContext(ctx, "dev-agent-egress", []string{"serve", "--config", "/etc/secret.json"}, &stdout, &stderr); code != 1 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if gotContext != ctx || gotPath != "/etc/secret.json" || stdout.Len() != 0 || stderr.String() != "dev-agent-egress: service start failed\n" {
		t.Fatalf("context/path/output mismatch ctx=%v path=%q stdout=%q stderr=%q", gotContext, gotPath, stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "secret") || strings.Contains(stderr.String(), "cause") {
		t.Fatalf("diagnostic leaked detail: %q", stderr.String())
	}
}

func TestEgressServePassesCanceledContextAsCleanSuccess(t *testing.T) {
	old := runEgressService
	defer func() { runEgressService = old }()
	var got context.Context
	runEgressService = func(ctx context.Context, _ string) error { got = ctx; return nil }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr bytes.Buffer
	if code := RunContext(ctx, "dev-agent-egress", []string{"serve", "--config", "/etc/harness.json"}, &stdout, &stderr); code != 0 || got != ctx || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d got=%v stdout=%q stderr=%q", code, got, stdout.String(), stderr.String())
	}
}

func TestEgressServeRejectsNonExactArguments(t *testing.T) {
	old := runEgressService
	defer func() { runEgressService = old }()
	runEgressService = func(context.Context, string) error { t.Fatal("service called for invalid args"); return nil }
	for _, args := range [][]string{{"serve"}, {"serve", "--config"}, {"serve", "--config", ""}, {"serve", "--other", "/tmp/config"}, {"start", "--config", "/tmp/config"}} {
		var stdout, stderr bytes.Buffer
		if code := RunContext(context.Background(), "dev-agent-egress", args, &stdout, &stderr); code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "invalid serve arguments") {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
}

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
	input := `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"identity":{"workspace_id":"workspace-1"},"network":{"default":"deny"},"egress":{"github_repositories":["octo/repo"],"openai_models":["gpt-4o-mini"]}}`
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
	if strings.Contains(stdout.String(), path) || strings.Contains(stdout.String(), "dev-runtime") || strings.Contains(stdout.String(), "dev-broker") {
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

func TestSetupPlanProvision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "harness.json")
	root := filepath.Join(dir, "target")
	input := `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"identity":{"workspace_id":"workspace-1"},"network":{"default":"deny"},"egress":{"github_repositories":["octo/repo"],"openai_models":["gpt-4o-mini"]}}`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run("dev-agent-harness-setup", []string{"plan-provision", "--config", path, "--target-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("plan-provision returned %d: %q", code, stderr.String())
	}
	if stderr.Len() != 0 || len(strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")) != 12 {
		t.Fatalf("unexpected output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"kind":"manifest"`) || !strings.Contains(stdout.String(), `"action_count":11`) {
		t.Fatal("manifest header missing")
	}
	if strings.Contains(stdout.String(), path) || stderr.Len() != 0 {
		t.Fatal("success output leaked diagnostics or input")
	}
}

func TestSetupPlanProvisionRejectsArgumentsAndInputs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "harness.json")
	root := filepath.Join(dir, "target")
	input := `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"identity":{"workspace_id":"workspace-1"},"network":{"default":"deny"},"egress":{"github_repositories":["octo/repo"],"openai_models":["gpt-4o-mini"]}}`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	cases := [][]string{
		{"plan-provision"},
		{"plan-provision", "--config", path},
		{"plan-provision", "--target-root", root, "--config", path},
		{"plan-provision", "--config", path, "--target-root", root, "extra"},
		{"plan-provision", "--config", path, "--target-root", "relative"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run("dev-agent-harness-setup", args, &stdout, &stderr); code == 0 || stdout.Len() != 0 {
				t.Fatalf("accepted invalid args code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if strings.Contains(stderr.String(), path) {
				t.Fatalf("stderr leaked input: %q", stderr.String())
			}
		})
	}
	if err := os.WriteFile(path, []byte(strings.Replace(input, `"deny"`, `"allow"`, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run("dev-agent-harness-setup", []string{"plan-provision", "--config", path, "--target-root", root}, &stdout, &stderr); code == 0 || stdout.Len() != 0 {
		t.Fatalf("accepted invalid config code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), path) || strings.Contains(stderr.String(), "allow") {
		t.Fatalf("stderr leaked invalid config: %q", stderr.String())
	}
}

type failingOutput struct {
	calls int
}

func (w *failingOutput) Write(p []byte) (int, error) {
	w.calls++
	return len(p) / 2, os.ErrClosed
}

func TestPlanProvisionWriterFailureIsNotRetried(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "harness.json")
	input := `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"identity":{"workspace_id":"workspace-1"},"network":{"default":"deny"},"egress":{"github_repositories":["octo/repo"],"openai_models":["gpt-4o-mini"]}}`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	w := &failingOutput{}
	var stderr bytes.Buffer
	if code := Run("dev-agent-harness-setup", []string{"plan-provision", "--config", path, "--target-root", filepath.Join(dir, "root")}, w, &stderr); code == 0 || w.calls != 1 {
		t.Fatalf("writer result code=%d calls=%d stderr=%q", code, w.calls, stderr.String())
	}
}

func TestSetupVerifyProvision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "harness.json")
	manifest := filepath.Join(dir, "manifest.jsonl")
	root := filepath.Join(dir, "target")
	input := `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"identity":{"workspace_id":"workspace-1"},"network":{"default":"deny"},"egress":{"github_repositories":["octo/repo"],"openai_models":["gpt-4o-mini"]}}`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	var planned bytes.Buffer
	if code := Run("dev-agent-harness-setup", []string{"plan-provision", "--config", path, "--target-root", root}, &planned, new(bytes.Buffer)); code != 0 {
		t.Fatalf("plan-provision returned %d", code)
	}
	if err := os.WriteFile(manifest, planned.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run("dev-agent-harness-setup", []string{"verify-provision", "--config", path, "--manifest", manifest, "--target-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("verify-provision returned %d: %q", code, stderr.String())
	}
	if stdout.String() != "provision manifest version=1 actions=11 verified\n" || stderr.Len() != 0 {
		t.Fatalf("unexpected output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), path) || strings.Contains(stdout.String(), "dev-agent") {
		t.Fatal("success summary leaked input values")
	}
	if err := os.WriteFile(manifest, append(planned.Bytes(), ' '), 0600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run("dev-agent-harness-setup", []string{"verify-provision", "--config", path, "--manifest", manifest, "--target-root", root}, &stdout, &stderr); code == 0 || stdout.Len() != 0 {
		t.Fatalf("accepted noncanonical manifest code=%d stdout=%q", code, stdout.String())
	}
	if strings.Contains(stderr.String(), path) || strings.Contains(stderr.String(), "dev-runtime") || strings.Contains(stderr.String(), "dev-broker") || strings.Contains(stderr.String(), "replacement") {
		t.Fatalf("diagnostic leaked input: %q", stderr.String())
	}
}

func TestSetupVerifyProvisionArgumentsAndFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "harness.json")
	manifest := filepath.Join(dir, "manifest.jsonl")
	root := filepath.Join(dir, "target")
	input := `{"version":1,"paths":{"config_dir":"/etc/dev-agent","state_dir":"/var/lib/dev-agent","runtime_dir":"/run/dev-agent"},"users":{"agent":"dev-agent","runtime":"dev-runtime","broker":"dev-broker"},"identity":{"workspace_id":"workspace-1"},"network":{"default":"deny"},"egress":{"github_repositories":["octo/repo"],"openai_models":["gpt-4o-mini"]}}`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte("manifest sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"verify-provision"},
		{"verify-provision", "--config", path},
		{"verify-provision", "--manifest", manifest, "--config", path, "--target-root", root},
		{"verify-provision", "--config", path, "--manifest", manifest, "--target-root", root, "extra"},
		{"verify-provision", "--config", path, "--manifest", manifest, "--target-root", "relative"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run("dev-agent-harness-setup", args, &stdout, &stderr); code == 0 || stdout.Len() != 0 {
			t.Fatalf("accepted invalid args=%v code=%d stdout=%q", args, code, stdout.String())
		}
		if strings.Contains(stderr.String(), path) || strings.Contains(stderr.String(), manifest) {
			t.Fatalf("diagnostic leaked path for args=%v: %q", args, stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := Run("dev-agent-harness-setup", []string{"verify-provision", "--config", path, "--manifest", manifest, "--target-root", root}, &stdout, &stderr); code == 0 || stdout.Len() != 0 {
		t.Fatalf("accepted invalid manifest code=%d stdout=%q", code, stdout.String())
	}
	if strings.Contains(stderr.String(), path) || strings.Contains(stderr.String(), manifest) || strings.Contains(stderr.String(), "manifest sentinel") {
		t.Fatalf("diagnostic leaked input: %q", stderr.String())
	}
	if err := os.WriteFile(path, []byte(strings.Replace(input, `"deny"`, `"allow"`, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run("dev-agent-harness-setup", []string{"verify-provision", "--config", path, "--manifest", manifest, "--target-root", root}, &stdout, &stderr); code == 0 || stdout.Len() != 0 {
		t.Fatalf("accepted invalid config code=%d stdout=%q", code, stdout.String())
	}
	if strings.Contains(stderr.String(), path) || strings.Contains(stderr.String(), "allow") {
		t.Fatalf("diagnostic leaked config: %q", stderr.String())
	}
}
