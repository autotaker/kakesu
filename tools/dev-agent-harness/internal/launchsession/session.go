// Package launchsession composes one fail-closed coding-agent session from
// the fixed capability-control socket, loopback bridge, and child process.
package launchsession

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/controlclient"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/proxybridge"
)

const (
	FailureExitCode = 125
	caParent        = "/tmp"
	caPattern       = "dev-agent-session-"
	caBasename      = "proxy-ca.pem"
	bridgeCapacity  = 16
)

// These values are fixed by the build system. They are deliberately not
// flags, configuration, or environment inputs.
var (
	controlSocketPath    string
	credentialHelperPath string
)

// Request contains the only per-run caller input. Routing, providers, models,
// and credential locations are not configurable through this type.
type Request struct {
	Repository string
	Argv       []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

// Result distinguishes an ordinary child status (including 125) from a
// launcher-owned failure that receives the fixed session diagnostic.
type Result struct {
	ExitCode      int
	SessionFailed bool
}

// Run starts one session and returns either the ordinary child exit status or
// FailureExitCode. It intentionally returns no detail-bearing error.
func Run(ctx context.Context, request Request) Result {
	return run(ctx, request, productionDependencies(), controlSocketPath, credentialHelperPath, os.Environ())
}

type controlOperations struct {
	proxyCA     func(string) ([]byte, error)
	issueGitHub func(string, string) (string, error)
	issueOpenAI func(string) (string, error)
	revoke      func(string, string) error
}

type bridge interface {
	Serve(context.Context) error
}

type childProcess interface {
	Start() error
	Wait() (int, bool)
	Stop()
}

type caFile interface {
	io.Writer
	Close() error
	Chmod(os.FileMode) error
	Stat() (os.FileInfo, error)
}

type dependencies struct {
	control    controlOperations
	newBridge  func(string) (bridge, string, error)
	newProcess func([]string, []string, io.Reader, io.Writer, io.Writer) childProcess
	mkdirTemp  func(string, string) (string, error)
	chmod      func(string, os.FileMode) error
	openCA     func(string) (caFile, error)
	lstat      func(string) (os.FileInfo, error)
	removeAll  func(string) error
}

func productionDependencies() dependencies {
	return dependencies{
		control: controlOperations{
			proxyCA:     controlclient.ProxyCA,
			issueGitHub: controlclient.IssueGitHubREST,
			issueOpenAI: controlclient.IssueOpenAI,
			revoke:      controlclient.Revoke,
		},
		newBridge: func(socket string) (bridge, string, error) {
			return proxybridge.New(proxybridge.Rules{UnixSocketPath: socket, MaxConcurrent: bridgeCapacity})
		},
		newProcess: func(argv, environment []string, stdin io.Reader, stdout, stderr io.Writer) childProcess {
			command := exec.Command(argv[0], argv[1:]...)
			command.Env = append([]string(nil), environment...)
			command.Stdin, command.Stdout, command.Stderr = stdin, stdout, stderr
			return &execChild{command: command}
		},
		mkdirTemp: os.MkdirTemp,
		chmod:     os.Chmod,
		openCA: func(path string) (caFile, error) {
			return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		},
		lstat:     os.Lstat,
		removeAll: os.RemoveAll,
	}
}

type execChild struct {
	command  *exec.Cmd
	stopOnce sync.Once
}

func (p *execChild) Start() error { return p.command.Start() }

func (p *execChild) Wait() (int, bool) {
	err := p.command.Wait()
	if err == nil {
		return 0, true
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ProcessState.Exited() {
		return exitError.ExitCode(), true
	}
	return FailureExitCode, false
}

func (p *execChild) Stop() {
	p.stopOnce.Do(func() {
		if p.command.Process != nil {
			_ = p.command.Process.Kill()
		}
	})
}

type childResult struct {
	code     int
	ordinary bool
}

type owner struct {
	deps             dependencies
	socket           string
	githubHandle     string
	openAIHandle     string
	caDirectory      string
	bridgeCancel     context.CancelFunc
	bridgeDone       <-chan error
	localCleanupFail bool
}

func run(ctx context.Context, request Request, deps dependencies, socket, helper string, parentEnvironment []string) Result {
	if ctx == nil || !validRequest(request) || !validFixedPath(socket) || !validFixedPath(helper) || !validDependencies(deps) {
		return failureResult()
	}
	if ctx.Err() != nil {
		return failureResult()
	}

	o := owner{deps: deps, socket: socket}
	ca, err := deps.control.proxyCA(socket)
	if err != nil || len(ca) == 0 {
		return failureResult()
	}
	if ctx.Err() != nil {
		return failureResult()
	}
	o.githubHandle, err = deps.control.issueGitHub(socket, request.Repository)
	if err != nil || o.githubHandle == "" {
		o.cleanup()
		return failureResult()
	}
	if ctx.Err() != nil {
		o.cleanup()
		return failureResult()
	}
	o.openAIHandle, err = deps.control.issueOpenAI(socket)
	if err != nil || o.openAIHandle == "" {
		o.cleanup()
		return failureResult()
	}
	if ctx.Err() != nil {
		o.cleanup()
		return failureResult()
	}

	caPath, err := o.createCA(ca)
	if err != nil {
		o.cleanup()
		return failureResult()
	}
	server, endpoint, err := deps.newBridge(socket)
	if err != nil || server == nil || !validLoopbackEndpoint(endpoint) {
		o.cleanup()
		return failureResult()
	}
	bridgeContext, bridgeCancel := context.WithCancel(context.Background())
	bridgeDone := make(chan error, 1)
	o.bridgeCancel, o.bridgeDone = bridgeCancel, bridgeDone
	go func() { bridgeDone <- server.Serve(bridgeContext) }()
	select {
	case <-bridgeDone:
		o.bridgeDone = nil
		o.cleanup()
		return failureResult()
	default:
	}
	if ctx.Err() != nil {
		o.cleanup()
		return failureResult()
	}

	environment, ok := buildEnvironment(parentEnvironment, endpoint, caPath, helper, o.githubHandle, o.openAIHandle)
	if !ok {
		o.cleanup()
		return failureResult()
	}
	process := deps.newProcess(append([]string(nil), request.Argv...), environment, request.Stdin, request.Stdout, request.Stderr)
	if process == nil || process.Start() != nil {
		o.cleanup()
		return failureResult()
	}

	childDone := make(chan childResult, 1)
	go func() {
		code, ordinary := process.Wait()
		childDone <- childResult{code: code, ordinary: ordinary}
	}()
	result, bridgeConsumed := supervise(ctx, process, childDone, bridgeDone)
	if bridgeConsumed {
		o.bridgeDone = nil
	}
	o.cleanup()
	if o.localCleanupFail || !result.ordinary {
		return failureResult()
	}
	return Result{ExitCode: result.code}
}

func failureResult() Result { return Result{ExitCode: FailureExitCode, SessionFailed: true} }

func supervise(ctx context.Context, process childProcess, childDone <-chan childResult, bridgeDone <-chan error) (childResult, bool) {
	select {
	case result := <-childDone:
		return result, false
	case <-bridgeDone:
		process.Stop()
		<-childDone
		return childResult{code: FailureExitCode}, true
	case <-ctx.Done():
		process.Stop()
		<-childDone
		return childResult{code: FailureExitCode}, false
	}
}

func (o *owner) cleanup() {
	if o.bridgeCancel != nil {
		o.bridgeCancel()
		if o.bridgeDone != nil {
			if err := <-o.bridgeDone; err != nil {
				o.localCleanupFail = true
			}
		}
		o.bridgeCancel, o.bridgeDone = nil, nil
	}
	if o.caDirectory != "" {
		if o.deps.removeAll(o.caDirectory) != nil {
			o.localCleanupFail = true
		}
		o.caDirectory = ""
	}
	if o.openAIHandle != "" {
		_ = o.deps.control.revoke(o.socket, o.openAIHandle)
		o.openAIHandle = ""
	}
	if o.githubHandle != "" {
		_ = o.deps.control.revoke(o.socket, o.githubHandle)
		o.githubHandle = ""
	}
}

func (o *owner) createCA(contents []byte) (string, error) {
	directory, err := o.deps.mkdirTemp(caParent, caPattern)
	if err != nil || !directChild(caParent, directory) {
		return "", errors.New("ca setup failed")
	}
	o.caDirectory = directory
	if o.deps.chmod(directory, 0700) != nil {
		return "", errors.New("ca setup failed")
	}
	directoryInfo, err := o.deps.lstat(directory)
	if err != nil || directoryInfo == nil || !directoryInfo.IsDir() || directoryInfo.Mode().Perm() != 0700 || directoryInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("ca setup failed")
	}
	path := filepath.Join(directory, caBasename)
	file, err := o.deps.openCA(path)
	if err != nil || file == nil {
		return "", errors.New("ca setup failed")
	}
	writeOK := writeAll(file, contents)
	chmodErr := file.Chmod(0600)
	openedInfo, statErr := file.Stat()
	closeErr := file.Close()
	if !writeOK || chmodErr != nil || statErr != nil || openedInfo == nil || !openedInfo.Mode().IsRegular() || openedInfo.Mode().Perm() != 0600 || closeErr != nil {
		return "", errors.New("ca setup failed")
	}
	info, err := o.deps.lstat(path)
	if err != nil || info == nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("ca setup failed")
	}
	return path, nil
}

func writeAll(writer io.Writer, contents []byte) bool {
	for len(contents) > 0 {
		n, err := writer.Write(contents)
		if n > 0 {
			contents = contents[n:]
		}
		if err != nil || n == 0 {
			return false
		}
	}
	return true
}

func buildEnvironment(parent []string, endpoint, caPath, helper, githubHandle, openAIHandle string) ([]string, bool) {
	if !validLoopbackEndpoint(endpoint) || !validFixedPath(caPath) || !validFixedPath(helper) || githubHandle == "" || openAIHandle == "" ||
		strings.ContainsRune(githubHandle, '\x00') || strings.ContainsRune(openAIHandle, '\x00') {
		return nil, false
	}
	allowed := make(map[string]string)
	seen := make(map[string]bool)
	for _, entry := range parent {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" || strings.ContainsRune(entry, '\x00') || seen[key] {
			return nil, false
		}
		seen[key] = true
		if inheritedKey(key) {
			allowed[key] = value
		}
	}
	keys := make([]string, 0, len(allowed))
	for key := range allowed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	environment := make([]string, 0, len(keys)+23)
	for _, key := range keys {
		environment = append(environment, key+"="+allowed[key])
	}
	appendValue := func(key, value string) { environment = append(environment, key+"="+value) }
	appendValue("GH_TOKEN", githubHandle)
	appendValue("OPENAI_API_KEY", openAIHandle)
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		appendValue(key, endpoint)
	}
	for _, key := range []string{"SSL_CERT_FILE", "CURL_CA_BUNDLE", "REQUESTS_CA_BUNDLE", "NODE_EXTRA_CA_CERTS", "GIT_SSL_CAINFO"} {
		appendValue(key, caPath)
	}
	appendValue("GIT_TERMINAL_PROMPT", "0")
	for _, entry := range [][2]string{{"GIT_CONFIG_NOSYSTEM", "1"}, {"GIT_CONFIG_GLOBAL", "/dev/null"}} {
		appendValue(entry[0], entry[1])
	}
	configs := [][2]string{
		{"credential.helper", ""},
		{"credential.helper", helper},
		{"credential.https://github.com.useHttpPath", "true"},
		{"http.proxy", endpoint},
		{"http.sslCAInfo", caPath},
		{"credential.interactive", "false"},
		{"core.askPass", "/bin/false"},
	}
	appendValue("GIT_CONFIG_COUNT", "7")
	for index, config := range configs {
		appendValue("GIT_CONFIG_KEY_"+itoa(index), config[0])
		appendValue("GIT_CONFIG_VALUE_"+itoa(index), config[1])
	}
	return environment, true
}

func inheritedKey(key string) bool {
	if key == "HOME" || key == "PATH" || key == "TERM" || key == "LANG" || key == "CODEX_HOME" {
		return true
	}
	if !strings.HasPrefix(key, "LC_") || len(key) == 3 {
		return false
	}
	for _, char := range key[3:] {
		if char < 'A' || char > 'Z' {
			if char < '0' || char > '9' {
				if char != '_' {
					return false
				}
			}
		}
	}
	return true
}

func validRequest(request Request) bool {
	if !canonicalRepository(request.Repository) || len(request.Argv) == 0 || request.Argv[0] == "" {
		return false
	}
	for _, argument := range request.Argv {
		if strings.ContainsRune(argument, '\x00') {
			return false
		}
	}
	return true
}

func canonicalRepository(repository string) bool {
	parts := strings.Split(repository, "/")
	return len(parts) == 2 && canonicalRepositoryPart(parts[0]) && canonicalRepositoryPart(parts[1])
}

func canonicalRepositoryPart(part string) bool {
	if part == "" || part == "." || part == ".." || !lowerAlphaNumeric(part[0]) {
		return false
	}
	for index := 1; index < len(part); index++ {
		char := part[index]
		if !lowerAlphaNumeric(char) && char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func lowerAlphaNumeric(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

func validFixedPath(path string) bool {
	return path != "" && !strings.ContainsRune(path, '\x00') && filepath.IsAbs(path) && filepath.Clean(path) == path
}

func directChild(parent, child string) bool {
	return validFixedPath(child) && filepath.Dir(child) == parent && child != parent
}

func validLoopbackEndpoint(endpoint string) bool {
	if !strings.HasPrefix(endpoint, "http://127.0.0.1:") {
		return false
	}
	port := strings.TrimPrefix(endpoint, "http://127.0.0.1:")
	if strings.ContainsAny(port, "/?#\x00") {
		return false
	}
	if port == "" || len(port) > 5 || len(port) > 1 && port[0] == '0' {
		return false
	}
	value := 0
	for _, char := range port {
		if char < '0' || char > '9' {
			return false
		}
		value = value*10 + int(char-'0')
	}
	return value >= 1 && value <= 65535
}

func validDependencies(deps dependencies) bool {
	return deps.control.proxyCA != nil && deps.control.issueGitHub != nil && deps.control.issueOpenAI != nil && deps.control.revoke != nil &&
		deps.newBridge != nil && deps.newProcess != nil && deps.mkdirTemp != nil && deps.chmod != nil && deps.openCA != nil && deps.lstat != nil && deps.removeAll != nil
}

func itoa(value int) string { return string(rune('0' + value)) }
