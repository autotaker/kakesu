package launchsession

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testSocket = "/run/dev-agent-harness/egress.sock"
	testHelper = "/usr/local/bin/git-credential-dev-agent"
	testCAPath = "/tmp/dev-agent-session-test/proxy-ca.pem"
)

type fakeInfo struct {
	name string
	mode os.FileMode
}

func (i fakeInfo) Name() string       { return i.name }
func (i fakeInfo) Size() int64        { return 10 }
func (i fakeInfo) Mode() os.FileMode  { return i.mode }
func (i fakeInfo) ModTime() time.Time { return time.Time{} }
func (i fakeInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fakeInfo) Sys() any           { return nil }

type fakeFile struct {
	writes   int
	contents []byte
	writeErr error
	closeErr error
	chmodErr error
}

func (f *fakeFile) Write(contents []byte) (int, error) {
	f.writes++
	if f.writeErr != nil {
		return len(contents) / 2, f.writeErr
	}
	f.contents = append(f.contents, contents...)
	return len(contents), nil
}
func (f *fakeFile) Close() error               { return f.closeErr }
func (f *fakeFile) Chmod(os.FileMode) error    { return f.chmodErr }
func (f *fakeFile) Stat() (os.FileInfo, error) { return fakeInfo{name: caBasename, mode: 0600}, nil }

type fakeBridge struct {
	started chan struct{}
	returnC chan error
	once    sync.Once
}

type drainingBridge struct{ canceled, release chan struct{} }

func (b drainingBridge) Serve(ctx context.Context) error {
	<-ctx.Done()
	close(b.canceled)
	<-b.release
	return nil
}

func (b *fakeBridge) Serve(ctx context.Context) error {
	b.once.Do(func() { close(b.started) })
	select {
	case err := <-b.returnC:
		return err
	case <-ctx.Done():
		return nil
	}
}

type fakeChild struct {
	startErr error
	waitC    chan childResult
	startC   chan struct{}
	stopC    chan struct{}
	started  int
	stopped  int
	mu       sync.Mutex
}

func (p *fakeChild) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.started++
	if p.startC != nil {
		close(p.startC)
		p.startC = nil
	}
	return p.startErr
}
func (p *fakeChild) Wait() (int, bool) {
	result := <-p.waitC
	return result.code, result.ordinary
}
func (p *fakeChild) Stop() {
	p.mu.Lock()
	p.stopped++
	if p.stopC != nil {
		close(p.stopC)
		p.stopC = nil
	}
	p.mu.Unlock()
}

type fixture struct {
	deps        dependencies
	calls       []string
	revokes     []string
	file        *fakeFile
	bridge      *fakeBridge
	child       *fakeChild
	argv        []string
	environment []string
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	removeErr   error
	removeCalls int
}

func newFixture() *fixture {
	f := &fixture{
		file:   &fakeFile{},
		bridge: &fakeBridge{started: make(chan struct{}), returnC: make(chan error, 1)},
		child:  &fakeChild{waitC: make(chan childResult, 1), startC: make(chan struct{}), stopC: make(chan struct{})},
	}
	f.deps = dependencies{
		control: controlOperations{
			proxyCA: func(socket string) ([]byte, error) {
				f.calls = append(f.calls, "ca:"+socket)
				return []byte("public ca"), nil
			},
			issueGitHub: func(socket, repository string) (string, error) {
				f.calls = append(f.calls, "github:"+socket+":"+repository)
				return "github-handle", nil
			},
			issueOpenAI: func(socket string) (string, error) {
				f.calls = append(f.calls, "openai:"+socket)
				return "openai-handle", nil
			},
			revoke: func(_ string, handle string) error {
				f.revokes = append(f.revokes, handle)
				return nil
			},
		},
		newBridge: func(socket string) (bridge, string, error) {
			f.calls = append(f.calls, "bridge:"+socket)
			return f.bridge, "http://127.0.0.1:43210", nil
		},
		newProcess: func(argv, environment []string, stdin io.Reader, stdout, stderr io.Writer) childProcess {
			f.calls = append(f.calls, "process")
			f.argv = append([]string(nil), argv...)
			f.environment = append([]string(nil), environment...)
			f.stdin, f.stdout, f.stderr = stdin, stdout, stderr
			return f.child
		},
		mkdirTemp: func(parent, pattern string) (string, error) {
			f.calls = append(f.calls, "mkdir:"+parent+":"+pattern)
			return "/tmp/dev-agent-session-test", nil
		},
		chmod: func(path string, mode os.FileMode) error {
			f.calls = append(f.calls, "chmod:"+path+":"+mode.String())
			return nil
		},
		openCA: func(path string) (caFile, error) {
			f.calls = append(f.calls, "open:"+path)
			return f.file, nil
		},
		lstat: func(path string) (os.FileInfo, error) {
			if path == "/tmp/dev-agent-session-test" {
				return fakeInfo{name: "dev-agent-session-test", mode: os.ModeDir | 0700}, nil
			}
			return fakeInfo{name: caBasename, mode: 0600}, nil
		},
		removeAll: func(path string) error {
			f.calls = append(f.calls, "remove:"+path)
			f.removeCalls++
			return f.removeErr
		},
	}
	return f
}

func request() Request {
	return Request{Repository: "octo/repo", Argv: []string{"agent", "argument with spaces", ""}, Stdin: strings.NewReader("input"), Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)}
}

func TestRunSuccessPreservesProcessSpecAndCleansUp(t *testing.T) {
	f := newFixture()
	r := request()
	f.child.waitC <- childResult{code: 0, ordinary: true}
	got := run(context.Background(), r, f.deps, testSocket, testHelper, []string{
		"PATH=/usr/bin", "HOME=/home/agent", "TERM=xterm", "LANG=C.UTF-8", "LC_ALL=C", "CODEX_HOME=/home/agent/.codex",
		"GH_TOKEN=parent-secret", "OPENAI_API_KEY=parent-secret", "HTTPS_PROXY=http://host", "TMPDIR=/hostile", "GIT_CONFIG_COUNT=99", "GIT_CONFIG_NOSYSTEM=0", "GIT_CONFIG_GLOBAL=/hostile", "LD_PRELOAD=evil", "OTHER=value",
	})
	if got.ExitCode != 0 || got.SessionFailed {
		t.Fatalf("result=%+v calls=%v", got, f.calls)
	}
	if !reflect.DeepEqual(f.argv, r.Argv) || f.stdin != r.Stdin || f.stdout != r.Stdout || f.stderr != r.Stderr {
		t.Fatalf("process spec changed argv=%q", f.argv)
	}
	if f.file.writes != 1 || string(f.file.contents) != "public ca" || f.removeCalls != 1 {
		t.Fatalf("CA lifecycle writes=%d contents=%q removes=%d", f.file.writes, f.file.contents, f.removeCalls)
	}
	if !reflect.DeepEqual(f.revokes, []string{"openai-handle", "github-handle"}) {
		t.Fatalf("revokes=%v", f.revokes)
	}
	wantOrder := []string{"ca:" + testSocket, "github:" + testSocket + ":octo/repo", "openai:" + testSocket, "mkdir:/tmp:" + caPattern}
	if len(f.calls) < len(wantOrder) || !reflect.DeepEqual(f.calls[:len(wantOrder)], wantOrder) {
		t.Fatalf("initialization order=%v", f.calls)
	}
	assertEnvironment(t, f.environment)
}

func assertEnvironment(t *testing.T, environment []string) {
	t.Helper()
	values := make(map[string]string)
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || values[key] != "" || hasKey(values, key) {
			t.Fatalf("malformed/duplicate environment entry %q in %q", entry, environment)
		}
		values[key] = value
	}
	for _, key := range []string{"GH_TOKEN", "OPENAI_API_KEY", "HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "SSL_CERT_FILE", "CURL_CA_BUNDLE", "REQUESTS_CA_BUNDLE", "NODE_EXTRA_CA_CERTS", "GIT_SSL_CAINFO", "GIT_TERMINAL_PROMPT", "GIT_CONFIG_NOSYSTEM", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_COUNT"} {
		if !hasKey(values, key) {
			t.Fatalf("missing %s in %q", key, environment)
		}
	}
	if values["GH_TOKEN"] != "github-handle" || values["OPENAI_API_KEY"] != "openai-handle" || values["HTTPS_PROXY"] != "http://127.0.0.1:43210" || values["SSL_CERT_FILE"] != testCAPath || values["GIT_CONFIG_NOSYSTEM"] != "1" || values["GIT_CONFIG_GLOBAL"] != "/dev/null" {
		t.Fatalf("incorrect session environment %v", values)
	}
	for _, key := range []string{"LD_PRELOAD", "OTHER"} {
		if hasKey(values, key) {
			t.Fatalf("inherited hostile key %s", key)
		}
	}
	wantConfig := [][2]string{{"credential.helper", ""}, {"credential.helper", testHelper}, {"credential.https://github.com.useHttpPath", "true"}, {"http.proxy", "http://127.0.0.1:43210"}, {"http.sslCAInfo", testCAPath}, {"credential.interactive", "false"}, {"core.askPass", "/bin/false"}}
	if values["GIT_CONFIG_COUNT"] != "7" {
		t.Fatalf("config count=%q", values["GIT_CONFIG_COUNT"])
	}
	for index, pair := range wantConfig {
		if values["GIT_CONFIG_KEY_"+itoa(index)] != pair[0] || values["GIT_CONFIG_VALUE_"+itoa(index)] != pair[1] {
			t.Fatalf("config[%d]=(%q,%q)", index, values["GIT_CONFIG_KEY_"+itoa(index)], values["GIT_CONFIG_VALUE_"+itoa(index)])
		}
	}
}

func hasKey(values map[string]string, key string) bool {
	_, ok := values[key]
	return ok
}

func TestRunRejectsBeforeControl(t *testing.T) {
	for name, mutate := range map[string]func(*Request, *string, *string){
		"repository":      func(r *Request, _, _ *string) { r.Repository = "Octo/repo" },
		"empty command":   func(r *Request, _, _ *string) { r.Argv[0] = "" },
		"nul":             func(r *Request, _, _ *string) { r.Argv[0] = "a\x00b" },
		"relative socket": func(_ *Request, socket, _ *string) { *socket = "run/socket" },
		"unclean helper":  func(_ *Request, _, helper *string) { *helper = "/usr/bin/../bin/helper" },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture()
			r, socket, helper := request(), testSocket, testHelper
			mutate(&r, &socket, &helper)
			if result := run(context.Background(), r, f.deps, socket, helper, nil); result != failureResult() || len(f.calls) != 0 {
				t.Fatalf("result=%+v calls=%v", result, f.calls)
			}
		})
	}
}

func TestPartialAcquisitionStopsAndRevokesOwnedHandles(t *testing.T) {
	for _, test := range []struct {
		name        string
		fail        string
		wantCalls   int
		wantRevokes []string
	}{
		{name: "ca", fail: "ca", wantCalls: 1},
		{name: "github", fail: "github", wantCalls: 2},
		{name: "openai", fail: "openai", wantCalls: 3, wantRevokes: []string{"github-handle"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture()
			switch test.fail {
			case "ca":
				f.deps.control.proxyCA = func(string) ([]byte, error) { f.calls = append(f.calls, "ca"); return nil, errors.New("secret") }
			case "github":
				f.deps.control.issueGitHub = func(string, string) (string, error) {
					f.calls = append(f.calls, "github")
					return "", errors.New("secret")
				}
			case "openai":
				f.deps.control.issueOpenAI = func(string) (string, error) { f.calls = append(f.calls, "openai"); return "", errors.New("secret") }
			}
			if result := run(context.Background(), request(), f.deps, testSocket, testHelper, nil); result != failureResult() || len(f.calls) != test.wantCalls || !reflect.DeepEqual(f.revokes, test.wantRevokes) {
				t.Fatalf("result=%+v calls=%v revokes=%v", result, f.calls, f.revokes)
			}
		})
	}
}

func TestSetupAndStartFailuresCleanExactlyOnce(t *testing.T) {
	for name, breakIt := range map[string]func(*fixture){
		"mkdir": func(f *fixture) {
			f.deps.mkdirTemp = func(string, string) (string, error) { return "", errors.New("path secret") }
		},
		"write":      func(f *fixture) { f.file.writeErr = errors.New("CA secret") },
		"close":      func(f *fixture) { f.file.closeErr = errors.New("CA secret") },
		"chmod file": func(f *fixture) { f.file.chmodErr = errors.New("CA secret") },
		"existing CA": func(f *fixture) {
			f.deps.openCA = func(string) (caFile, error) { return nil, os.ErrExist }
		},
		"directory symlink": func(f *fixture) {
			f.deps.lstat = func(string) (os.FileInfo, error) { return fakeInfo{mode: os.ModeSymlink | 0700}, nil }
		},
		"file symlink": func(f *fixture) {
			old := f.deps.lstat
			f.deps.lstat = func(path string) (os.FileInfo, error) {
				if path == testCAPath {
					return fakeInfo{mode: os.ModeSymlink | 0600}, nil
				}
				return old(path)
			}
		},
		"file nonregular": func(f *fixture) {
			old := f.deps.lstat
			f.deps.lstat = func(path string) (os.FileInfo, error) {
				if path == testCAPath {
					return fakeInfo{mode: os.ModeNamedPipe | 0600}, nil
				}
				return old(path)
			}
		},
		"file mode": func(f *fixture) {
			old := f.deps.lstat
			f.deps.lstat = func(path string) (os.FileInfo, error) {
				if path == testCAPath {
					return fakeInfo{name: caBasename, mode: 0644}, nil
				}
				return old(path)
			}
		},
		"bridge": func(f *fixture) {
			f.deps.newBridge = func(string) (bridge, string, error) { return nil, "", errors.New("socket secret") }
		},
		"endpoint": func(f *fixture) {
			f.deps.newBridge = func(string) (bridge, string, error) { return f.bridge, "http://0.0.0.0:1", nil }
		},
		"child start": func(f *fixture) { f.child.startErr = errors.New("command secret") },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture()
			breakIt(f)
			if result := run(context.Background(), request(), f.deps, testSocket, testHelper, nil); result != failureResult() {
				t.Fatalf("result=%+v", result)
			}
			if !reflect.DeepEqual(f.revokes, []string{"openai-handle", "github-handle"}) {
				t.Fatalf("revokes=%v", f.revokes)
			}
			if name != "mkdir" && f.removeCalls != 1 {
				t.Fatalf("remove calls=%d", f.removeCalls)
			}
		})
	}
}

func TestChildExitWaitsForBridgeDrainBeforeCleanup(t *testing.T) {
	f := newFixture()
	canceled, release := make(chan struct{}), make(chan struct{})
	f.deps.newBridge = func(string) (bridge, string, error) {
		return drainingBridge{canceled, release}, "http://127.0.0.1:43210", nil
	}
	f.child.waitC <- childResult{ordinary: true}
	done := make(chan Result, 1)
	go func() {
		done <- run(context.Background(), request(), f.deps, testSocket, testHelper, []string{"TMPDIR=/hostile"})
	}()
	<-canceled
	select {
	case result := <-done:
		t.Fatalf("returned before drain: %+v", result)
	default:
	}
	if f.removeCalls != 0 || len(f.revokes) != 0 {
		t.Fatalf("cleanup before drain removes=%d revokes=%v", f.removeCalls, f.revokes)
	}
	close(release)
	if result := <-done; result != (Result{}) || f.removeCalls != 1 || len(f.revokes) != 2 {
		t.Fatalf("result=%+v removes=%d revokes=%v", result, f.removeCalls, f.revokes)
	}
}

func TestAlreadyCanceledContextDoesNotAcquireOrStart(t *testing.T) {
	f := newFixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result := run(ctx, request(), f.deps, testSocket, testHelper, nil); result != failureResult() || len(f.calls) != 0 {
		t.Fatalf("result=%+v calls=%v", result, f.calls)
	}
}

func TestChildExitCancellationAndBridgeFailure(t *testing.T) {
	t.Run("ordinary nonzero retained", func(t *testing.T) {
		f := newFixture()
		f.child.waitC <- childResult{code: 23, ordinary: true}
		if result := run(context.Background(), request(), f.deps, testSocket, testHelper, nil); result != (Result{ExitCode: 23}) {
			t.Fatalf("result=%+v", result)
		}
	})
	t.Run("signal folded", func(t *testing.T) {
		f := newFixture()
		f.child.waitC <- childResult{code: -1, ordinary: false}
		if result := run(context.Background(), request(), f.deps, testSocket, testHelper, nil); result != failureResult() {
			t.Fatalf("result=%+v", result)
		}
	})
	t.Run("cancel stops and waits", func(t *testing.T) {
		f := newFixture()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan Result, 1)
		startC, stopC := f.child.startC, f.child.stopC
		go func() { done <- run(ctx, request(), f.deps, testSocket, testHelper, nil) }()
		<-startC
		cancel()
		<-stopC
		f.child.waitC <- childResult{ordinary: false}
		if result := <-done; result != failureResult() || f.child.stopped != 1 || f.removeCalls != 1 {
			t.Fatalf("result=%+v stopped=%d removes=%d", result, f.child.stopped, f.removeCalls)
		}
	})
	t.Run("unexpected bridge stops and waits", func(t *testing.T) {
		f := newFixture()
		done := make(chan Result, 1)
		startC, stopC := f.child.startC, f.child.stopC
		go func() { done <- run(context.Background(), request(), f.deps, testSocket, testHelper, nil) }()
		<-startC
		f.bridge.returnC <- errors.New("bridge secret")
		<-stopC
		f.child.waitC <- childResult{ordinary: false}
		if result := <-done; result != failureResult() || f.child.stopped != 1 || f.removeCalls != 1 {
			t.Fatalf("result=%+v stopped=%d removes=%d", result, f.child.stopped, f.removeCalls)
		}
	})
}

func TestLocalCleanupFailureButNotRevokeFailureFoldsResult(t *testing.T) {
	f := newFixture()
	f.removeErr = errors.New("temporary path secret")
	f.child.waitC <- childResult{code: 0, ordinary: true}
	if result := run(context.Background(), request(), f.deps, testSocket, testHelper, nil); result != failureResult() {
		t.Fatalf("cleanup failure result=%+v", result)
	}
	f = newFixture()
	f.deps.control.revoke = func(string, string) error { return errors.New("handle secret") }
	f.child.waitC <- childResult{code: 19, ordinary: true}
	if result := run(context.Background(), request(), f.deps, testSocket, testHelper, nil); result != (Result{ExitCode: 19}) {
		t.Fatalf("revoke failure changed child result=%+v", result)
	}
}

func TestEnvironmentValidationAndEndpointStrictness(t *testing.T) {
	for _, parent := range [][]string{{"PATH=/bin", "PATH=/other"}, {"malformed"}, {"=empty"}, {"LC_bad=value"}} {
		environment, ok := buildEnvironment(parent, "http://127.0.0.1:1234", testCAPath, testHelper, "gh", "oa")
		if len(parent) == 1 && strings.HasPrefix(parent[0], "LC_bad") {
			if !ok || len(environment) == 0 {
				t.Fatal("noncanonical LC key should be omitted, not inherited")
			}
			continue
		}
		if ok || environment != nil {
			t.Fatalf("accepted parent=%q", parent)
		}
	}
	for _, endpoint := range []string{"http://localhost:1", "http://127.0.0.1:0", "http://127.0.0.1:01", "http://127.0.0.1:65536", "http://127.0.0.1:1/path"} {
		if validLoopbackEndpoint(endpoint) {
			t.Fatalf("accepted endpoint=%q", endpoint)
		}
	}
}
