package socketactivation

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
)

func validTestRules() Rules {
	return Rules{RuntimeDir: "/run/dev-agent-harness", BrokerUID: 1001, AgentGID: 1002}
}

func TestNewCopiesFixedRulesAndRejectsInvalidValues(t *testing.T) {
	r := validTestRules()
	receiver, err := New(r)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if receiver.runtimeDir != r.RuntimeDir || receiver.socketPath != r.RuntimeDir+"/egress.sock" || receiver.brokerUID != r.BrokerUID || receiver.agentGID != r.AgentGID {
		t.Fatalf("receiver=%#v", receiver)
	}
	for _, tc := range []Rules{
		{RuntimeDir: "run/dev-agent-harness", BrokerUID: 1001, AgentGID: 1002},
		{RuntimeDir: "/run/../run/dev-agent-harness", BrokerUID: 1001, AgentGID: 1002},
		{RuntimeDir: "/run/dev-agent-harness", BrokerUID: 0, AgentGID: 1002},
		{RuntimeDir: "/run/dev-agent-harness", BrokerUID: 1001, AgentGID: 0},
		{RuntimeDir: "/", BrokerUID: 1001, AgentGID: 1002},
		{RuntimeDir: "/run/dev-agent-harness\x00bad", BrokerUID: 1001, AgentGID: 1002},
	} {
		if got, err := New(tc); got != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("New(%#v)=%#v,%v", tc, got, err)
		}
	}
}

func TestTakeRejectsCorruptReceiverAndKeepsDiagnosticsFixed(t *testing.T) {
	for _, receiver := range []*Receiver{nil, {}, {runtimeDir: "/run/dev-agent-harness", socketPath: "/tmp/wrong", brokerUID: 1001, agentGID: 1002}} {
		if listener, err := receiver.Take(); listener != nil || !errors.Is(err, ErrDenied) {
			t.Fatalf("Take(%#v)=%#v,%v", receiver, listener, err)
		}
	}
	var receiver *Receiver
	if got := fmt.Sprintf("%v|%+v|%#v|%s", receiver, receiver, receiver, receiver); strings.Contains(got, "/run") || strings.Contains(got, "1001") {
		t.Fatalf("diagnostic leaked values: %q", got)
	}
	valid, err := New(validTestRules())
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%v|%+v|%#v", valid, valid, valid); got != "socketactivation.Receiver|socketactivation.Receiver|socketactivation.Receiver" {
		t.Fatalf("format=%q", got)
	}
}

func TestTakeClearsRecognizedActivationEnvironmentAndIsOneShot(t *testing.T) {
	oldLookup, oldUnset, oldPID, oldNew, oldConvert := lookupEnv, unsetEnv, currentPID, newActivationFile, convertFileListener
	t.Cleanup(func() {
		lookupEnv, unsetEnv, currentPID, newActivationFile, convertFileListener = oldLookup, oldUnset, oldPID, oldNew, oldConvert
	})
	values := map[string]string{listenPID: "bad", listenFDS: "2", listenFDNames: "wrong"}
	cleared := make([]string, 0, 3)
	lookupEnv = func(key string) (string, bool) { value, ok := values[key]; return value, ok }
	unsetEnv = func(key string) error { cleared = append(cleared, key); delete(values, key); return nil }
	currentPID = func() int { return 7 }
	receiver, err := New(validTestRules())
	if err != nil {
		t.Fatal(err)
	}
	if listener, err := receiver.Take(); listener != nil || !errors.Is(err, ErrDenied) {
		t.Fatalf("malformed Take=%v,%v", listener, err)
	}
	if got := strings.Join(cleared, ","); got != listenPID+","+listenFDS+","+listenFDNames {
		t.Fatalf("cleared=%q", got)
	}
	if listener, err := receiver.Take(); listener != nil || !errors.Is(err, ErrDenied) {
		t.Fatalf("second Take=%v,%v", listener, err)
	}
}

func TestTakeRejectsMissingEnvironmentBeforeOpeningFD(t *testing.T) {
	oldLookup, oldNew := lookupEnv, newActivationFile
	t.Cleanup(func() { lookupEnv, newActivationFile = oldLookup, oldNew })
	lookupEnv = func(string) (string, bool) { return "", false }
	opened := false
	newActivationFile = func(uintptr) *os.File { opened = true; return os.NewFile(3, "test") }
	receiver, err := New(validTestRules())
	if err != nil {
		t.Fatal(err)
	}
	if listener, err := receiver.Take(); listener != nil || !errors.Is(err, ErrDenied) || opened {
		t.Fatalf("Take=%v,%v opened=%v", listener, err, opened)
	}
}

func TestTakeRejectsNonCanonicalActivationTable(t *testing.T) {
	oldLookup, oldUnset, oldPID, oldRaw, oldNew := lookupEnv, unsetEnv, currentPID, rawEnvironment, newActivationFile
	t.Cleanup(func() {
		lookupEnv, unsetEnv, currentPID, rawEnvironment, newActivationFile = oldLookup, oldUnset, oldPID, oldRaw, oldNew
	})
	currentPID = func() int { return 7 }
	unsetEnv = func(string) error { return nil }
	rawEnvironment = func() ([]string, bool) { return nil, true }
	base := map[string]string{listenPID: "7", listenFDS: "1", listenFDNames: fdName}
	for _, tc := range []struct {
		name   string
		mutate func(map[string]string)
	}{
		{"missing-pid", func(values map[string]string) { delete(values, listenPID) }},
		{"missing-fds", func(values map[string]string) { delete(values, listenFDS) }},
		{"missing-names", func(values map[string]string) { delete(values, listenFDNames) }},
		{"pid-leading-zero", func(values map[string]string) { values[listenPID] = "07" }},
		{"fds-leading-zero", func(values map[string]string) { values[listenFDS] = "01" }},
		{"names-extra-token", func(values map[string]string) { values[listenFDNames] = "egress extra" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := make(map[string]string, len(base))
			for key, value := range base {
				values[key] = value
			}
			tc.mutate(values)
			lookupEnv = func(key string) (string, bool) { value, ok := values[key]; return value, ok }
			opened := 0
			newActivationFile = func(uintptr) *os.File { opened++; return spareTestFile(t) }
			receiver, err := New(validTestRules())
			if err != nil {
				t.Fatal(err)
			}
			if listener, err := receiver.Take(); listener != nil || !errors.Is(err, ErrDenied) || opened != 0 {
				t.Fatalf("Take=%v,%v opened=%d", listener, err, opened)
			}
		})
	}
}

func TestTakeRejectsDuplicateRawActivationEntries(t *testing.T) {
	oldLookup, oldUnset, oldPID, oldRaw, oldNew := lookupEnv, unsetEnv, currentPID, rawEnvironment, newActivationFile
	t.Cleanup(func() {
		lookupEnv, unsetEnv, currentPID, rawEnvironment, newActivationFile = oldLookup, oldUnset, oldPID, oldRaw, oldNew
	})
	lookupEnv = canonicalEnvironment
	unsetEnv = func(string) error { return nil }
	currentPID = func() int { return 7 }
	rawEnvironment = func() ([]string, bool) {
		return []string{"LISTEN_PID=7", "LISTEN_PID=7", "LISTEN_FDS=1", "LISTEN_FDNAMES=egress"}, true
	}
	opened := 0
	newActivationFile = func(uintptr) *os.File { opened++; return spareTestFile(t) }
	receiver, err := New(validTestRules())
	if err != nil {
		t.Fatal(err)
	}
	if listener, err := receiver.Take(); listener != nil || !errors.Is(err, ErrDenied) || opened != 0 {
		t.Fatalf("duplicate Take=%v,%v opened=%d", listener, err, opened)
	}
}

func TestTakeRejectsNonUnixListenerAndClosesIt(t *testing.T) {
	oldLookup, oldUnset, oldPID, oldNew, oldConvert := lookupEnv, unsetEnv, currentPID, newActivationFile, convertFileListener
	t.Cleanup(func() {
		lookupEnv, unsetEnv, currentPID, newActivationFile, convertFileListener = oldLookup, oldUnset, oldPID, oldNew, oldConvert
	})
	lookupEnv = func(key string) (string, bool) {
		switch key {
		case listenPID:
			return "7", true
		case listenFDS:
			return "1", true
		case listenFDNames:
			return fdName, true
		default:
			return "", false
		}
	}
	unsetEnv = func(string) error { return nil }
	currentPID = func() int { return 7 }
	activationCalls := 0
	newActivationFile = func(uintptr) *os.File { activationCalls++; return spareTestFile(t) }
	fake := &fakeListener{}
	conversionCalls := 0
	convertFileListener = func(*os.File) (net.Listener, error) { conversionCalls++; return fake, nil }
	receiver, err := New(validTestRules())
	if err != nil {
		t.Fatal(err)
	}
	if listener, err := receiver.Take(); listener != nil || !errors.Is(err, ErrDenied) || !fake.closed || conversionCalls != 1 || activationCalls != 1 {
		t.Fatalf("Take=%v,%v closed=%v conversions=%d activations=%d", listener, err, fake.closed, conversionCalls, activationCalls)
	}
}

func TestTakeClosesOriginalAfterConversionAndOnCloseFailure(t *testing.T) {
	oldLookup, oldUnset, oldPID, oldNew, oldConvert := lookupEnv, unsetEnv, currentPID, newActivationFile, convertFileListener
	t.Cleanup(func() {
		lookupEnv, unsetEnv, currentPID, newActivationFile, convertFileListener = oldLookup, oldUnset, oldPID, oldNew, oldConvert
	})
	lookupEnv = canonicalEnvironment
	unsetEnv = func(string) error { return nil }
	currentPID = func() int { return 7 }
	fake := &fakeListener{}
	conversionCalls := 0
	activationCalls := 0
	var original *os.File
	newActivationFile = func(uintptr) *os.File {
		activationCalls++
		read, _, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		original = read
		return read
	}
	convertFileListener = func(*os.File) (net.Listener, error) { conversionCalls++; return fake, nil }
	receiver, err := New(validTestRules())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.Take(); !errors.Is(err, ErrDenied) || !fake.closed || conversionCalls != 1 || activationCalls != 1 {
		t.Fatalf("conversion rejection err=%v closed=%v conversions=%d activations=%d", err, fake.closed, conversionCalls, activationCalls)
	}
	if _, err := original.Stat(); err == nil {
		t.Fatal("original descriptor remained open")
	}

	fake = &fakeListener{}
	conversionCalls = 0
	activationCalls = 0
	newActivationFile = func(uintptr) *os.File {
		activationCalls++
		file := os.NewFile(0, "invalid")
		_ = file.Close()
		return file
	}
	convertFileListener = func(*os.File) (net.Listener, error) { conversionCalls++; return fake, nil }
	if _, err := receiver.Take(); !errors.Is(err, ErrDenied) || !fake.closed || conversionCalls != 1 || activationCalls != 1 {
		t.Fatalf("close failure err=%v closed=%v conversions=%d activations=%d", err, fake.closed, conversionCalls, activationCalls)
	}
}

func TestTakeRejectsWrongUnixAddressAndDoesNotUnlink(t *testing.T) {
	oldLookup, oldUnset, oldPID, oldNew, oldConvert := lookupEnv, unsetEnv, currentPID, newActivationFile, convertFileListener
	t.Cleanup(func() {
		lookupEnv, unsetEnv, currentPID, newActivationFile, convertFileListener = oldLookup, oldUnset, oldPID, oldNew, oldConvert
	})
	lookupEnv = canonicalEnvironment
	unsetEnv = func(string) error { return nil }
	currentPID = func() int { return 7 }
	path := "socketactivation-wrong.sock"
	if err := os.WriteFile(path, []byte("sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	listener := &net.UnixListener{}
	activationCalls := 0
	newActivationFile = func(uintptr) *os.File { activationCalls++; return spareTestFile(t) }
	conversionCalls := 0
	convertFileListener = func(*os.File) (net.Listener, error) { conversionCalls++; return listener, nil }
	receiver, err := New(validTestRules())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := receiver.Take(); got != nil || !errors.Is(err, ErrDenied) || conversionCalls != 1 || activationCalls != 1 {
		t.Fatalf("wrong address Take=%v,%v conversions=%d activations=%d", got, err, conversionCalls, activationCalls)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("wrong socket was unlinked: %v", err)
	}
}

func canonicalEnvironment(key string) (string, bool) {
	switch key {
	case listenPID:
		return "7", true
	case listenFDS:
		return "1", true
	case listenFDNames:
		return fdName, true
	default:
		return "", false
	}
}

func spareTestFile(t *testing.T) *os.File {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = write.Close()
	return read
}

type fakeListener struct{ closed bool }

func (*fakeListener) Accept() (net.Conn, error) { return nil, errors.New("unused") }
func (*fakeListener) Addr() net.Addr            { return fakeAddr{} }
func (l *fakeListener) Close() error            { l.closed = true; return nil }

type fakeAddr struct{}

func (fakeAddr) Network() string { return "tcp" }
func (fakeAddr) String() string  { return "unused" }
